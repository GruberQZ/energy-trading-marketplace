# Chaincode V3  
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
- **method:** Can be "query" or "invoke", refer to the Chaincode Functions section below.  
- **params.chaincodeID:** Should match the chaincodeID of the chaincode in Bluemix. Refer to the Bluemix dashboard for this information once the chaincode has been deployed.  
- **params.ctorMsg.function:** The name of the function  
- **params.ctorMsg.args:** An array of strings that represent arguments to the function. Refer to the Chaincode Functions section below. In the absense of parameters, an empty array should be used.  
- **params.secureContext:** EnrollmentID that was registered with one of the peers in Bluemix.  

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
- Offers represent the units of energy for sale at the EV charger.
- Offers are sorted by the price per unit of energy.
- Price per unit of energy is the key, the number of units for sale at that price per unit is the value of that key.
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
### Get total amount of energy for sale
Function name: "getTotalEnergyForSale"

Arguments: None

Notes/Restrictions:
- Sums all of the available energy at all price per unit tiers.
- Successful query returns an integer.

### Get transactions
Function name: "getTransactions"

Arguments: None

Notes/Restrictions:
- Transactions represent offers that have been accepted.
- The transactions returned by this function are only those that have been completed (pending transaction not included).
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
- There is a persistant/reserved account called "owner" which represents the owner of the EV charger. Revenue from sales at this EV charger will be directed to this account.
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
### Add quantity to offer tier
Function name: "addOfferQuantity"

Arguments: 

1. Offer ID
2. Quantity to add

Notes/Restrictions:
- Offers are identified by the price per unit of the energy, also referred to as offer tier.
- If offer ID exists, quantity will be added to the existing tier.
- If offer ID does not exist, a new price per unit tier will be created and its value will be initialized to the quantity.

### Subtract quantity to offer tier
Function name: "subtractOfferQuantity"

Arguments:

1. Offer ID
2. Quantity to add

Notes/Restrictions:
- Offers are identified by the price per unit of the energy, also referred to as offer tier.
- If offer ID exists and quantity to subtract is less than available amount, quantity will be subtracted from the existing tier.
- If offer ID exists and quantity to subtract is greater than or equal to available amount, offer tier will be removed.
- If offer ID does not exist, a new price per unit tier will be created and its value will be initialized to the quantity.

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
2. Units of energy to buy

Notes/Restrictions:
- Units of energy to buy must be an integer string
- Units of energy cannot be greater than the total amount of energy available for purchase across all tiers
- Buyer must have the necessary funds to purchase the specified energy in their account
- Energy will be purchased from cheapest to most expensive price per unit
- Units of energy to buy can be greater than the amount of energy in the cheapest offer tier
 - In this case, all of the units in the cheapest offer tier will be purchased and the next cheapest tier will be used recursively until enough units of energy have been purchased
- Units of energy are removed from the available offer tiers at the time of acceptance, not upon completion

### Complete a transaction
Function name: "completeTransaction"

Arguments: None

Notes/Restrictions:
- Used by the EV charger to mark the pending transaction as complete
 - transaction.Status = "Completed"
- transaction.TXID will be set to the current Unix time
- Pending transaction gets copied into the list of past transactions
- Pending transaction becomes empty

### Cancel a transaction
Function name: "cancelTransaction"

Arguments: 

1. Number of units to refund

Notes/Restrictions:
- Used by the EV charger to partially refund the customer part of their purchase if their transaction did not complete
 - transaction.Status = "Refunded x" where x is the number of units refunded
- Percentage of transaction to refund must be an integer between 1 and the total number of energy units purchased
- Energy units will be refunded in order from most expensive to least expensive
 - Example: Offer was accepted for 100 units for 2/ea, 50 units for 4/ea. If number of units to refund from this transaction is 75, 50 units at 4/ea and 25 units at 2/ea will be refunded. The total refund will be 250.
- The cost of the refund will be transferred from the owner's account to the buyer's account
- transaction.TXID will be set to the current Unix time

### Add a transaction
Function name: "addTransaction"

Arguments: an even number greater than or equal to 6

1. TXID (int64 as a string)
2. Buyer
3. Energy (int as a string)
4. Cost (int as a string)
5. Offers accepted in this transaction
- odd numbers >= 5: Offer tier
- even numbers >= 6: Amount bought at offer tier

Example: ["1490127351","ross","50","200","3","25","5","25"]
- This set of parameters corresponds to: "At Unix time 1490127351, Ross completed a transaction of 50 units of energy for a cost of 200. 25 units were bought at 3/ea and 25 units were bought at 5/ea.

Notes/Restrictions:
- addTransaction is used to inject custom data in order to create visualizations on the website. Should not be used for any other purpose.
- This function does NOT check to ensure Energy and Cost match the values described in the offer details. The example above is mathematically correct with respect to the total Energy and Cost of the transaction, but this is not mandatory.

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
Query methods return a message to the sender through the result.message property as a string. The returned message will be a stringified object with two properties "success" and "data". The "success" field is a boolean value that indicates if the query was completed successfully. If "success" is false, the "data" field will contain an error message as a string indicating what went wrong with the query. If success is true, the "data" field contains the requested data. The variable type of "data" will vary based on the query.
### Invoke Method Return Object
Regardless of the validity of the request, invoke methods will pass the UUID of the transaction through the result.message field of the return object. The UUID can be used to determine if the request was successful.
#### Return Object from /transactions/{UUID}
A GET request to /transactions/{UUID} can be used to determine the validity/success of an invocation. If the function and arguments are valid and legal and the invocation is not rejected, an object with transaction details will be returned. If the invocation is rejected, the return object will have a single property "Error" with a message stating that the transaction UUID does not exist.
