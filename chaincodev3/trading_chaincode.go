package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"sort"
)

// Needed for function pointer (t *SimpleChaincode)
// Leave empty
type SimpleChaincode struct {
}

var customersKey = "_customers"       // key for list of customers
var offersKey = "_offers"             // key for list of current offers
var transactionsKey = "_transactions" // key for list of transactions
var pendingTransactionKey = "_pendingtransaction" // key for tracking the pending transaction

// Transaction structure
type Transaction struct {
	TXID 	int64 			`json:"txid"`
	Offers	map[string]int 	`json:"offers"`
	Buyer	string			`json:"buyer"`
	Cost 	int				`json:"cost"`
	Energy 	int				`json:"energy"`
	Status 	string			`json:"status"`
}

// Query response structs, used to provide a predictable response structure
type QueryResponseInt struct {
	Success	bool	`json:"success"`
	Data	int		`json:"data"`
}

type QueryResponseMap struct {
	Success	bool			`json:"success"`
	Data	map[string]int	`json:"data"`
}

type QueryResponseString struct {
	Success	bool	`json:"success"`
	Data	string	`json:"data"`
}

type QueryResponseTransactions struct {
	Success	bool			`json:"success"`
	Data	[]Transaction	`json:"data"`
}

type QueryResponseBytes struct {
	Success	bool	`json:"success"`
	Data	[]byte	`json:"data"`
}

// Main function - runs on start
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting simple chaincode: %s", err)
	}
}

//////////////////////////////////////// CHAINCODE INTERFACE FUNCTIONS ////////////////////////////////////////

// Init - reset the state of the chaincode
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var initVal int
	var err error
	var retStr string

	// Check the number of args passed in
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: Initial value"
		return []byte(retStr), errors.New(retStr)
	}

	// Get initial value
	initVal, err = strconv.Atoi(args[0])
	if err != nil {
		retStr = "Incorrect number of arguments. Expecting 1: Initial value"
		return []byte(retStr), errors.New(retStr)
	}

	// Write initVal to the ledger
	// Use test var ece because reasons
	err = stub.PutState("ece", []byte(strconv.Itoa(initVal)))
	if err != nil {
		return nil, err
	}

	// Clear/initialize the list of customers
	// "owner" represents the owner of the charger and is always present
	emptyCustomers := make(map[string]int)
	emptyCustomers["owner"] = 0
	err = marshalAndPut(stub, customersKey, emptyCustomers)
	if err != nil {
		return nil, err
	}

	// Clear the list of offers
	emptyOffers := make(map[string]int)
	err = marshalAndPut(stub, offersKey, emptyOffers)
	if err != nil {
		return nil, err
	}

	// Clear the list of transactions
	var emptyTransactions []Transaction
	err = marshalAndPut(stub, transactionsKey, emptyTransactions)
	if err != nil {
		return nil, err
	}

	// Clear the pendingTransaction array
	var emptyPendingTransaction []Transaction
	err = marshalAndPut(stub, pendingTransactionKey, emptyPendingTransaction)
	if err != nil {
		return nil, err
	}

	// Successful init return
	retStr = "Chaincode state initialized successfully."
	return []byte(retStr), nil
}

// Invoke function - entry point for invocations
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Print debug message
	fmt.Println("Invoke() is running: " + function)

	// Handle the different possible function calls
	switch function {
	case "addOfferQuantity":
		return addOfferQuantity(stub, args)
	case "subtractOfferQuantity":
		return subtractOfferQuantity(stub, args)
	case "addCustomer":
		return addCustomer(stub, args)
	case "addCustomerFunds":
		return addCustomerFunds(stub, args)
	case "acceptOffer":
		return acceptOffer(stub, args)
	case "completeTransaction":
		return completeTransaction(stub)
	case "cancelTransaction":
		return cancelTransaction(stub, args)
	case "addTransaction":
		return addTransaction(stub, args)
	case "init":
		return t.Init(stub, "init", args)
	default:
		// Print error message if function not found
		fmt.Println("Invoke() did not find function: " + function)
		// Return error
		return []byte("Invoke() did not find function: " + function), errors.New("Received unknown function invocation: " + function)
	}

	return nil, nil

}

