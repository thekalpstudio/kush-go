package token

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

const uriKey = "uri"

const balancePrefix = "account~tokenId~sender"
const approvalPrefix = "account~operator"

const minterMSPID = "mailabs"

// Define key names for options
const nameKey = "name"
const symbolKey = "symbol"

// SmartContract provides functions for transferring tokens between accounts
type SmartContract struct {
	kalpsdk.Contract
}

// TransferSingle MUST emit when a single token is transferred, including zero
// value transfers as well as minting or burning.
type TransferSingle struct {
	Operator string `json:"operator"`
	From     string `json:"from"`
	To       string `json:"to"`
	ID       uint64 `json:"id"`
	Value    uint64 `json:"value"`
}

// TransferBatch MUST emit when tokens are transferred, including zero value
// transfers as well as minting or burning.
type TransferBatch struct {
	Operator string   `json:"operator"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	IDs      []uint64 `json:"ids"`
	Values   []uint64 `json:"values"`
}

// ApprovalForAll MUST emit when approval for a second party/operator address
// to manage all tokens for an owner address is enabled or disabled
type ApprovalForAll struct {
	Owner    string `json:"owner"`
	Operator string `json:"operator"`
	Approved bool   `json:"approved"`
}

// URI MUST emit when the URI is updated for a token ID.
type URI struct {
	Value string `json:"value"`
	ID    uint64 `json:"id"`
}
// Mint creates amount tokens of token type id and assigns them to account.
func (s *SmartContract) Mint(sdk kalpsdk.TransactionContextInterface, account string, id uint64, amount uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	err = mintHelper(sdk, operator, account, id, amount)
	if err != nil {
		return err
	}
	transferSingleEvent := TransferSingle{operator, "0x0", account, id, amount}
	return emitTransferSingle(sdk, transferSingleEvent)
}

// Mint creates amount tokens of token type id and assigns them to account.
func (s *SmartContract) Mint(sdk kalpsdk.TransactionContextInterface, account string, id uint64, amount uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	err = mintHelper(sdk, operator, account, id, amount)
	if err != nil {
		return err
	}
	transferSingleEvent := TransferSingle{operator, "0x0", account, id, amount}
	return emitTransferSingle(sdk, transferSingleEvent)
}

// MintBatch creates amount tokens for each token type id and assigns them to account.
func (s *SmartContract) MintBatch(sdk kalpsdk.TransactionContextInterface, account string, ids []uint64, amounts []uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if len(ids) != len(amounts) {
		return fmt.Errorf("ids and amounts must have the same length")
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	amountToSend := make(map[uint64]uint64)
	for i := 0; i < len(amounts); i++ {
		amountToSend[ids[i]], err = add(amountToSend[ids[i]], amounts[i])
		if err != nil {
			return err
		}
	}
	amountToSendKeys := sortedKeys(amountToSend)
	for _, id := range amountToSendKeys {
		amount := amountToSend[id]
		err = mintHelper(sdk, operator, account, id, amount)
		if err != nil {
			return err
		}
	}
	transferBatchEvent := TransferBatch{operator, "0x0", account, ids, amounts}
	return emitTransferBatch(sdk, transferBatchEvent)
}
// Burn destroys amount tokens of token type id from account.
func (s *SmartContract) Burn(sdk kalpsdk.TransactionContextInterface, account string, id uint64, amount uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if account == "0x0" {
		return fmt.Errorf("burn to the zero address")
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	err = removeBalance(sdk, account, []uint64{id}, []uint64{amount})
	if err != nil {
		return err
	}
	transferSingleEvent := TransferSingle{operator, account, "0x0", id, amount}
	return emitTransferSingle(sdk, transferSingleEvent)
}


// TransferFrom transfers tokens from sender account to recipient account.
func (s *SmartContract) TransferFrom(sdk kalpsdk.TransactionContextInterface, sender string, recipient string, id uint64, amount uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if sender == recipient {
		return fmt.Errorf("transfer to self")
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	if operator != sender {
		approved, err := _isApprovedForAll(sdk, sender, operator)
		if err != nil || !approved {
			return fmt.Errorf("caller is not owner nor is approved")
		}
	}
	err = removeBalance(sdk, sender, []uint64{id}, []uint64{amount})
	if err != nil {
		return err
	}
	if recipient == "0x0" {
		return fmt.Errorf("transfer to the zero address")
	}
	err = addBalance(sdk, sender, recipient, id, amount)
	if err != nil {
		return err
	}
	transferSingleEvent := TransferSingle{operator, sender, recipient, id, amount}
	return emitTransferSingle(sdk, transferSingleEvent)
}
// BurnBatch destroys amount tokens of for each token type id from account.
func (s *SmartContract) BurnBatch(sdk kalpsdk.TransactionContextInterface, account string, ids []uint64, amounts []uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if account == "0x0" {
		return fmt.Errorf("burn to the zero address")
	}
	if len(ids) != len(amounts) {
		return fmt.Errorf("ids and amounts must have the same length")
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	err = removeBalance(sdk, account, ids, amounts)
	if err != nil {
		return err
	}
	transferBatchEvent := TransferBatch{operator, account, "0x0", ids, amounts}
	return emitTransferBatch(sdk, transferBatchEvent)
}

// BatchTransferFrom transfers multiple tokens from sender account to recipient account.
func (s *SmartContract) BatchTransferFrom(sdk kalpsdk.TransactionContextInterface, sender string, recipient string, ids []uint64, amounts []uint64) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if sender == recipient {
		return fmt.Errorf("transfer to self")
	}
	if len(ids) != len(amounts) {
		return fmt.Errorf("ids and amounts must have the same length")
	}
	operator, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	if operator != sender {
		approved, err := _isApprovedForAll(sdk, sender, operator)
		if err != nil || !approved {
			return fmt.Errorf("caller is not owner nor is approved")
		}
	}
	err = removeBalance(sdk, sender, ids, amounts)
	if err != nil {
		return err
	}
	if recipient == "0x0" {
		return fmt.Errorf("transfer to the zero address")
	}
	amountToSend := make(map[uint64]uint64)
	for i := 0; i < len(amounts); i++ {
		amountToSend[ids[i]], err = add(amountToSend[ids[i]], amounts[i])
		if err != nil {
			return err
		}
	}
	amountToSendKeys := sortedKeys(amountToSend)
	for _, id := range amountToSendKeys {
		amount := amountToSend[id]
		err = addBalance(sdk, sender, recipient, id, amount)
		if err != nil {
			return err
		}
	}
	transferBatchEvent := TransferBatch{operator, sender, recipient, ids, amounts}
	return emitTransferBatch(sdk, transferBatchEvent)
}

// IsApprovedForAll returns true if operator is approved to transfer account's tokens.
func (s *SmartContract) IsApprovedForAll(sdk kalpsdk.TransactionContextInterface, account string, operator string) (bool, error) {
	return _isApprovedForAll(sdk, account, operator)
}

// SetApprovalForAll returns true if operator is approved to transfer account's tokens.
func (s *SmartContract) SetApprovalForAll(sdk kalpsdk.TransactionContextInterface, operator string, approved bool) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	account, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}
	if account == operator {
		return fmt.Errorf("setting approval status for self")
	}
	approvalForAllEvent := ApprovalForAll{account, operator, approved}
	approvalForAllEventJSON, err := json.Marshal(approvalForAllEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = sdk.SetEvent("ApprovalForAll", approvalForAllEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}
	approvalKey, err := sdk.CreateCompositeKey(approvalPrefix, []string{account, operator})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", approvalPrefix, err)
	}
	approvalJSON, err := json.Marshal(approved)
	if err != nil {
		return fmt.Errorf("failed to encode approval JSON of operator %s for account %s: %v", operator, account, err)
	}
	err = sdk.PutStateWithoutKYC(approvalKey, approvalJSON)
	if err != nil {
		return err
	}
	return nil
}

// BalanceOf returns the balance of the given account
func (s *SmartContract) BalanceOf(sdk kalpsdk.TransactionContextInterface, account string, id uint64) (uint64, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	return balanceOfHelper(sdk, account, id)
}
// BalanceOfBatch returns the balance of multiple account/token pairs
func (s *SmartContract) BalanceOfBatch(sdk kalpsdk.TransactionContextInterface, accounts []string, ids []uint64) ([]uint64, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return nil, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if len(accounts) != len(ids) {
		return nil, fmt.Errorf("accounts and ids must have the same length")
	}
	balances := make([]uint64, len(accounts))
	for i := 0; i < len(accounts); i++ {
		balances[i], err = balanceOfHelper(sdk, accounts[i], ids[i])
		if err != nil {
			return nil, err
		}
	}
	return balances, nil
}

// ClientAccountBalance returns the balance of the requesting client's account
func (s *SmartContract) ClientAccountBalance(sdk kalpsdk.TransactionContextInterface, id uint64) (uint64, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	clientID, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return 0, fmt.Errorf("failed to get client id: %v", err)
	}
	return balanceOfHelper(sdk, clientID, id)
}

// ClientAccountID returns the id of the requesting client's account
func (s *SmartContract) ClientAccountID(sdk kalpsdk.TransactionContextInterface) (string, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	clientAccountID, err := sdk.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client id: %v", err)
	}
	return clientAccountID, nil
}

// URI returns the URI
func (s *SmartContract) URI(sdk kalpsdk.TransactionContextInterface, id uint64) (string, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	uriBytes, err := sdk.GetState(uriKey)
	if err != nil || uriBytes == nil {
		return "", fmt.Errorf("failed to get uri: %v", err)
	}
	return string(uriBytes), nil
}

// SetURI set the URI value
func (s *SmartContract) SetURI(sdk kalpsdk.TransactionContextInterface, uri string) error {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	err = authorizationHelper(sdk)
	if err != nil {
		return err
	}
	if !strings.Contains(uri, "{id}") {
		return fmt.Errorf("failed to set uri, uri should contain '{id}'")
	}
	err = sdk.PutStateWithoutKYC(uriKey, []byte(uri))
	if err != nil {
		return fmt.Errorf("failed to set uri: %v", err)
	}
	return nil
}

// Symbol returns an abbreviated name for fungible tokens in this contract.
func (s *SmartContract) Symbol(sdk kalpsdk.TransactionContextInterface) (string, error) {
	initialized, err := checkInitialized(sdk)
	if err != nil || !initialized {
		return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	bytes, err := sdk.GetState(symbolKey)
	if err != nil {
		return "", fmt.Errorf("failed to get Symbol: %v", err)
	}
	return string(bytes), nil
}

// Set information for a token and initialize contract.
func (s *SmartContract) Initialize(sdk kalpsdk.TransactionContextInterface, name string, symbol string) (bool, error) {
	clientMSPID, err := sdk.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != minterMSPID {
		return false, fmt.Errorf("client is not authorized to initialize contract")
	}
	bytes, err := sdk.GetState(nameKey)
	if err != nil || bytes != nil {
		return false, fmt.Errorf("contract options are already set, client is not authorized to change them")
	}
	err = sdk.PutStateWithoutKYC(nameKey, []byte(name))
	if err != nil {
		return false, fmt.Errorf("failed to set token name: %v", err)
	}
	err = sdk.PutStateWithoutKYC(symbolKey, []byte(symbol))
	if err != nil {
		return false, fmt.Errorf("failed to set symbol: %v", err)
	}
	return true, nil
}

// Helper Functions

func authorizationHelper(sdk kalpsdk.TransactionContextInterface) error {
	clientMSPID, err := sdk.GetClientIdentity().GetMSPID()
	if err != nil || clientMSPID != minterMSPID {
		return fmt.Errorf("client is not authorized to mint new tokens")
	}
	return nil
}
func mintHelper(sdk kalpsdk.TransactionContextInterface, operator string, account string, id uint64, amount uint64) error {
	if account == "0x0" {
		return fmt.Errorf("mint to the zero address")
	}
	if amount <= 0 {
		return fmt.Errorf("mint amount must be a positive integer")
	}
	return addBalance(sdk, operator, account, id, amount)
}

// addBalance is a function that adds the specified amount of tokens to the balance of a recipient.
// It takes a transaction context interface, sender address, recipient address, token ID, and amount as parameters.
// The function creates a composite key using the recipient, token ID, and sender address.
// It then retrieves the current balance from the world state using the composite key.
// If the balance exists, it is parsed into a uint64 value.
// The function adds the specified amount to the balance using the add function.
// Finally, it updates the balance in the world state and returns any error that occurred during the process.
func addBalance(sdk kalpsdk.TransactionContextInterface, sender string, recipient string, id uint64, amount uint64) error {
	idString := strconv.FormatUint(uint64(id), 10)
	balanceKey, err := sdk.CreateCompositeKey(balancePrefix, []string{recipient, idString, sender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", balancePrefix, err)
	}
	balanceBytes, err := sdk.GetState(balanceKey)
	if err != nil {
		return fmt.Errorf("failed to read account %s from world state: %v", recipient, err)
	}
	balance := uint64(0)
	if balanceBytes != nil {
		balance, _ = strconv.ParseUint(string(balanceBytes), 10, 64)
	}
	balance, err = add(balance, amount)
	if err != nil {
		return err
	}
	return sdk.PutStateWithoutKYC(balanceKey, []byte(strconv.FormatUint(uint64(balance), 10)))
}

// setBalance sets the balance of a specific token for a given sender and recipient.
// It creates a composite key using the recipient, token ID, and sender, and stores the balance amount in the state.
// Parameters:
// - sdk: The transaction context interface for interacting with the blockchain.
// - sender: The address of the sender.
// - recipient: The address of the recipient.
// - id: The ID of the token.
// - amount: The amount to set as the balance.
// Returns:
// - error: An error if the composite key creation or state update fails.
func setBalance(sdk kalpsdk.TransactionContextInterface, sender string, recipient string, id uint64, amount uint64) error {
	idString := strconv.FormatUint(uint64(id), 10)
	balanceKey, err := sdk.CreateCompositeKey(balancePrefix, []string{recipient, idString, sender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", balancePrefix, err)
	}
	return sdk.PutStateWithoutKYC(balanceKey, []byte(strconv.FormatUint(uint64(amount), 10)))
}
    
func removeBalance(sdk kalpsdk.TransactionContextInterface, sender string, ids []uint64, amounts []uint64) error {
    // Create a map to store the necessary funds for each token ID
    necessaryFunds := make(map[uint64]uint64)
    var err error
    
    // Iterate over the IDs and amounts to calculate the necessary funds
    for i := 0; i < len(amounts); i++ {
        // Add the amount to the necessary funds for the current token ID
        necessaryFunds[ids[i]], err = add(necessaryFunds[ids[i]], amounts[i])
        if err != nil {
            return err
        }
    }
    
    // Get the sorted keys of the necessary funds map
    necessaryFundsKeys := sortedKeys(necessaryFunds)
    
    // Iterate over the necessary funds keys
    for _, tokenId := range necessaryFundsKeys {
        // Get the needed amount for the current token ID
        neededAmount := necessaryFunds[tokenId]
        
        // Convert the token ID to a string
        idString := strconv.FormatUint(uint64(tokenId), 10)
        
        // Initialize the partial balance and self recipient key variables
        partialBalance := uint64(0)
        selfRecipientKeyNeedsToBeRemoved := false
        selfRecipientKey := ""
        
        // Get the balance iterator for the sender and token ID
        balanceIterator, err := sdk.GetStateByPartialCompositeKey(balancePrefix, []string{sender, idString})
        if err != nil {
            return fmt.Errorf("failed to get state for prefix %v: %v", balancePrefix, err)
        }
        defer balanceIterator.Close()
        
        // Iterate over the balance iterator
        for balanceIterator.HasNext() && partialBalance < neededAmount {
            // Get the next query response
            queryResponse, err := balanceIterator.Next()
            if err != nil {
                return fmt.Errorf("failed to get the next state for prefix %v: %v", balancePrefix, err)
            }
            
            // Parse the part balance amount from the query response value
            partBalAmount, _ := strconv.ParseUint(string(queryResponse.Value), 10, 64)
            
            // Add the part balance amount to the partial balance
            partialBalance, err = add(partialBalance, partBalAmount)
            if err != nil {
                return err
            }
            
            // Split the composite key into parts
            _, compositeKeyParts, err := sdk.SplitCompositeKey(queryResponse.Key)
            if err != nil {
                return err
            }
            
            // Check if the sender is the recipient
            if compositeKeyParts[2] == sender {
                // Set the self recipient key needs to be removed flag and store the self recipient key
                selfRecipientKeyNeedsToBeRemoved = true
                selfRecipientKey = queryResponse.Key
            } else {
                // Delete the state for the query response key
                err = sdk.DelStateWithoutKYC(queryResponse.Key)
                if err != nil {
                    return fmt.Errorf("failed to delete the state of %v: %v", queryResponse.Key, err)
                }
            }
        }
        
        // Check if the partial balance is less than the needed amount
        if partialBalance < neededAmount {
            return fmt.Errorf("sender has insufficient funds for token %v, needed funds: %v, available fund: %v", tokenId, neededAmount, partialBalance)
        } else if partialBalance > neededAmount {
            // Calculate the remainder
            remainder, err := sub(partialBalance, neededAmount)
            if err != nil {
                return err
            }
            
            // Check if the self recipient key needs to be removed
            if selfRecipientKeyNeedsToBeRemoved {
                // Set the balance for the sender and token ID
                err = setBalance(sdk, sender, sender, tokenId, remainder)
                if err != nil {
                    return err
                }
            } else {
                // Add the balance for the sender and token ID
                err = addBalance(sdk, sender, sender, tokenId, remainder)
                if err != nil {
                    return err
                }
            }
        } else {
            // Delete the self recipient key
            err = sdk.DelStateWithoutKYC(selfRecipientKey)
            if err != nil {
                return fmt.Errorf("failed to delete the state of %v: %v", selfRecipientKey, err)
            }
        }
    }
    
    return nil
}

 
func emitTransferSingle(sdk kalpsdk.TransactionContextInterface, transferSingleEvent TransferSingle) error {
	transferSingleEventJSON, err := json.Marshal(transferSingleEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = sdk.SetEvent("TransferSingle", transferSingleEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}
	return nil
}

func emitTransferBatch(sdk kalpsdk.TransactionContextInterface, transferBatchEvent TransferBatch) error {
	transferBatchEventJSON, err := json.Marshal(transferBatchEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = sdk.SetEvent("TransferBatch", transferBatchEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}
	return nil
}

func balanceOfHelper(sdk kalpsdk.TransactionContextInterface, account string, id uint64) (uint64, error) {
	if account == "0x0" {
		return 0, fmt.Errorf("balance query for the zero address")
	}
	idString := strconv.FormatUint(uint64(id), 10)
	var balance uint64
	balanceIterator, err := sdk.GetStateByPartialCompositeKey(balancePrefix, []string{account, idString})
	if err != nil {
		return 0, fmt.Errorf("failed to get state for prefix %v: %v", balancePrefix, err)
	}
	defer balanceIterator.Close()
	for balanceIterator.HasNext() {
		queryResponse, err := balanceIterator.Next()
		if err != nil {
			return 0, fmt.Errorf("failed to get the next state for prefix %v: %v", balancePrefix, err)
		}
		balAmount, _ := strconv.ParseUint(string(queryResponse.Value), 10, 64)
		balance, err = add(balance, balAmount)
		if err != nil {
			return 0, err
		}
	}
	return balance, nil
}