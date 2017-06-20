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

//simple SLA.
type ServiceLevelAgreement struct {
	ServiceProvider string
	TimeToArrive    int64 // contractually agreed time for a mechanic to arrive
	TimeToRepair    int64 // contractually agreed time for the repairs to be finished
	// representing the number of times the agreement was fulfilled, slightly violated or severely violated
	None   int //no violation
	Light  int
	Severe int
}

type Escalator struct {
	EscalatorID  string
	Trainstation string
	Platform     string
	IsWorking    bool
}

type Ticket struct {
	TicketID        string
	Timestamp       int64 // time of ticket creation
	Trainstation    string
	Platform        string
	Device          string // the device in need of repairs (Some form of identifier for the escalator)
	Status          string // current ticket status (not repair status), i.e. "OPEN". TODO: rework as some sort of enum to limit input.
	TechPart        string // representing the defective part of the escalator
	ErrorID         string
	ErrorMessage    string
	ServiceProvider string // the assigned service provider thats commissioned to do the repairs
	SpEmployee      string // mechanic assigned by ServiceProvider
	SpeCommentary   string // additional commentary, optionally to be filled out by the SpEmployee
	EstRepairTime   string
	TimeOfArrival   int64 // time of arrival
	RepairStatus    string
	FinalRepairTime int64 // closing the ticket
	FinalReport     string
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	//initialise counters for ticket and escalator ID creation
	stub.PutState("escalatorCounter", []byte("0"))
	stub.PutState("ticketCounter", []byte("0"))

	//create an escalator to use with createDefaultTicket
	idAsString, _ := createID(stub, "escalator")
	idAsString = "DO" + idAsString
	var escalator = Escalator{
		EscalatorID:  idAsString,
		Trainstation: "Dortmung Hbf",
		Platform:     "Gleis 4",
	}

	state, _ := json.Marshal(escalator)

	stub.PutState(idAsString, state)

	return nil, nil
}

//Invoke is the entry point for all other asset altering functions called by an CC invocation
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	switch function {
	case "setEscalatorState":
		return t.setEscalatorState(stub, args)
	case "createSLA":
		return t.createSLA(stub, args)
	case "createEscalator":
		return t.createEscalator(stub, args)
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

//Query is the entry point for all read-only operations
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	switch function {
	case "getEscalatorState":
		return t.getEscalatorState(stub, args)
	case "getSLA":
		return t.getSLA(stub, args)
	case "getFullTicket":
		return t.getFullTicket(stub, args)
	case "getTicketCounter":
		return t.getTicketCounter(stub, args)
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
	case "getNewSPTickets":
		return t.getNewSPTickets(stub, args)
	}
	fmt.Println("query did not find func: " + function)
	return nil, errors.New("Received unknown function query")
}

//..............................................
//............INVOKE FUNCTIONS..................
//..............................................

// create a service level agreement for a given ServiceProvider. Input should be the ServiceProvider, the time in seconds from ticket creation
// until arrival of a mechanic, and the time in seconds from ticket creation until the escalator repair is done.
func (t *SimpleChaincode) createSLA(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var sla ServiceLevelAgreement
	sla.ServiceProvider = args[0]
	sla.TimeToArrive, _ = strconv.ParseInt(args[1], 10, 64)
	sla.TimeToRepair, _ = strconv.ParseInt(args[2], 10, 64)
	sla.None = 0
	sla.Light = 0
	sla.Severe = 0
	slaKey := "sla" + strings.ToLower(args[0])

	slaAsByteArr, err := json.Marshal(sla)

	if err != nil {
		return nil, err
	}
	stub.PutState(slaKey, slaAsByteArr)
	return slaAsByteArr, nil
}

//update a SLA from the world state with new values. A value for both timeToArrive and timeToRepair has to be supplied.
func (t *SimpleChaincode) updateSLA(stub shim.ChaincodeStubInterface, args []string) error {
	if len(args) != 3 {
		return errors.New("Needs name of the service provider, timeToArrive and timeToRepair. If one of the latter two does not change, the old value still has to be supplied")
	}

	var sla ServiceLevelAgreement
	slaAsByteArr, err := stub.GetState("sla" + strings.ToLower(args[0]))
	if err != nil {
		return err
	}
	json.Unmarshal(slaAsByteArr, &sla)
	sla.TimeToArrive, _ = strconv.ParseInt(args[1], 10, 64)
	sla.TimeToRepair, _ = strconv.ParseInt(args[2], 10, 64)

	slaAsByteArr, _ = json.Marshal(sla)
	stub.PutState("sla"+strings.ToLower(args[0]), slaAsByteArr)
	return nil
}