// Run function - entry point for invocations
// Older versions of Hyperledger used Run() instead of Invoke()
// Probably unnecessary, but it can't hurt to have
// Just pass arguments along to Invoke()
func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Print debug message
	fmt.Println("Run() is running: " + function)
	// Pass arguments to Invoke()
	return t.Invoke(stub, function, args)

}

// Query function - entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte,error) {

	// Debug message
	fmt.Println("Query() is running: " + function)

	// Handle the different types of query functions
	if function == "read" {
		return read(stub, args)
	} else if function == "getPendingTransaction" {
		return getPendingTransaction(stub)
	} else if function == "getOffers" {
		return getOffers(stub)
	} else if function == "getTransactions" {
		return getTransactions(stub)
	} else if function == "getCustomers" {
		return getCustomers(stub)
	} else if function == "getCustomer" {
		return getCustomer(stub, args)
	} else if function == "getTotalEnergyForSale" {
		return getTotalEnergyForSale(stub)
	}

	// Print message if query function not found
	fmt.Println("Query() did not find function: " + function)

	// Return an error
	return createQueryResponseString(false, "Query() did not find function: " + function)

}

//////////////////////////////////////// QUERY FUNCTIONS ////////////////////////////////////////

// Read function
// Read a variable from chaincode state
// Mainly used for debugging purposes
func read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	// Check to make sure number of arguments is correct
	if len(args) != 1 {
		return createQueryResponseString(false, "Incorrect number of arguments. Expecting 1: name of variable to query")
	}
	// Check variable name
	if len(args[0]) == 0 {
		return createQueryResponseString(false,	"First argument (variable name) cannot be an empty string")
	}

	// Debug message
	fmt.Println("Trying to read variable named " + args[0])

	// Get the variable from the chaincode state
	name := args[0]
	valAsBytes, err := stub.GetState(name)
	if err != nil {
		return createQueryResponseString(false,	"Could not get state for variable " + name)
	}

	// Return message if variable doesn't exist
	// Variable does not exist if byte array has length 0
	if len(valAsBytes) == 0 {
		return createQueryResponseString(false, "Variable \"" + name + "\" does not exist")
	}

	// Successful return
	return createQueryResponseBytes(true, valAsBytes)

}

// See if there are any pending transactions. Return it if there is a pending transaction.
func getPendingTransaction(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var pt []Transaction
	fmt.Println("Trying to get the pending transaction")

	// Get the pending transaction from the chaincode state
	transactionAsBytes, err := stub.GetState(pendingTransactionKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get pending transaction")
	}
	json.Unmarshal(transactionAsBytes, &pt)

	// Make sure there isn't more than 1 pending transaction
	// Otherwise, return the pt array
	if len(pt) > 1 {
		return createQueryResponseString(false, "More than 1 pending transaction, something is wrong!")
	} else {
		return createQueryResponseTransactions(true, pt)
	}

}

// Get all of the available offers
func getOffers(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var offers map[string]int
	fmt.Println("Trying to get the available offers")

	// Get the available offers from the chaincode state
	offersAsBytes, err := stub.GetState(offersKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get available offers")
	}
	json.Unmarshal(offersAsBytes, &offers)

	return createQueryResponseMap(true, offers)

}

// Get all of the past transactions
func getTransactions(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var t []Transaction
	fmt.Println("Trying to get the past transactions")

	// Get the past transactions from the chaincode state
	transactionsAsBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get past transactions")
	}
	json.Unmarshal(transactionsAsBytes, &t)

	return createQueryResponseTransactions(true, t)

}

// Get the details of all of the customers
func getCustomers(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var c map[string]int
	fmt.Println("Trying to get the list of customers")

	// Get the customers from the chaincode state
	customersAsBytes, err := stub.GetState(customersKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get customers")
	}
	json.Unmarshal(customersAsBytes, &c)

	return createQueryResponseMap(true, c)
}

