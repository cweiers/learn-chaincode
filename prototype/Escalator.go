// Escalator
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	FinalRepairTime string
	FinalReport     string
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
	case "createDefaultTicket":
		return t.createDefaultTicket(stub, args)
	case "assignTicket":
		return t.assignTicket(stub, args)
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
	case "writeFinalReport":
		return t.writeFinalReport(stub, args)

	}

	return nil, errors.New("Received unknown function invocation: " + function)

}
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	switch function {
	case "getFullTicket":
		return t.getFullTicket(stub, args)
	case "getCounter":
		return t.getCounter(stub, args)
	case "getTicketsByRange":
		return t.getTicketsByRange(stub, args)
	case "getAllTickets":
		return t.getAllTickets(stub, args)
	case "getTicketsByStatus":
		return t.getTicketsByStatus(stub, args)
	case "getTicketsByServiceProvider":
		return t.getTicketsByServiceProvider(stub, args)
	case "getTicketsByMechanic":
		return t.getTicketsByMechanic(stub, args)
	case "getAssignedSPTickets":
		return t.getAssignedSPTickets(stub, args)
	case "getWIPTickets":
		return t.getWIPTickets(stub, args)
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

func (t *SimpleChaincode) getTicketsByServiceProvider(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	//construct iterator
	startKey := "1"
	MaxIdAsBytes, _ := stub.GetState("counter")
	endKey := string(MaxIdAsBytes[:])

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	var tempTicket Ticket //place to unmarshall our Tickets in []byte-Form into.

	for resultsIterator.HasNext() {
		_, queryResultValue, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		json.Unmarshal(queryResultValue, &tempTicket)

		// check if ticket has given ServiceProvider
		if strings.EqualFold(tempTicket.ServiceProvider, args[0]) {

			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

//returns a collection of Tickets with a given Status. Expects either "EINGETROFFEN", "ZUGEWIESEN", or "ERLEDIGT" as first input argument.
//OPTIONAL : Add ServiceProvider String as 2nd argument.
func (t *SimpleChaincode) getTicketsByStatus(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	//construct iterator
	startKey := "1"
	MaxIdAsBytes, _ := stub.GetState("counter")
	endKey := string(MaxIdAsBytes[:])

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	var tempTicket Ticket //place to unmarshall our Tickets in []byte-Form into.

	if len(args) == 2 {
		for resultsIterator.HasNext() {
			_, queryResultValue, err := resultsIterator.Next()
			if err != nil {
				return nil, err
			}

			json.Unmarshal(queryResultValue, &tempTicket)

			// check if ticket has given Status AND ServiceProvider
			if strings.EqualFold(tempTicket.Status, args[0]) && strings.EqualFold(tempTicket.ServiceProvider, args[1]) {

				// Add a comma before array members, suppress it for the first array member
				if bArrayMemberAlreadyWritten == true {
					buffer.WriteString(",")
				}
				buffer.WriteString("{")
				buffer.WriteString(string(queryResultValue))
				buffer.WriteString("}")
				bArrayMemberAlreadyWritten = true
			}
		}
		buffer.WriteString("]")
	}
	if len(args) == 1 {
		for resultsIterator.HasNext() {
			_, queryResultValue, err := resultsIterator.Next()
			if err != nil {
				return nil, err
			}

			json.Unmarshal(queryResultValue, &tempTicket)

			// check if ticket has given Status
			if strings.EqualFold(tempTicket.Status, args[0]) {

				// Add a comma before array members, suppress it for the first array member
				if bArrayMemberAlreadyWritten == true {
					buffer.WriteString(",")
				}
				buffer.WriteString("{")
				buffer.WriteString(string(queryResultValue))
				buffer.WriteString("}")
				bArrayMemberAlreadyWritten = true
			}
		}
		buffer.WriteString("]")
	}

	return buffer.Bytes(), nil
}

//returns a collection of tickets for a given SPEmployee and his/her Employer (the ServiceProvider)
func (t *SimpleChaincode) getTicketsByMechanic(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Wrong number of arguments, must be 2: ServiceProvider and SpEmployee ")
	}

	//construct iterator
	startKey := "1"
	MaxIdAsBytes, _ := stub.GetState("counter")
	endKey := string(MaxIdAsBytes[:])

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	var tempTicket Ticket //place to unmarshall our Tickets in []byte-Form into.

	for resultsIterator.HasNext() {
		_, queryResultValue, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		json.Unmarshal(queryResultValue, &tempTicket)

		// check if ticket has given ServiceProvider
		if strings.EqualFold(tempTicket.SpEmployee, args[1]) && strings.EqualFold(tempTicket.ServiceProvider, args[0]) {

			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// returns a collection of Tickets that belong to the "Work in Progress" column (RepairStatus = "Techniker vor Ort" or "Reparatur begonnen")
// Takes a ServiceProvider string as input.
func (t *SimpleChaincode) getWIPTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	//construct iterator
	startKey := "1"
	MaxIdAsBytes, _ := stub.GetState("counter")
	endKey := string(MaxIdAsBytes[:])

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	var tempTicket Ticket //place to unmarshall our Tickets in []byte-Form into.

	for resultsIterator.HasNext() {
		_, queryResultValue, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		json.Unmarshal(queryResultValue, &tempTicket)

		// check if ticket has given ServiceProvider
		if strings.EqualFold(tempTicket.ServiceProvider, args[0]) &&
			(strings.EqualFold(tempTicket.RepairStatus, "Reparatur begonnen") || strings.EqualFold(tempTicket.RepairStatus, "Techniker vor Ort")) {

			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// Returns Tickets for a given ServiceProvider that have been assigned to a Mechanic that has not yet had a look at the broken device
func (t *SimpleChaincode) getAssignedSPTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	//construct iterator
	startKey := "1"
	MaxIdAsBytes, _ := stub.GetState("counter")
	endKey := string(MaxIdAsBytes[:])

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	var tempTicket Ticket //place to unmarshall our Tickets in []byte-Form into.

	for resultsIterator.HasNext() {
		_, queryResultValue, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		json.Unmarshal(queryResultValue, &tempTicket)

		// check if ticket has given ServiceProvider
		if strings.EqualFold(tempTicket.ServiceProvider, args[0]) &&
			(strings.EqualFold(tempTicket.RepairStatus, "Techniker in Anfahrt") || strings.EqualFold(tempTicket.RepairStatus, "Ticket erhalten")) {

			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

func (t *SimpleChaincode) getTicketsByRange(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.RangeQueryState(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		_, queryResultValue, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}

		buffer.WriteString("{")
		//		buffer.WriteString("\"")
		//		buffer.WriteString(queryResultKey)
		//		buffer.WriteString("\"")

		//		buffer.WriteString(", \"Ticket\":")

		buffer.WriteString(string(queryResultValue))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return buffer.Bytes(), nil
}

func (t *SimpleChaincode) getAllTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	counterAsByteArr, err := stub.GetState("counter")
	s := string(counterAsByteArr[:])
	if err != nil {
		return nil, err
	}

	return t.getTicketsByRange(stub, []string{"1", s})
}

// Create a new ticket and store it on the ledger with ticket_id as key.
//
func (t *SimpleChaincode) createTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 6 {
		return nil, errors.New("Wrong number of arguments, must be 6: , Trainstation, Platform, Device, TechPart, ErrorID and ErrorMessage")
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

	stub.PutState(idAsString, state)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// Creates a default ticket. Maybe change to just call createTicket with arguments.
//
func (t *SimpleChaincode) createDefaultTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	idAsBytes, _ := stub.GetState("counter") // get highest current ticket id number from worldstate, increment and set as
	str := string(idAsBytes[:])              //	TicketID for newly created Ticket & update highest running ticket number.
	idAsInt, _ := strconv.Atoi(str)          //
	idAsInt++                                // TODO This seems unnecessarily complicated.
	idAsString := strconv.Itoa(idAsInt)
	err := stub.PutState("counter", []byte(idAsString))

	var ticket = Ticket{
		TicketID:     idAsString,
		Timestamp:    "2017-06-01T06:50:00.000Z",
		Trainstation: "Bonn Hbf",
		Platform:     "Gleis 5",
		Device:       "Rolltreppe nach oben",
		Status:       "Eingetroffen",
		TechPart:     "Motor RTM-X 64",
		ErrorID:      "#2356-102",
		ErrorMessage: "Totalausfall",
	}
	state, _ := json.Marshal(ticket)

	stub.PutState(idAsString, state)
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
	json.Unmarshal(state, ticket)    // translate back to struct (well, to "pointer to struct" actually)
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
	ticket.RepairStatus = "Techniker vor Ort"
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

func (t *SimpleChaincode) writeFinalReport(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Wrong number of arguments, must be 3: TicketID , final Repairtime and final commentary")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}
	ticket := new(Ticket)
	json.Unmarshal(state, ticket)
	ticket.FinalRepairTime = args[1]
	ticket.FinalReport = args[2]
	ticket.RepairStatus = "Im Abschluss"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil

}
