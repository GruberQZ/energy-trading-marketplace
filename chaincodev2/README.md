# Chaincode V2

# Example Request Object
Requests are POSTed to the /chaincode endpoint.
```javascript
{
  "jsonrpc": "2.0",
  "method": "query",
  "params": {
    "type": 1,
    "chaincodeID": {
      "name": <chaincodeID>
    },
    "ctorMsg": {
      "function": "getCustomer",
      "args": [
        "ross"
      ]
    },
    "secureContext": <enrollID>
  },
  "id": 0
}
```
## Properties of concern
All of the object properties above should remain the same except for the following:

method: Can be "query" or "invoke", refer to the Chaincode Functions section below.

params.chaincodeID: Should match the chaincodeID of the chaincode in Bluemix. Refer to the Bluemix dashboard for this information once the chaincode has been deployed.

params.ctorMsg.function: The name of the function 

params.ctorMsg.args: An array of strings that represent arguments to the function. Refer to the Chaincode Functions section below. In the absense of parameters, an empty array should be used.

params.secureContext: EnrollmentID that was registered with one of the peers in Bluemix.

# Chaincode Functions
This section breaks chaincode operations into sections based on their type and their usage. To use these commands, edit the "ctorMsg" property of the JSON object that is sent to /chaincode. Arguments to functions are always passed in as a string array.
## Query  
The "method" property in the JSON object that is sent to /chaincode for operations in this section should be set to "query".
### Read a variable from the chaincode state
Function name: "read"

Arguments: 

1. Name of variable

Notes/Restrictions: 
- This function is mainly used for debugging, it need not be used otherwise.
- Passing in the name of the variable that does not exist will yield an error message.
### Get the pending transaction
Function name: "getPendingTransaction"

Arguments: None

Notes/Restrictions: This function is used by the EV charger to determine if there are any pending transactions.
### Get available offers
Function name: "getOffers"

Arguments: None

Notes/Restrictions: 
- Offers represent potential transactions at this EV charger. 
- Offers are identified by an OfferID which is set automatically by the chaincode. 
- Each offer contains a cost, amount of energy, seller, and persistance. 
- Example return object below.
```javascript
{
  "jsonrpc": "2.0",
  "result": {
    "status": "OK",
    "message": "{\"offers\":{\"3\":{\"cost\":100,\"energy\":100,\"seller\":\"blake\",\"persist\":true},\"4\":{\"cost\":200,\"energy\":200,\"seller\":\"blake\",\"persist\":true}}}"
  },
  "id": 0
}
```
### Get transactions
Function name: "getTransactions"

Arguments: None

Notes/Restrictions:
- Transactions represent offers that have been accepted.
- The transactions returned by this function are only those that have been completed (pending transaction not included).
- The return object contains a single property "transactions" which contains an array of transactions.
- Each transaction contains an txid (timestamp of completion time), details of the accepted offer, buyer's ID, and the status of the transaction.
- Example return object below.
```javascript
{
  "jsonrpc": "2.0",
  "result": {
    "status": "OK",
    "message": "{\"transactions\":[{\"txid\":1488265396,\"offerid\":\"0\",\"cost\":30,\"energy\":100,\"seller\":\"james\",\"persist\":true,\"buyer\":\"blake\",\"status\":\"Completed\"},{\"txid\":1488266074,\"offerid\":\"1\",\"cost\":500,\"energy\":200,\"seller\":\"blake\",\"persist\":false,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488266428,\"offerid\":\"2\",\"cost\":500,\"energy\":200,\"seller\":\"blake\",\"persist\":false,\"buyer\":\"james\",\"status\":\"Refunded 250 (50%)\"},{\"txid\":1488267327,\"offerid\":\"3\",\"cost\":100,\"energy\":100,\"seller\":\"blake\",\"persist\":true,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488267434,\"offerid\":\"3\",\"cost\":100,\"energy\":100,\"seller\":\"blake\",\"persist\":true,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488267532,\"offerid\":\"4\",\"cost\":200,\"energy\":200,\"seller\":\"blake\",\"persist\":true,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488267599,\"offerid\":\"5\",\"cost\":400,\"energy\":400,\"seller\":\"blake\",\"persist\":false,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488267756,\"offerid\":\"4\",\"cost\":200,\"energy\":200,\"seller\":\"blake\",\"persist\":true,\"buyer\":\"james\",\"status\":\"Completed\"},{\"txid\":1488308042,\"offerid\":\"6\",\"cost\":400,\"energy\":200,\"seller\":\"james\",\"persist\":false,\"buyer\":\"blake\",\"status\":\"Completed\"}]}"
  },
  "id": 0
}
```
### Get customer accounts
Function name: "getCustomers"