// Get the details of a specific customer
func getCustomer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var customers map[string]int

	// Check parameters
	if len(args) != 1 {
		return createQueryResponseString(false, "Incorrect number of arguments. Expecting 1: Customer ID")
	}
	// Check customer name
	if len(args[0]) == 0 {
		return createQueryResponseString(false,	"First argument (customer name) cannot be an empty string")
	}

	// Convert customer ID argument to lowercase
	customerID := strings.ToLower(args[0])

	// Debug message
	fmt.Println("Trying to get the customer named " + customerID)

	// Get the customers from the chaincode state
	customersAsBytes, err := stub.GetState(customersKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get customers")
	}
	json.Unmarshal(customersAsBytes, &customers)

	// Make sure requested customer is in the list
	if val, ok := customers[customerID]; ok {
		return createQueryResponseInt(true, val)
	} else {
		return createQueryResponseString(false, "Failed to find customer with ID " + customerID)
	}

}

// Calculate the total number of energy units available
func getTotalEnergyForSale(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var offers map[string]int
	fmt.Println("Trying to calculate the total number of energy units available")

	// Get the available offers from the chaincode state
	offersAsBytes, err := stub.GetState(offersKey)
	if err != nil {
		return createQueryResponseString(false, "Failed to get available offers")
	}
	json.Unmarshal(offersAsBytes, &offers)

	// Calculate the total energy for sale
	// Sum the values over all of the keys
	total := 0
	for j := range offers {
		total += offers[j]
		fmt.Println("Key: " + j + ", Value: " + strconv.Itoa(offers[j]) + ". Total is now " + strconv.Itoa(total))
	}

	// Return the total
	return createQueryResponseInt(true, total)
}

//////////////////////////////////////// INVOKE FUNCTIONS ////////////////////////////////////////