//Takes either EscalatorID and "true" OR EscalatorID, "false", and 3 more : TechPart, ErrorID, and ErrorMsg
func (t *SimpleChaincode) setEscalatorState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var esc Escalator
	escAsByteArr, err := stub.GetState(args[0])
	if err != nil {
		return nil, err
	}
	json.Unmarshal(escAsByteArr, &esc)
	escState, _ := strconv.ParseBool(args[1])
	if len(args) == 2 && escState == true {
		esc.IsWorking = true
		escAsByteArr, _ = json.Marshal(esc)
		stub.PutState(args[0], escAsByteArr)
		return nil, nil
	}
	if len(args) == 5 && escState == false {
		esc.IsWorking = false
		escAsByteArr, _ = json.Marshal(esc)
		stub.PutState(args[0], escAsByteArr)
		return t.createTicket(esc.Trainstation, esc.Platform, args[0], args[2], args[3], args[4])
	}
	return nil, errors.New("Failed to properly set escalator status. Wrong number of arguments ?")
}

// Create a new ticket and store it on the ledger with TicketID as key.
//
func (t *SimpleChaincode) createTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 6 {
		return nil, errors.New("Wrong number of arguments, must be 6: Trainstation, Platform, Device, TechPart, ErrorID and ErrorMessage")
	}
	idAsString, _ := createID(stub, "ticket")
	time := getTransactionTime(stub)
	var ticket = Ticket{
		TicketID:     idAsString,
		Timestamp:    time,
		Trainstation: args[0],
		Platform:     args[1],
		Device:       args[2],
		Status:       "EINGETROFFEN",
		TechPart:     args[3],
		ErrorID:      args[4],
		ErrorMessage: args[5],
	}

	state, err := json.Marshal(ticket)

	stub.PutState(idAsString, state)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// Creates a default ticket.
//
func (t *SimpleChaincode) createDefaultTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var defaultEsc Escalator
	defaultEscAsByteArr, _ := stub.GetState("DO0001")

	json.Unmarshal(defaultEscAsByteArr, &defaultEsc)

	idAsString, _ := createID(stub, "ticket")
	time := getTransactionTime(stub)

	var ticket = Ticket{
		TicketID:     idAsString,
		Timestamp:    time,
		Trainstation: defaultEsc.Trainstation,
		Platform:     defaultEsc.Platform,
		Device:       defaultEsc.EscalatorID,
		Status:       "Eingetroffen",
		TechPart:     "Motor RTM-X 64",
		ErrorID:      "#2356-102",
		ErrorMessage: "Totalausfall",
	}
	state, err := json.Marshal(ticket)

	stub.PutState(idAsString, state)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

//takes Trainstation and Platform as input
func (t *SimpleChaincode) createEscalator(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Wrong number of arguments, must be 2: Trainstation and Platform")
	}

	idAsString, _ := createID(stub, "escalator")
	idAsString = strings.ToUpper(args[0][0:2]) + idAsString //Id is now the first two characters of the location + a sequential ID
	var escalator = Escalator{
		EscalatorID:  idAsString,
		Trainstation: args[0],
		Platform:     args[1],
	}

	state, err := json.Marshal(escalator)

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
	json.Unmarshal(state, &ticket)   // translate back to struct
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
	json.Unmarshal(state, &ticket)
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
		return nil, errors.New("Wrong number of arguments, must be 2: TicketID,SpeCommentary and EstRepairTime")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, &ticket)
	ticket.TimeOfArrival = getTransactionTime(stub)
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
	json.Unmarshal(state, &ticket)
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

	state, err := stub.GetState(args[0])
	if err != nil {
		return nil, err
	}

	ticket := new(Ticket)
	json.Unmarshal(state, &ticket)
	ticket.FinalRepairTime = getTransactionTime(stub)
	ticket.RepairStatus = "Reparatur abgeschlossen"
	ticket.Status = "ERLEDIGT"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	//update SLA depending on timestamps
	var sla ServiceLevelAgreement
	slaAsByteArr, _ := stub.GetState("sla" + strings.ToLower(ticket.ServiceProvider))
	json.Unmarshal(slaAsByteArr, &sla)

	ttA := ticket.TimeOfArrival - ticket.Timestamp   //time to arrive
	ttR := ticket.FinalRepairTime - ticket.Timestamp //time to repair
	switch {
	case (ttA < sla.TimeToArrive) && (ttR < sla.TimeToRepair): //All good
		sla.None += 1
	case (ttA > sla.TimeToArrive+10800) || (ttR > sla.TimeToRepair+14400): //mechanic arrived more than 10800s = 3hours late OR it took more than 4 hours longer to repair overall
		sla.Severe += 1
	default:
		sla.Light += 1
	}

	slaAsByteArr, _ = json.Marshal(sla)
	stub.PutState("sla"+strings.ToLower(ticket.ServiceProvider), slaAsByteArr)
	return nil, nil
}

func (t *SimpleChaincode) writeFinalReport(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.New("Wrong number of arguments, must be 2: TicketID and final commentary")
	}

	var state []byte
	var err error

	state, err = stub.GetState(args[0])
	if err != nil {
		return nil, err
	}
	ticket := new(Ticket)
	json.Unmarshal(state, &ticket)
	ticket.FinalReport = args[1]
	ticket.RepairStatus = "Im Abschluss"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

