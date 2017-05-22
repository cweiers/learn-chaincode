// Escalator
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type SimpleChaincode struct {
}

type Ticket struct {
	TicketID        string
	Timestamp       string // time of ticket creation
	Trainstation    string
	Platform        string
	Device          string // the device in need of repairs (i.e. left upwards escalator)
	Status          string // current ticket status (not repair status), i.e. "OPEN". TODO: rework as some sort of enum to limit input.
	TechPart        string // representing the defective part of the escalator
	ErrorID         string
	ErrorMessage    string
	ServiceProvider string // the assigned service provider thats commissioned to do the repairs
	SpEmployee      string // repairman assigned by service_provider
	SpeCommentary   string // additional commentary, optionally to be filled out by the sp_employee
	EstRepairTime   string
	RepairStatus    string
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	err := stub.PutState("counter", []byte("0"))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

//Invoke is the entry point for all other asset altering functions called by an CC invocation
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	switch function {
	case "createTicket":
		return t.createTicket(stub, args)
	case "assignTicket":
		return t.assignTicket(stub, args)
	case "acceptTicket":
		return t.acceptTicket(stub, args)
	case "assignMechanic":
		return t.assignMechanic(stub, args)
	case "startJourney":
		return t.startJourney(stub, args)
	case "onArrival":
		return t.onArrival(stub, args)
	case "startRepair":
		return t.startRepair(stub, args)
	case "finishRepair":
		return t.finishRepair(stub, args)
	case "createDefaultTicket":
		return t.createDefaultTicket(stub, args)
	}

	return nil, errors.New("Received unknown function invocation: " + function)

}
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if function == "getFullTicket" {
		return t.getFullTicket(stub, args)
	}
	if function == "getCounter" {
		return t.getCounter(stub, args)
	}
	fmt.Println("query did not find func: " + function)

	return nil, errors.New("Received unknown function query")
}

func (t *SimpleChaincode) getCounter(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	counterAsByteArr, err := stub.GetState("counter")
	if err != nil {
		return nil, errors.New("Query failure for getCounter")
	}
	return counterAsByteArr, nil
}

func (t *SimpleChaincode) getFullTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments. Must be (1): TicketID")
	}

	//	var ticketAsByteArr []byte

	ticketAsByteArr, err := stub.GetState(args[0])

	if err != nil {
		return nil, errors.New("Query failure for getFullTicket")
	}

	return ticketAsByteArr, nil
}

// Create a new ticket and store it on the ledger with ticket_id as key.
// args should have length 4: The fields TicketID, Timestamp, Device, TechPart, and ErrorID have to be initialised
//
func (t *SimpleChaincode) createTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 8 {
		return nil, errors.New("Wrong number of arguments, must be 7: Timestamp, Trainstation, Platform, Device, TechPart, ErrorID and ErrorMessage")
	}

	idAsBytes, _ := stub.GetState("counter") // get highest current ticket id number from worldstate, increment, and set as
	str := string(idAsBytes[:])              //	TicketID for newly created Ticket & update highest running ticket number.
	idAsInt, _ := strconv.Atoi(str)          // TODO This seems unnecessarily complicated.
	idAsInt++                                //
	idAsString := strconv.Itoa(idAsInt)
	err := stub.PutState("counter", []byte(idAsString))

	var ticket = Ticket{
		TicketID:     idAsString,
		Timestamp:    args[0],
		Trainstation: args[1],
		Platform:     args[2],
		Device:       args[3],
		Status:       "EINGETROFFEN",
		TechPart:     args[4],
		ErrorID:      args[5],
		ErrorMessage: args[6],
	}

	state, _ := json.Marshal(ticket)

	stub.PutState(args[0], []byte(state))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// Creates a default ticket. Maybe change to just call createTicket with arguments.
//
func (t *SimpleChaincode) createDefaultTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	idAsBytes, _ := stub.GetState("counter") // get highest current ticket id number from worldstate, increment, and set as
	str := string(idAsBytes[:])              //	TicketID for newly created Ticket & update highest running ticket number.
	idAsInt, _ := strconv.Atoi(str)          // TODO This seems unnecessarily complicated.
	idAsInt++                                //
	idAsString := strconv.Itoa(idAsInt)
	err := stub.PutState("counter", []byte(idAsString))

	var ticket = Ticket{
		TicketID:     idAsString,
		Timestamp:    "2017-12-03T12:35:00.000Z",
		Trainstation: "Bonn Hbf",
		Platform:     "Gleis 5",
		Device:       "Rolltreppe nach oben",
		Status:       "Eingetroffen",
		TechPart:     "Motor RTM-X 64",
		ErrorID:      "#2356-102",
		ErrorMessage: "Totalausfall",
	}
	state, _ := json.Marshal(ticket)

	stub.PutState(args[0], []byte(state))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// Assign an existing Ticket to a ServiceProvider. Arguments should be TicketID and the name of the serviceprovider that the ticket gets assigned to.
func (t *SimpleChaincode) assignTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Wrong number of arguments, must be 2: TicketID and ServiceProvider")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0]) //get ticket from world state as byte array
	if err != nil {
		return nil, err
	}
	ticket := new(Ticket)
	json.Unmarshal(state, ticket)    // translate back to struct (well, to "pointer to struct" actually
	ticket.ServiceProvider = args[1] //set new  ServiceProvider
	ticket.Status = "ZUGEWIESEN"     //update status to "assigned"
	ticket.RepairStatus = "Wird geprueft"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

func (t *SimpleChaincode) acceptTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments, must be 1: TicketID ")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.Status = "ZUGEWIESEN"
	ticket.RepairStatus = "Auftrag angenommen"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

func (t *SimpleChaincode) assignMechanic(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Wrong number of arguments, must be 2: TicketID and SpEmployee ")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.SpEmployee = args[1]
	ticket.RepairStatus = "Ticket erhalten"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

func (t *SimpleChaincode) startJourney(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments, must be 1: TicketID")
	}

	var state []byte
	var err error
	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.RepairStatus = "Techniker in Anfahrt"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

func (t *SimpleChaincode) onArrival(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Wrong number of arguments, must be 2: TicketID,EstRepairTime and SpeCommentary ")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.SpeCommentary = args[1]
	ticket.EstRepairTime = args[2]
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}
func (t *SimpleChaincode) startRepair(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments, must be 1: TicketID")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.RepairStatus = "Reparatur begonnen"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

func (t *SimpleChaincode) finishRepair(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments, must be 1: TicketID")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.RepairStatus = "Reparatur abgeschlossen"
	ticket.Status = "ERLEDIGT"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}
