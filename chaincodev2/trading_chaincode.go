package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Needed for function pointer (t *SimpleChaincode)
// Leave empty
type SimpleChaincode struct {
}

var customersKey = "_customers"       // key for list of customers
var offersKey = "_offers"             // key for list of current offers
var transactionsKey = "_transactions" // key for list of transactions
var offerIDKey = "_offerid"           // key for tracking the next offer ID
var pendingTransactionsKey = "_pendingtransactions" // key for tracking the pending transactions

type Customer struct {
	CustID	string 	`json:"custid"`
	Balance	int		`json:"balance"`
}

type Offer struct {
	Cost 	int		`json:"cost"`
	Energy 	int		`json:"energy"`
	Seller	string	`json:"seller"`
	Persist bool	`json:"persist"`
}

type Transaction struct {
	TXID 	int64 	`json:"txid"`
	OfferID	string	`json:"offerid"`
	Offer
	Buyer	string	`json:"buyer"`
	Status 	string	`json:"status"`
}

type CustomerList struct {
	Customers map[string]int `json:"customers"`
}

type OfferList struct {
	Offers map[string]Offer `json:"offers"`
}

type TransactionList struct {
	Transactions []Transaction `json:"transactions"`
}

type OfferID struct {
	NextID int `json:"nextid"`
}

type PendingTransactions struct {
	Transactions []Transaction `json:"transactions"`
}

// Main function - runs on start
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init - reset the state of the chaincode
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var initVal int
	var err error
	var retStr string

	// Check the number of args passed in
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: Initial value"
		return []byte(retStr), errors.New("Incorrect number of arguments. Expecting 1: Initial value")
	}

	// Get initial value
	initVal, err = strconv.Atoi(args[0])
	if err != nil {
		retStr = "Incorrect number of arguments. Expecting 1: Initial value"
		return []byte(retStr), errors.New("Expecting integer value for asset holding")
	}

	// Write initVal to the ledger
	// Use test var ece because reasons
	err = stub.PutState("ece", []byte(strconv.Itoa(initVal)))
	if err != nil {
		return nil, err
	}

	// Clear the list of customers
	var emptyCustomers CustomerList
	emptyCustomers.Customers = make(map[string]int)
	err = marshalAndPut(stub, customersKey, emptyCustomers)
	if err != nil {
		return nil, err
	}

	// Clear the list of offers
	var emptyOffers OfferList
	emptyOffers.Offers = make(map[string]Offer)
	err = marshalAndPut(stub, offersKey, emptyOffers)
	if err != nil {
		return nil, err
	}

	// Clear the list of transactions
	var emptyTransactions TransactionList
	emptyTransactions.Transactions = nil
	err = marshalAndPut(stub, transactionsKey, emptyTransactions)
	if err != nil {
		return nil, err
	}

	// Reset the offer ID counter
	var emptyOfferID OfferID
	emptyOfferID.NextID = 0
	err = marshalAndPut(stub, offerIDKey, emptyOfferID)
	if err != nil {
		return nil, err
	}

	// Clear the pending transaction
	var emptyPendingTransactions PendingTransactions
	emptyPendingTransactions.Transactions = nil
	err = marshalAndPut(stub, pendingTransactionsKey, emptyPendingTransactions)
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
	case "addOffer":
		return addOffer(stub, args)
	case "deleteOffer":
		return deleteOffer(stub, args)
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
	}

	// Print message if query function not found
	fmt.Println("Query() did not find function: " + function)

	// Return an error
	return []byte("Query() did not find function: " + function), errors.New("Received unknown query function: " + function)

}

// Read function
// Read a variable from chaincode state
func read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error
	var retStr string

	// Check to make sure number of arguments is correct
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: name of variable to query"
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to read variable named " + args[0])

	// Get the variable from the chaincode state
	name = args[0]
	valAsBytes, err := stub.GetState(name)
	if err != nil {
		retStr = "Could not get state for variable " + name
		fmt.Println(retStr)
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return []byte(retStr), errors.New(jsonResp)
	}

	// Return message if variable doesn't exist
	// Variable does not exist if byte array has length 0
	if len(valAsBytes) == 0 {
		return []byte("Variable \"" + name + "\" does not exist"), nil
	}

	// Successful return
	return valAsBytes, nil
}