//..............................................
//............QUERY FUNCTIONS..................
//..............................................

//Input should be the name of the serviceprovider
func (t *SimpleChaincode) getEscalatorState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	escAsByteArr, err := stub.GetState(args[0])
	if err != nil {
		return nil, err
	}
	var esc Escalator
	json.Unmarshal(escAsByteArr, &esc)
	var byteArr []byte
	return strconv.AppendBool(byteArr, esc.IsWorking), nil

}

func (t *SimpleChaincode) getSLA(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	slaAsByteArr, err := stub.GetState("sla" + strings.ToLower(args[0]))
	if err != nil {
		return nil, err
	}
	return slaAsByteArr, nil
}

func (t *SimpleChaincode) getTicketCounter(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	ticketCounterAsByteArr, err := stub.GetState("ticketCounter")
	if err != nil {
		return nil, err
	}
	return ticketCounterAsByteArr, nil
}

func (t *SimpleChaincode) getFullTicket(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Wrong number of arguments. Must be (1): TicketID")
	}

	ticketAsByteArr, err := stub.GetState(args[0])

	if err != nil {
		return nil, errors.New("Query failure for getFullTicket")
	}

	return ticketAsByteArr, nil
}

func (t *SimpleChaincode) getTicketsByServiceProvider(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	//construct iterator
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
			//			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			//			buffer.WriteString("}")
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
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
				//				buffer.WriteString("{")
				buffer.WriteString(string(queryResultValue))
				//				buffer.WriteString("}")
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
				//				buffer.WriteString("{")
				buffer.WriteString(string(queryResultValue))
				//				buffer.WriteString("}")
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
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
			//			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			//			buffer.WriteString("}")
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
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
			//			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			//			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

func (t *SimpleChaincode) getNewSPTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
		if strings.EqualFold(tempTicket.ServiceProvider, args[0]) && strings.EqualFold(tempTicket.RepairStatus, "Wird geprueft") {

			// Add a comma before array members, suppress it for the first array member
			if bArrayMemberAlreadyWritten == true {
				buffer.WriteString(",")
			}
			//			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			//			buffer.WriteString("}")
			bArrayMemberAlreadyWritten = true
		}

	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// Returns Tickets for a given ServiceProvider that have been assigned to a Mechanic that has not yet had a look at the broken device
func (t *SimpleChaincode) getAssignedSPTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	//construct iterator
	startKey := "0001"
	MaxIdAsBytes, _ := stub.GetState("ticketCounter")
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
			//			buffer.WriteString("{")
			buffer.WriteString(string(queryResultValue))
			//			buffer.WriteString("}")
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

		//		buffer.WriteString("{")
		buffer.WriteString(string(queryResultValue))
		//		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return buffer.Bytes(), nil
}

func (t *SimpleChaincode) getAllTickets(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	ticketCounterAsByteArr, err := stub.GetState("ticketCounter")
	s := string(ticketCounterAsByteArr[:])
	if err != nil {
		return nil, err
	}

	return t.getTicketsByRange(stub, []string{"0001", s})
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
	json.Unmarshal(state, &ticket)
	ticket.SpEmployee = args[1]
	ticket.RepairStatus = "Ticket erhalten"
	state, err = json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	stub.PutState(args[0], state) //write updated ticket to world state again

	return nil, nil
}

//..............................................
//............INTERNAL UTILITY FUNCTIONS........
//..............................................

func getTransactionTime(stub shim.ChaincodeStubInterface) int64 {
	timePointer, _ := stub.GetTxTimestamp()
	return timePointer.Seconds

}

func leftPad2Len(s string, padStr string, overallLen int) string {
	var padCountInt int
	padCountInt = 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = strings.Repeat(padStr, padCountInt) + s
	return retStr[(len(retStr) - overallLen):]
}

func getEscalatorAsByteArr(stub shim.ChaincodeStubInterface, escalatorID string) ([]byte, error) {
	return stub.GetState(escalatorID)
}

//creates a sequential ID for either a new Ticket or a new Escalator. structname should be "ticket" or "escalator" respectively
func createID(stub shim.ChaincodeStubInterface, structName string) (string, error) {

	var idAsBytes []byte
	switch structName {
	case "ticket":
		idAsBytes, _ = stub.GetState("ticketCounter")
	case "escalator":
		idAsBytes, _ = stub.GetState("escalatorCounter")
	default:
		return "", errors.New("ID creation not supported for input string: Must be ticketCounter or escalatorCounter")
	}

	// get highest current ticket id number from worldstate, increment and set as
	str := string(idAsBytes[:])     //	TicketID for newly created Ticket & update highest running ticket number.
	idAsInt, _ := strconv.Atoi(str) //
	idAsInt++                       // TODO This seems unnecessarily complicated.
	idAsString := strconv.Itoa(idAsInt)
	idAsString = leftPad2Len(idAsString, "0", 4)

	stub.PutState(structName+"Counter", []byte(idAsString))
	return idAsString, nil
}