Arguments: None

Notes/Restrictions:
- Customers represent potential buyers and sellers at the EV charger.
- Each customer has an associated account balance.
- Example return object below.
```javascript
{
  "jsonrpc": "2.0",
  "result": {
    "status": "OK",
    "message": "{\"customers\":{\"blake\":1370,\"james\":710,\"jose\":0,\"ross\":0}}"
  },
  "id": 0
}
```
### Get customer account
Function name: "getCustomer"

Arguments:

1. Customer ID

Notes/Restrictions:
- Returns the customer ID and account balance of a customer.
- Example return object below.
```javascript
{
  "jsonrpc": "2.0",
  "result": {
    "status": "OK",
    "message": "{\"custid\":\"blake\",\"balance\":1370}"
  },
  "id": 0
}
```
## Invoke  
The "method" property in the JSON object that is sent to /chaincode for operations in this section should be set to "invoke".
### Add an offer
Function name: "addOffer"

Arguments: 

1. Seller's Customer ID
2. Cost
3. Amount of Energy
4. Persistance

Notes/Restrictions:
- Used to add an offer to the list of available offers
- Seller's Customer ID must be an existing account
- Cost must be an integer
- Cost must not be less than 0
- Amount of energy must be an integer
- Amount of energy must not be less than 0
- Persistance must be passed in as "true" or "false" 
 - If persistance is true, the offer will be available until it is deleted
 - If persistance is false, the offer will be deleted once it has been accepted
- Offer ID will be added automatically

### Delete an offer
Function name: "deleteOffer"

Arguments:

1. Offer ID

Notes/Restrictions:
- Used to remove an offer from the list of available offers
- Offer ID must match an available Offer's ID

### Add a customer
Function name: "addCustomer"

Arguments: 

1. Customer ID

Notes/Restrictions:
- Used to create a new customer account
- New customer account will initialize to a balance of 0
- Customer ID will be converted to lower case
- Customer ID must not match the ID of an existing customer account
- Customer accounts cannot be deleted

### Add funds to customer account
Function name: "addCustomerFunds"

Arguments:

1. Customer ID
2. Amount to add

Notes/Restrictions:
- Used to add funds to a customer account
- Customer ID must match a customer account that already exists
- Amount to add must be non-zero and positive

### Accept an offer
Function name: "acceptOffer"

Arguments: 

1. Buyer's Customer ID
2. Offer ID

Notes/Restrictions:
- Used to accept an available offer
- An offer cannot be accepted while there is a pending transaction
- Buyer's Customer ID must match an existing customer account
- Offer ID must match an available Offer's ID
- Buyer's account must contain greater than or equal to the cost of the offer
- If purchase requirements are met:
 - Funds will be transferred from buyer to seller
 - If the accepted offer was not persistant, the offer will be deleted
 - The accepted offer is used to create a pending transaction

### Complete a transaction
Function name: "completeTransaction"

Arguments: None

Notes/Restrictions:
- Used by the EV charger to mark the pending transaction as complete
- Pending transaction gets copied into the list of past transactions
- Pending transaction becomes empty

### Cancel a transaction
Function name: "cancelTransaction"

Arguments: 

1. Percentage of transaction to refund

Notes/Restrictions:
- Used by the EV charger to partially refund the customer part of their purchase if their transaction did not complete
- Percentage of transaction to refund must be an integer between 1 and 100
- Example: Offer cost 200, percentage to refund is 30
 - 60 funds will be transferred back from seller to buyer

# Chaincode Function Return Object
## Return object from /chaincode
```javascript
{
  "jsonrpc": "2.0",
  "result": {
    "status": "OK",
    "message": "500"
  },
  "id": 123
}
```
### Important Note
"OK" in result.status in the return object does NOT mean that the input parameters were accepted and the chaincode function executed correctly. This merely means that the /chaincode endpoint received and processed the POSTed object.
### Query Method Return Object
Query methods return a message to the sender through the result.message property as a string. The message will return the requested information or give a reason for the failure of the request.
### Invoke Method Return Object
Regardless of the validity of the request, invoke methods will pass the UUID of the transaction through the result.message field of the return object. The UUID can be used to determine if the request was successful.
#### Return Object from /transactions/{UUID}
A GET request to /transactions/{UUID} can be used to determine the validity/success of an invocation. If the function and arguments are valid and legal and the invocation is not rejected, an object with transaction details will be returned. If the invocation is rejected, the return object will have a single property "Error" with a message stating that the transaction UUID does not exist.