func addOfferQuantity(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {

	var retStr string
	var offers map[string]int

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: offer ID, quantity to add to the offer"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (offer ID) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if len(args[1]) == 0 {
		retStr = "Second argument (quantity to add) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Check to make sure offer ID is a valid integer and not less than or equal to 0
	offerIDInt, err := strconv.Atoi(args[0])
	if err != nil {
		retStr = "First argument (Offer ID) must be an integer string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if offerIDInt <= 0 {
		retStr = "First argument (Offer ID) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Offers IDs are strings (thanks JSON!)
	offerID := args[0]

	// Check to make sure quantity to add is not less than or equal to 0
	quantity, err := strconv.Atoi(args[1])
	if err != nil {
		retStr = "Second argument (quantity to add) must be an integer string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if quantity <= 0 {
		retStr = "Second argument (quantity to add) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the available offers from the chaincode state
	offersAsBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offersAsBytes, &offers)

	// Try to find the specified offer
	// If found, add quantity to the offer
	// If not found, add new key and initialize value to quantity
	if _, ok := offers[offerID]; ok {
		offers[offerID] += quantity
	} else {
		offers[offerID] = quantity
	}

	// Save updated offer list
	marshalAndPut(stub, offersKey, offers)

	// Successful return
	retStr = "Successfully added " + args[1] + " to offer " + args[0]
	fmt.Println(retStr)
	return []byte(retStr), nil

}

func subtractOfferQuantity(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {

	var retStr string
	var offers map[string]int

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: offer ID, quantity to subtract from the offer"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (offer ID) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if len(args[1]) == 0 {
		retStr = "Second argument (quantity to subtract) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Check to make sure offer ID is a valid integer and not less than or equal to 0
	offerIDInt, err := strconv.Atoi(args[0])
	if err != nil {
		retStr = "First argument (Offer ID) must be an integer string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if offerIDInt <= 0 {
		retStr = "First argument (Offer ID) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Offers IDs are strings (thanks JSON!)
	offerID := args[0]

	// Check to make sure quantity to subtract is not less than or equal to 0
	quantity, err := strconv.Atoi(args[1])
	if err != nil {
		retStr = "Second argument (quantity to subtract) must be an integer string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if quantity <= 0 {
		retStr = "Second argument (quantity to subtract) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the available offers from the chaincode state
	offersAsBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offersAsBytes, &offers)

	// Try to find the specified offer
	// If found and quantity < val, subtract quantity from the offer
	// If found and quantity >= val, remove offer from map
	// If not found, return error
	if val, ok := offers[offerID]; ok {
		if quantity < val {
			offers[offerID] -= quantity
		} else {
			delete(offers, offerID)
		}
	} else {
		retStr = "Offer ID " + offerID + " does not exist"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Save updated offer list
	marshalAndPut(stub, offersKey, offers)

	// Successful return
	retStr = "Successfully subtracted " + args[1] + " from offer " + offerID
	fmt.Println(retStr)
	return []byte(retStr), errors.New(retStr)

}

// Add a new customer to the list of customers
func addCustomer(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {

	var retStr string
	var err error
	var customers map[string]int

	// Check parameters
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: new customer ID"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (customer ID) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to add a customer with ID " + args[0])

	// Convert potential new customer's name to lowercase
	newCustomer := strings.ToLower(args[0])

	// Get the list of customers from the chaincode state
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customers)

	// Check to see if the new customer is already a customer
	if _, ok := customers[newCustomer]; ok {
		retStr = "Cannot add customer '" + newCustomer + "': customer already exists"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Customer is able to be added, add them to the list of customers
	customers[newCustomer] = 0

	// Write customer list to chaincode state
	marshalAndPut(stub, customersKey, customers)

	// Successful return
	fmt.Println("Successfully added new customer")
	retStr = "Successfully added new customer"
	return []byte(retStr), nil

}

// Add amount to customer's balance
func addCustomerFunds(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {
	var retStr string
	var err error
	var customers map[string]int

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: customer ID, amount to add"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (customer ID) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if len(args[1]) == 0 {
		retStr = "Second argument (amount to add) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to add " + args[1] + " to " + args[0] + "'s balance")

	// Build Customer to hold Customer update
	customerName := strings.ToLower(args[0])
	funds, err := strconv.Atoi(args[1])
	if err != nil {
		retStr = "Second argument (funds) must be a numeric string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Funds to add cannot be zero or negative
	if funds <= 0 {
		retStr = "Second argument (funds) must not be negative"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the list of customers from the chaincode state
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customers)

	// Try to find the customer in the list of customers
	if _, ok := customers[customerName]; ok {
		// Update balance
		customers[customerName] += funds
		// Write updated customer list to the chaincode state
		marshalAndPut(stub, customersKey, customers)
		// Successful return
		fmt.Println("Successfully added " + strconv.Itoa(funds) + " to " + customerName + "'s balance")
		retStr = "Successfully added " + strconv.Itoa(funds) + " to " + customerName + "'s balance"
		return []byte(retStr), nil
	} else {
		// Customer wasn't found, return error message
		retStr = "Could not find customer " + customerName + " to add funds"
		fmt.Println(retStr)
		return []byte(retStr), nil
	}

}
// Accept offer
func acceptOffer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var retStr string
	var pendingTransaction []Transaction
	var newTransaction Transaction
	var offers map[string]int
	var customers map[string]int

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: customer ID, units of energy to buy"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (customer ID) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if len(args[1]) == 0 {
		retStr = "Second argument (quantity to buy) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println(args[0] + " is trying to purchase " + args[1] + " units of energy")

	// Check to see if there is a pending transaction
	fmt.Println("Checking to see if there is a pending transaction")
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransaction)
	// Return and do nothing if no pending transactions
	if len(pendingTransaction) > 0 {
		retStr = "There is already a pending transaction. Cannot accept an offer while a transaction is in progress."
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Process parameters
	fmt.Println("Processing parameters")
	buyer := strings.ToLower(args[0])
	requestedQuantity, err := strconv.Atoi(args[1])
	if err != nil {
		retStr = "Second argument (Offer ID) must be an integer string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Set new transaction energy total now because requestedQuantity will be altered later
	newTransaction.Energy = requestedQuantity

	// Get the list of available offers
	fmt.Println("Getting available offers")
	offerListBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offerListBytes, &offers)

	// Make sure quantity to buy is not greater than quantity available
	totalAvailable := 0
	for i, val := range offers {
		totalAvailable += val
		fmt.Println("Key: " + i + ", Value: " + strconv.Itoa(val) + ". Total available is now " + strconv.Itoa(totalAvailable))
	}
	if totalAvailable < requestedQuantity {
		retStr = "Requested " + args[1] + " with only " + strconv.Itoa(totalAvailable) + " available"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the list of customers from the chaincode state
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customers)

	// Make sure the buyer is a valid customer
	if _, ok := customers[buyer]; !ok {
		retStr = args[0] + " is not a valid buyer"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Calculate the cost of the transaction
	// Initialize newTransaction map before writing offers details to it
	newTransaction.Offers = make(map[string]int)
	// Make an array of price per unit offers in ascending order as integers
	ascendingOfferKeys := getMapStringKeysAsAscendingInts(offers)
	totalCost := 0
	for _, pricePerUnit := range ascendingOfferKeys {
		pricePerUnitStr := strconv.Itoa(pricePerUnit)
		unitsAvailable := offers[pricePerUnitStr]
		if unitsAvailable > requestedQuantity {
			// This price tier has enough, don't need to go to the next one
			// Subtract requested quantity from current price tier
			offers[pricePerUnitStr] -= requestedQuantity
			// Calculate cost and add to running total
			totalCost += requestedQuantity * pricePerUnit
			// Update new transaction to include this price tier
			newTransaction.Offers[pricePerUnitStr] = requestedQuantity
			// Done calculating cost and finding assets to buy, break
			break
		} else {
			// Take everything from this price tier and move to the next one
			// Update requested quantity
			requestedQuantity -= unitsAvailable
			// Calculate cost and add to running total
			totalCost += unitsAvailable * pricePerUnit
			// Update new transaction to include this tier
			newTransaction.Offers[pricePerUnitStr] = unitsAvailable
			// Delete the map key for this price tier
			delete(offers, pricePerUnitStr)
			// Check exit condition: requestedQuantity = 0
			if requestedQuantity == 0 {
				break
			}
			// Continue to the next one
		}
	}

	// Make sure the customer has enough funds to purchase this transaction
	if customers[buyer] < totalCost {
		retStr = "Buyer does not have enough funds: total cost = " + strconv.Itoa(totalCost) + ", available funds = " + strconv.Itoa(customers[buyer])
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// TRANSACTION IS VALID
	// Clean up transaction and finalize all changes that must be made

	// Subtract funds from customer and add funds to "owner" (owner of the EV charger)
	customers[buyer] -= totalCost
	customers["owner"] += totalCost

	// Add remaining fields to new transaction
	newTransaction.Status = "Pending"
	newTransaction.Buyer = buyer
	newTransaction.Cost = totalCost
	// newTransaction.Energy was set at the beginning of this function
	// newTransaction.Offers was set in the previous loop
	// TXID will be updated upon completion or cancellation
	newTransaction.TXID = 0

	// Update pending transactions
	fmt.Println("Adding new transaction to pending transaction")
	pendingTransaction = append(pendingTransaction, newTransaction)
	err = marshalAndPut(stub, pendingTransactionKey, pendingTransaction)
	if err != nil {
		retStr = "Could not write pendingTransactionKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Update customer accounts
	fmt.Println("Writing updated customer list to chaincode state")
	err = marshalAndPut(stub, customersKey, customers)
	if err != nil {
		retStr = "Could not write customersKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Update available offers
	fmt.Println("Writing updated available offers to chaincode state")
	err = marshalAndPut(stub, offersKey, offers)
	if err != nil {
		retStr = "Could not write offersKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Successful return
	retStr = "Successfully accepted the offer"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

// Complete transaction
func completeTransaction(stub shim.ChaincodeStubInterface) ([]byte, error) {

	var retStr string
	var pendingTransaction []Transaction
	var newTransaction Transaction
	var pastTransactions []Transaction

	// Debug message
	fmt.Println("Trying to complete the transaction")

	// Check to see if there is a pending transaction
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransaction)
	// Return and do nothing if no pending transactions
	if len(pendingTransaction) == 0 {
		retStr = "No pending transaction to be completed: accept an offer first"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Build the transaction to be added to the transactions list
	newTransaction = pendingTransaction[0]
	newTransaction.Status = "Completed"
	// TXID is the current UTC timestamp
	newTransaction.TXID = time.Now().Unix()

	// Get the list of past transactions
	transactionListBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		retStr = "Could not get transactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(transactionListBytes, &pastTransactions)

	// Append the new transaction to the list of completed transactions
	pastTransactions = append(pastTransactions, newTransaction)

	// Save the list of transactions to the chaincode state
	err = marshalAndPut(stub, transactionsKey, pastTransactions)
	if err != nil {
		retStr = "Could not write pendingTransactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Clear the pending transaction
	var emptyTransactions []Transaction
	// Update the pending transactions in the chaincode state
	err = marshalAndPut(stub, pendingTransactionKey, emptyTransactions)
	if err != nil {
		retStr = "Could not write pendingTransactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Successful return
	retStr = "Successfully completed the pending transaction"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

func cancelTransaction(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var retStr string
	var err error
	var pendingTransaction []Transaction
	var pastTransactions []Transaction
	var offers map[string]int
	var customers map[string]int

	// Check arguments
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: percentage to refund as an integer between 1 and 100"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check variable lengths
	if len(args[0]) == 0 {
		retStr = "First argument (units to refund) cannot be an empty string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	unitsToRefund, err := strconv.Atoi(args[0])
	if err != nil {
		retStr = "Could not convert " + args[0] + " to integer"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// unitsToRefund cannot be less than or equal to 0
	if unitsToRefund <= 0 {
		retStr = "First argument (units to refund) cannot be less than or equal to 0"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to cancel part the current transaction and refund " + args[0] + " units")

	// Check to see if there is a pending transaction
	fmt.Println("Getting pending transactions")
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransaction)
	// Return and do nothing if no pending transactions
	if len(pendingTransaction) == 0 {
		retStr = "No pending transactions to be completed: accept an offer first"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get pending transaction
	pt := pendingTransaction[0]

	// Check to make sure unitsToRefund is not greater than the amount of energy in the transaction
	if unitsToRefund > pt.Energy {
		retStr = "Cannot refund " + args[0] + " units, there are only " + strconv.Itoa(pt.Energy) + " in the current transaction"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the list of customers from the chaincode state
	fmt.Println("Getting customer accounts")
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customers)

	// Get the list of available offers
	fmt.Println("Getting available offers")
	offerListBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offerListBytes, &offers)

	// Get the list of past transactions
	transactionListBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		retStr = "Could not get transactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(transactionListBytes, &pastTransactions)


	// Make a slice out of the offer map's keys
	// Reverse the order so the most expensive tier is first
	offerKeys := reverseIntSlice(getMapStringKeysAsAscendingInts(pt.Offers))
	fmt.Println("Order of offer keys to refund: ", offerKeys)

	// Set pt.Energy now because unitsToRefund will be used & changed in the algorithm below
	pt.Energy -= unitsToRefund

	// Refund the most expensive units first
	// Keep refunding until enough units have been returned
	totalRefund := 0
	for i, pricePerUnit := range offerKeys {
		fmt.Println("Refund pass", i, "-", unitsToRefund, "units left to refund")
		pricePerUnitStr := strconv.Itoa(pricePerUnit)
		unitsBoughtAtCurrentTier := pt.Offers[pricePerUnitStr]
		fmt.Println("Currently processing offer tier " + pricePerUnitStr + " - " + strconv.Itoa(unitsBoughtAtCurrentTier) + " bought at this tier")
		if unitsToRefund <= unitsBoughtAtCurrentTier {
			// This price tier is the last that needs to be refunded from
			// Calculate cost of this part of the refund
			totalRefund += unitsToRefund * pricePerUnit
			// Check to see if this price tier still exists
			// If it exists, add the number of units
			// If it does not exist, create the offer tier and initialize it to
			if _, ok := offers[pricePerUnitStr]; ok {
				offers[pricePerUnitStr] += unitsToRefund
			} else {
				offers[pricePerUnitStr] = unitsToRefund
			}
			// Update this price tier in the pending transaction
			// If units bought at this tier ends up being zero, delete this tier from the map
			if unitsToRefund == unitsBoughtAtCurrentTier {
				delete(pt.Offers, pricePerUnitStr)
			} else {
				pt.Offers[pricePerUnitStr] -= unitsToRefund
			}
			// Don't need to update unitsToRefund
			// Last key that needs to be visited so break
			break
		} else {
			// This price tier is not the last that needs to be refunded from
			// Calculate cost of this part of the refund
			totalRefund += unitsBoughtAtCurrentTier * pricePerUnit
			// Check to see if this price tier still exists
			// If it exists, add the number of units
			// If it does not exist, create the offer tier and initialize it to
			if _, ok := offers[pricePerUnitStr]; ok {
				offers[pricePerUnitStr] += unitsBoughtAtCurrentTier
			} else {
				offers[pricePerUnitStr] = unitsBoughtAtCurrentTier
			}
			// Remove this price tier from the pending transaction
			delete(pt.Offers, pricePerUnitStr)
			// Update unitsToRefund
			unitsToRefund -= unitsBoughtAtCurrentTier
		}
	}

	// Refund the customer totalRefund from the owner's account
	customers[pt.Buyer] += totalRefund
	customers["owner"] -= totalRefund

	// Update pending transaction fields
	pt.Cost -= totalRefund
	pt.TXID = time.Now().Unix()
	pt.Status = "Refunded " + args[0]

	// Transaction has been refunded -- finalize transaction and save changes to the chaincode state

	// Append the pending transaction to past transactions
	pastTransactions = append(pastTransactions, pt)

	// Save the list of transactions to the chaincode state
	fmt.Println("Writing updated past transaction list to chaincode state")
	err = marshalAndPut(stub, transactionsKey, pastTransactions)
	if err != nil {
		retStr = "Could not write transactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Clear the pending transaction
	var emptyTransaction []Transaction
	fmt.Println("Writing updated pending transactions to chaincode state")
	// Update the pending transactions in the chaincode state
	err = marshalAndPut(stub, pendingTransactionKey, emptyTransaction)
	if err != nil {
		retStr = "Could not write pendingTransactionKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Update customer accounts
	fmt.Println("Writing updated customer list to chaincode state")
	err = marshalAndPut(stub, customersKey, customers)
	if err != nil {
		retStr = "Could not write customersKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Update available offers
	fmt.Println("Writing updated available offers to chaincode state")
	err = marshalAndPut(stub, offersKey, offers)
	if err != nil {
		retStr = "Could not write offersKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Successful return
	retStr = "Successfully refunded " + args[0] + " units of the pending transaction"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

// Add a transaction to the list of past transactions directly
// Used to create data for the website visualization
func addTransaction(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var retStr string
	var err error
	var newTransaction Transaction
	var pastTransactions []Transaction

	// Parameter order and needed type:
	//	TXID 	int64
	//	Buyer	string
	//	Energy 	int
	//	Cost 	int
	//	Offers	map[string]int

	// Check parameters
	// Number of parameters must be even and greater than or equal to 6
	if len(args) >= 6 || len(args) % 2 != 0 {
		retStr = "Incorrect number of arguments: Expecting an even number >= 6, received " + strconv.Itoa(len(args))
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Process parameters and make new transaction
	newTransaction.TXID, err = strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		retStr = "Could not parse first parameter [TXID] (" + args[0] + ") to int64"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	newTransaction.Buyer = args[1]
	newTransaction.Energy, err = strconv.Atoi(args[2])
	if err != nil {
		retStr = "Could not parse third parameter [amount of energy] (" + args[2] + ") to int"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	newTransaction.Cost, err = strconv.Atoi(args[3])
	if err != nil {
		retStr = "Could not parse fourth parameter [cost] (" + args[3] + ") to int"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Remaining parameters are offers and come in pairs: price tier, quantity
	newTransaction.Offers = make(map[string]int)
	offers := args[4:]
	for len(offers) > 0 {
		newTransaction.Offers[offers[0]], err = strconv.Atoi(offers[1])
		if err != nil {
			retStr = "Could not parse offer parameter (" + offers[1] + ") to int"
			fmt.Println(retStr)
			return []byte(retStr), errors.New(retStr)
		}
		offers = offers[2:]
	}

	// Transaction has been built, add it to past transactions list

	// Get the list of past transactions
	transactionListBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		retStr = "Could not get transactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(transactionListBytes, &pastTransactions)

	// Add new transaction to past transactions
	pastTransactions = append(pastTransactions, newTransaction)

	// Save the list of past transactions to the chaincode state
	fmt.Println("Writing updated past transaction list to chaincode state")
	err = marshalAndPut(stub, transactionsKey, pastTransactions)
	if err != nil {
		retStr = "Could not write transactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Successful return
	retStr = "Successfully added transaction to chaincode state"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

//////////////////////////////////////// UTILITY FUNCTIONS ////////////////////////////////////////

// Use the json package to marshal the interface into bytes, then store it in the chaincode state as the value of key
func marshalAndPut(stub shim.ChaincodeStubInterface, key string, v interface{}) (error) {

	var err error
	jsonAsBytes, _ := json.Marshal(v)
	err = stub.PutState(key,jsonAsBytes)
	if err != nil {
		return err
	}
	return nil

}

// Use the json package to marshal the data into bytes and construct a query response
func createQueryResponseString(success bool, data string) ([]byte, error) {
	var response QueryResponseString
	response.Success = success
	response.Data = data
	r, _ := json.Marshal(response)
	// If success, error is nil. If not success, data is an error message
	if success {
		return r, nil
	} else {
		return r, errors.New(data)
	}
}

func createQueryResponseMap(success bool, data map[string]int) ([]byte, error) {
	var response QueryResponseMap
	response.Success = success
	response.Data = data
	r, _ := json.Marshal(response)
	return r, nil
}

func createQueryResponseInt(success bool, data int) ([]byte, error) {
	var response QueryResponseInt
	response.Success = success
	response.Data = data
	r, _ := json.Marshal(response)
	return r, nil
}

func createQueryResponseTransactions(success bool, data []Transaction) ([]byte, error) {
	var response QueryResponseTransactions
	response.Success = success
	response.Data = data
	r, _ := json.Marshal(response)
	return r, nil
}

func createQueryResponseBytes(success bool, data []byte) ([]byte, error) {
	var response QueryResponseBytes
	response.Success = success
	response.Data = data
	r, _ := json.Marshal(response)
	return r, nil
}

// Get the keys of a map[string]int as ints
func getMapStringKeysAsAscendingInts(m map[string]int) ([]int) {
	// Create keys int array
	keys := make([]int, len(m))
	i := 0
	// Get all keys and turn them into ints with Atoi
	for j := range m {
		keys[i], _ = strconv.Atoi(j)
		i++
	}
	// Sort the integers in ascending order
	sort.Ints(keys)
	// Print out sorted keys for sanity check
	fmt.Println("Sorted integers:", keys)
	return keys
}

func reverseIntSlice(s []int) ([]int) {
	fmt.Println("Reversing integer slice")
	fmt.Println(s)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	fmt.Println(s)
	return s
}