// See if there are any pending transactions. Return it if there is a pending transaction.
func getPendingTransaction(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var pt PendingTransactions
	fmt.Println("Trying to get the pending transaction")
	// Get the pending transaction from the chaincode state
	transactionAsBytes, err := stub.GetState(pendingTransactionsKey)
	if err != nil {
		return nil, errors.New("Failed to get pending transaction")
	}
	json.Unmarshal(transactionAsBytes, &pt)
	// Make sure there isn't more than 1 pending transaction
	if len(pt.Transactions) > 1 {
		return []byte("More than 1 pending transaction, something is wrong!"), errors.New("More than 1 pending transaction, something is wrong!")
	} else {
		return transactionAsBytes, nil
	}
}

// Get all of the available offers
func getOffers(stub shim.ChaincodeStubInterface) ([]byte, error) {
	// Get the available offers from the chaincode state
	fmt.Println("Trying to get the available offers")
	offersAsBytes, err := stub.GetState(offersKey)
	if err != nil {
		return nil, errors.New("Failed to get available offers")
	}
	return offersAsBytes, nil
}

// Get all of the past transactions
func getTransactions(stub shim.ChaincodeStubInterface) ([]byte, error) {
	fmt.Println("Trying to get the past transactions")
	// Get the past transactions from the chaincode state
	transactionsAsBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		return nil, errors.New("Failed to get past transactions")
	}
	return transactionsAsBytes, nil
}

// Get the details of all of the customers
func getCustomers(stub shim.ChaincodeStubInterface) ([]byte, error) {
	fmt.Println("Trying to get the list of customers")
	// Get the customers from the chaincode state
	customersAsBytes, err := stub.GetState(customersKey)
	if err != nil {
		return nil, errors.New("Failed to get customers")
	}
	return customersAsBytes, nil
}

// Get the details of a specific customer
func getCustomer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var cl CustomerList
	var retStr string
	var c Customer

	 // Check parameters
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: Customer ID"
		return []byte(retStr), errors.New(retStr)
	}
	customerID := strings.ToLower(args[0])

	// Debug message
	fmt.Println("Trying to get the customer named " + customerID)

	// Get the customers from the chaincode state
	customersAsBytes, err := stub.GetState(customersKey)
	if err != nil {
		return nil, errors.New("Failed to get customers")
	}
	json.Unmarshal(customersAsBytes, &cl)

	// Make sure requested customer is in the list
	if val, ok := cl.Customers[customerID]; ok {
		c.CustID = customerID
		c.Balance = val
		customerAsBytes, err := json.Marshal(c)
		if err != nil {
			return nil, errors.New("Failed to marshal customer details")
		}
		return customerAsBytes, nil
	}

	// Customer ID wasn't found, return error message
	retStr = "Failed to find customer with ID " + customerID
	return []byte(retStr), nil
}

func marshalAndPut(stub shim.ChaincodeStubInterface, key string, v interface{}) (error) {
	var err error
	jsonAsBytes, _ := json.Marshal(v)
	err = stub.PutState(key,jsonAsBytes)
	if err != nil {
		return err
	}
	return nil
}

//func getAndUnmarshal(stub shim.ChaincodeStubInterface, key string, v *interface{}) (error) {
//	var err error
//	jsonAsBytes, err := stub.GetState(key)
//	if err != nil {
//		return errors.New("Failed to get state for key: " + key)
//	}
//	json.Unmarshal(jsonAsBytes, v)
//}

// Add a new offer to the list of offers
func addOffer(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {
	var retStr string
	var err error
	var currentOffers OfferList
	var newOffer Offer
	var offerID OfferID
	var customerList CustomerList

	// Check parameters
	if len(args) != 4 {
		retStr = "Incorrect number of arguments. Expecting 4: seller, cost, amount of energy, persistance"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to add an offer with seller " + args[0] + ", cost " + args[1] + ", amount of energy " + args[2] + ", and persistance " + args[3])

	requestingCustomer := strings.ToLower(args[0])

	// Use parameters to build new Offer
	newOffer.Cost, err = strconv.Atoi(args[1])
	if err != nil {
		retStr = "Second argument (cost) must be a numeric string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if newOffer.Cost < 0 {
		retStr = "Second argument (cost) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	newOffer.Energy, err = strconv.Atoi(args[2])
	if err != nil {
		retStr = "Third argument (amount of energy) must be a numeric string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if newOffer.Energy < 0 {
		retStr = "Third argument (amount of energy) must not be less than zero"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if args[3] == "true" {
		newOffer.Persist = true
	} else if args[3] == "false" {
		newOffer.Persist = false
	} else {
		retStr = "Fourth argument (persistance) must be 'true' or 'false'"
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
	json.Unmarshal(customerListBytes, &customerList)

	// Make sure seller in the list of customers
	if _, ok := customerList.Customers[requestingCustomer]; !ok {
		retStr = "CustomerID " + requestingCustomer + " does not exist"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Set seller of offer
	newOffer.Seller = requestingCustomer

	// Get the array of offers from the chaincode state
	currentOffersBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(currentOffersBytes, &currentOffers)

	// Get the next offer ID from the chaincode state
	offerIDBytes, err := stub.GetState(offerIDKey)
	if err != nil {
		retStr = "Could not get offerIDKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offerIDBytes, &offerID)

	// Add the new offer to the list of current offers with the next valid offer ID
	currentOffers.Offers[strconv.Itoa(offerID.NextID)] = newOffer

	// Increment the offer ID tracker
	offerID.NextID += 1

	// Save the updated offer ID tracker to the chaincode state
	marshalAndPut(stub, offerIDKey, offerID)
	// Write the updated offer list to the chaincode state
	marshalAndPut(stub, offersKey, currentOffers)

	// Successful return
	fmt.Println("Successfully added new offer to offer list")
	retStr = "Successfully added new offer to offer list"
	fmt.Println(retStr)
	return []byte(retStr), nil
}

// Delete an offer from the list of offers
func deleteOffer(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {
	var retStr string
	var err error
	var offerList OfferList

	// Check parameters
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: Offer ID"
		return []byte(retStr), errors.New(retStr)
	}

	fmt.Println("Trying to delete offer with ID " + args[0])

	// Convert parameter to integer
	requestedOfferID := args[0]

	// Get the offer list from the chaincode state
	offerListBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offerListBytes, &offerList)

	// Delete the offer from the available offers if it exists
	if _, ok := offerList.Offers[requestedOfferID]; ok {
		// Delete from map
		delete(offerList.Offers, requestedOfferID)
		// Write offer list back to chaincode state
		marshalAndPut(stub, offersKey, offerList)
		// Successful return
		fmt.Println("Successfully deleted offer from the offer list")
		retStr = "Successfully deleted offer from the offer list"
		return []byte(retStr), nil
	} else {
		// Offer wasn't found, return error message
		fmt.Println("Could not find offer with ID " + args[0] + " to remove")
		retStr = "Could not find offer with ID " + args[0] + " to remove"
		return []byte(retStr), nil
	}

}

// Add a new customer to the list of customers
func addCustomer(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {
	var retStr string
	var err error
	var customerList CustomerList

	// Check parameters
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: new customer ID"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to add a customer with ID " + args[0])

	// Build the new Customer
	newCustomer := strings.ToLower(args[0])

	// Get the list of customers from the chaincode state
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customerList)

	// Check to see if the new customer is already a customer
	if _, ok := customerList.Customers[newCustomer]; ok {
		retStr = "Cannot add customer '" + newCustomer + "': customer already exists"
		return []byte(retStr), errors.New(retStr)
	}

	// Customer is able to be added, add them to the list of customers
	customerList.Customers[newCustomer] = 0

	// Write customer list to chaincode state
	marshalAndPut(stub, customersKey, customerList)

	// Successful return
	fmt.Println("Successfully added new customer")
	retStr = "Successfully added new customer"
	return []byte(retStr), nil

}

// Add amount to customer's balance
func addCustomerFunds(stub shim.ChaincodeStubInterface, args []string) ([]byte,error) {
	var retStr string
	var err error
	var customerList CustomerList

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: customer ID, amount to add"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to add " + args[1] + " to " + args[0] + " balance")

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
	json.Unmarshal(customerListBytes, &customerList)

	// Try to find the customer in the list of customers
	if _, ok := customerList.Customers[customerName]; ok {
		// Update balance
		customerList.Customers[customerName] += funds
		// Write updated customer list to the chaincode state
		marshalAndPut(stub, customersKey, customerList)
		// Successful return
		fmt.Println("Successfully added " + strconv.Itoa(funds) + " to " + customerName + "'s balance")
		retStr = "Successfully added " + strconv.Itoa(funds) + " to " + customerName + "'s balance"
		return []byte(retStr), nil
	} else {
		// Customer wasn't found, return error message
		fmt.Println("Could not find customer " + customerName + " to add funds")
		retStr = "Could not find customer " + customerName + " to add funds"
		fmt.Println(retStr)
		return []byte(retStr), nil
	}

}

// Accept offer
func acceptOffer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var retStr string
	var err error
	var customerList CustomerList
	var requestedOffer Offer
	var offerList OfferList
	var newTransaction Transaction
	var pendingTransactions PendingTransactions

	// Check parameters
	if len(args) != 2 {
		retStr = "Incorrect number of arguments. Expecting 2: customer ID, ID of the offer to accept"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println(args[0] + " is trying to accept an offer with ID " + args[1])

	// Check to see if there is a pending transaction
	fmt.Println("Checking to see if there is a pending transaction")
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionsKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransactions)
	// Return and do nothing if no pending transactions
	if len(pendingTransactions.Transactions) > 0 {
		retStr = "There is already a pending transaction. Cannot accept an offer while a transaction is in progress."
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Process parameters
	fmt.Println("Processing parameters")
	requestingCustomer := strings.ToLower(args[0])
	requestedOfferID := args[1]
	if err != nil {
		retStr = "Second argument (Offer ID) must be a numeric string"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get the list of available offers
	fmt.Println("Getting available offers")
	offerListBytes, err := stub.GetState(offersKey)
	if err != nil {
		retStr = "Could not get offersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(offerListBytes, &offerList)

	// Look for the offer in the offer list
	fmt.Println("Looking for the requested offer in the offer list")
	if _, ok := offerList.Offers[requestedOfferID]; !ok {
		retStr = "Could not find offer with ID " + args[1]
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	fmt.Println("Found the requested offer")
	requestedOffer = offerList.Offers[requestedOfferID]

	// Get the list of customers from the chaincode state
	fmt.Println("Getting customer list")
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customerList)

	// Look for the buyer + seller in the Customer list
	fmt.Println("Looking for customer accounts for buyer and seller")
	if _, ok := customerList.Customers[requestingCustomer]; !ok {
		retStr = "Could not find customer with ID " + requestingCustomer
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if _, ok := customerList.Customers[requestedOffer.Seller]; !ok {
		retStr = "Could not find seller with ID " + requestedOffer.Seller
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	// Check buyer's balance
	if customerList.Customers[requestingCustomer] < requestedOffer.Cost {
		retStr = "Customer does not have necessary funds to purchase offer " + requestedOfferID
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	fmt.Println("Found the customer and seller accounts")

	// Subtract funds from Customer's balance
	fmt.Println("Buyer: subtracting " + strconv.Itoa(requestedOffer.Cost) + " from " + requestingCustomer)
	customerList.Customers[requestingCustomer] -= requestedOffer.Cost
	// Add funds to seller's balance
	fmt.Println("Seller: adding " + strconv.Itoa(requestedOffer.Cost) + " to " + requestedOffer.Seller)
	customerList.Customers[requestedOffer.Seller] += requestedOffer.Cost

	// Write the updated Customer list to the chaincode state
	fmt.Println("Writing updated customer list to chaincode state")
	err = marshalAndPut(stub, customersKey, customerList)
	if err != nil {
		retStr = "Could not write customersKey to chaincode state"
		return []byte(retStr), errors.New(retStr)
	}

	// Build a pending transaction with offer and customer details
	fmt.Println("Building new transaction (pending)")
	newTransaction.TXID = 0
	newTransaction.Buyer = requestingCustomer
	newTransaction.OfferID = args[1]
	newTransaction.Offer = requestedOffer
	//newTransaction.Cost = requestedOffer.Cost
	//newTransaction.Energy = requestedOffer.Energy
	//newTransaction.Persist = requestedOffer.Persist
	//newTransaction.Seller = requestedOffer.Seller
	newTransaction.Status = "Pending"

	// Add the new transaction to the pending transactions
	fmt.Println("Adding new transaction to pending transaction")
	pendingTransactions.Transactions = nil
	pendingTransactions.Transactions = append(pendingTransactions.Transactions, newTransaction)

	// Update pending transactions in chaincode
	fmt.Println("Writing pending transaction to chaincode state")
	err = marshalAndPut(stub, pendingTransactionsKey, pendingTransactions)
	if err != nil {
		retStr = "Could not write pendingTransactionsKey to chaincode state"
		return []byte(retStr), errors.New(retStr)
	}

	// If the offer was not persistant, delete it from the offer list and update the chaincode state
	if !requestedOffer.Persist {
		fmt.Println("Removing offer from available offers")
		delete(offerList.Offers, requestedOfferID)
		err = marshalAndPut(stub, offersKey, offerList)
		if err != nil {
			retStr = "Could not write offersKey to chaincode state"
			fmt.Println(retStr)
			return []byte(retStr), errors.New(retStr)
		}
	} else {
		fmt.Println("Requested offer is persistant so it will not be removed")
	}

	// Successful return
	retStr = "Successfully accepted the offer"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

// Complete transaction
func completeTransaction(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var retStr string
	var err error
	var pendingTransactions PendingTransactions
	var newTransaction Transaction
	var transactionList TransactionList

	// Debug message
	fmt.Println("Trying to complete the transaction")

	// Check to see if there is a pending transaction
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionsKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransactions)
	// Return and do nothing if no pending transactions
	if len(pendingTransactions.Transactions) == 0 {
		retStr = "No pending transactions to be completed: accept an offer first"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Build the transaction to be added to the transactions list
	newTransaction = pendingTransactions.Transactions[0]
	//newTransaction.OfferID = pendingTransactions.Transactions[0].OfferID
	//newTransaction.Cost = pendingTransactions.Transactions[0].Cost
	//newTransaction.Energy = pendingTransactions.Transactions[0].Energy
	//newTransaction.Persist = pendingTransactions.Transactions[0].Persist
	//newTransaction.Buyer = pendingTransactions.Transactions[0].Buyer
	//newTransaction.Seller = pendingTransactions.Transactions[0].Seller
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
	json.Unmarshal(transactionListBytes, &transactionList)
	// Append the new transaction to the list of completed transactions
	transactionList.Transactions = append(transactionList.Transactions, newTransaction)
	// Save the list of transactions to the chaincode state
	err = marshalAndPut(stub, transactionsKey, transactionList)
	if err != nil {
		retStr = "Could not write transactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Clear the pending transaction
	pendingTransactions.Transactions = nil
	// Update the pending transactions in the chaincode state
	err = marshalAndPut(stub, pendingTransactionsKey, pendingTransactions)
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

// Cancel transaction
func cancelTransaction(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var retStr string
	var err error
	var pendingTransactions PendingTransactions
	var customerList CustomerList
	var newTransaction Transaction
	var transactionList TransactionList

	// Check arguments
	if len(args) != 1 {
		retStr = "Incorrect number of arguments. Expecting 1: percentage to refund as an integer between 1 and 100"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	percentRefund, err := strconv.Atoi(args[0])
	if err != nil {
		retStr = "Could not convert " + args[0] + " to integer"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if percentRefund < 1 || percentRefund > 100 {
		retStr = "First argument must be an integer between 1 and 100"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Debug message
	fmt.Println("Trying to cancel the current transaction and refund " + args[0] + " percent")

	// Check to see if there is a pending transaction
	pendingTransactionsBytes, err := stub.GetState(pendingTransactionsKey)
	if err != nil {
		retStr = "Could not get pendingTransactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(pendingTransactionsBytes, &pendingTransactions)
	// Return and do nothing if no pending transactions
	if len(pendingTransactions.Transactions) == 0 {
		retStr = "No pending transactions to be completed: accept an offer first"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Get pending transaction
	pt := pendingTransactions.Transactions[0]

	// Get the list of customers from the chaincode state
	customerListBytes, err := stub.GetState(customersKey)
	if err != nil {
		retStr = "Could not get customersKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(customerListBytes, &customerList)

	// Ensure that both customer accounts exist
	// Look for the buyer + seller in the Customer list
	fmt.Println("Looking for customer accounts for buyer and seller")
	if _, ok := customerList.Customers[pt.Buyer]; !ok {
		retStr = "Could not find customer with ID " + pt.Buyer
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	if _, ok := customerList.Customers[pt.Seller]; !ok {
		retStr = "Could not find seller with ID " + pt.Seller
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Create percentage
	percentage := float32(percentRefund)/float32(100)
	refund := int(float32(pt.Cost)*percentage)

	// Remove funds from seller's account
	customerList.Customers[pt.Seller] -= refund
	// Add funds to buyer's account
	customerList.Customers[pt.Buyer] += refund

	// Save the updated customer list
	err = marshalAndPut(stub, customersKey, customerList)
	if err != nil {
		retStr = "Could not write customersKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Build the transaction to be added to the transactions list
	newTransaction = pendingTransactions.Transactions[0]
	//newTransaction.OfferID = pendingTransactions.Transactions[0].OfferID
	//newTransaction.Buyer = pendingTransactions.Transactions[0].Buyer
	//newTransaction.Cost = pendingTransactions.Transactions[0].Cost
	//newTransaction.Energy = pendingTransactions.Transactions[0].Energy
	//newTransaction.Persist = pendingTransactions.Transactions[0].Persist
	//newTransaction.Seller = pendingTransactions.Transactions[0].Seller
	newTransaction.Status = "Refunded " + strconv.Itoa(refund) + " (" + args[0] + "%)"
	// TXID is the current UTC timestamp
	newTransaction.TXID = time.Now().Unix()

	// Get the list of past transactions
	transactionListBytes, err := stub.GetState(transactionsKey)
	if err != nil {
		retStr = "Could not get transactionsKey from chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}
	json.Unmarshal(transactionListBytes, &transactionList)
	// Append the new transaction to the list of completed transactions
	transactionList.Transactions = append(transactionList.Transactions, newTransaction)
	// Save the list of transactions to the chaincode state
	err = marshalAndPut(stub, transactionsKey, transactionList)
	if err != nil {
		retStr = "Could not write transactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Clear the pending transaction
	pendingTransactions.Transactions = nil
	// Update the pending transactions in the chaincode state
	err = marshalAndPut(stub, pendingTransactionsKey, pendingTransactions)
	if err != nil {
		retStr = "Could not write pendingTransactionsKey to chaincode state"
		fmt.Println(retStr)
		return []byte(retStr), errors.New(retStr)
	}

	// Successful return
	retStr = "Successfully refunded " + args[0] + " percent of the pending transaction"
	fmt.Println(retStr)
	return []byte(retStr), nil

}

// Ignore this comment
