package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
    "github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

const (
	nameKey         = "name"
	symbolKey       = "symbol"
	decimalsKey     = "decimals"
	totalSupplyKey  = "totalSupply"
	allowancePrefix = "allowance"
)

type TokenERC20Contract struct {
	kalpsdk.Contract
}

// event represents the transfer event.
type event struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value int    `json:"value"`
}

// Initialize initializes the ERC20 token contract with the given name, symbol, and decimals.
// It can only be called by a client with the MSPID "mailabs".
// It sets the token name, symbol, and decimals in the contract state.
func (c *TokenERC20Contract) Initialize(ctx kalpsdk.TransactionContextInterface, name, symbol string, decimals int) (bool, error) {
	// Check if the client is authorized to initialize the contract.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "mailabs" {
		return false, fmt.Errorf("client is not authorized to initialize contract")
	}

	// Check if the contract options are already set.
	bytes, err := ctx.GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("failed to get Name: %v", err)
	}
	if bytes != nil {
		return false, fmt.Errorf("contract options are already set, client is not authorized to change them")
	}

	// Set the token name in the contract state.
	err = ctx.PutStateWithoutKYC(nameKey, []byte(name))
	if err != nil {
		return false, fmt.Errorf("failed to set token name: %v", err)
	}

	// Set the token symbol in the contract state.
	err = ctx.PutStateWithoutKYC(symbolKey, []byte(symbol))
	if err != nil {
		return false, fmt.Errorf("failed to set symbol: %v", err)
	}

	// Set the token decimals in the contract state.
	err = ctx.PutStateWithoutKYC(decimalsKey, []byte(strconv.Itoa(decimals)))
	if err != nil {
		return false, fmt.Errorf("failed to set decimals: %v", err)
	}

	return true, nil
}

// Mint mints new tokens and adds them to the minter's account.
// It can only be called by a client with the MSPID "mailabs".
// It updates the minter's account balance and the total token supply.
// It emits a Transfer event.
func (c *TokenERC20Contract) Mint(ctx kalpsdk.TransactionContextInterface, amount int) error {
	// Check if the contract is already initialized.
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Check if the client is authorized to mint new tokens.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "mailabs" {
		return fmt.Errorf("client is not authorized to mint new tokens")
	}

	// Get the minter's account ID.
	minter, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	// Check if the mint amount is valid.
	if amount <= 0 {
		return fmt.Errorf("mint amount must be a positive integer")
	}

	// Get the current balance of the minter's account.
	currentBalanceBytes, err := ctx.GetState(minter)
	if err != nil {
		return fmt.Errorf("failed to read minter account %s from world state: %v", minter, err)
	}

	var currentBalance int
	if currentBalanceBytes == nil {
		currentBalance = 0
	} else {
		currentBalance, _ = strconv.Atoi(string(currentBalanceBytes))
	}

	// Calculate the updated balance.
	updatedBalance, err := add(currentBalance, amount)
	if err != nil {
		return err
	}

	// Update the minter's account balance in the contract state.
	err = ctx.PutStateWithoutKYC(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// Get the current total token supply.
	totalSupplyBytes, err := ctx.GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	var totalSupply int
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes))
	}

	// Calculate the updated total token supply.
	totalSupply, err = add(totalSupply, amount)
	if err != nil {
		return err
	}

	// Update the total token supply in the contract state.
	err = ctx.PutStateWithoutKYC(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Emit a Transfer event.
	transferEvent := event{"0x0", minter, amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

// Burn burns tokens from the minter's account.
// It can only be called by a client with the MSPID "mailabs".
// It updates the minter's account balance and the total token supply.
// It emits a Transfer event.
func (c *TokenERC20Contract) Burn(ctx kalpsdk.TransactionContextInterface, amount int) error {
	// Check if the contract is already initialized.
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Check if the client is authorized to burn tokens.
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "mailabs" {
		return fmt.Errorf("client is not authorized to burn tokens")
	}

	// Get the minter's account ID.
	minter, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	// Check if the burn amount is valid.
	if amount <= 0 {
		return errors.New("burn amount must be a positive integer")
	}

	// Get the current balance of the minter's account.
	currentBalanceBytes, err := ctx.GetState(minter)
	if err != nil {
		return fmt.Errorf("failed to read minter account %s from world state: %v", minter, err)
	}

	if currentBalanceBytes == nil {
		return errors.New("the balance does not exist")
	}

	currentBalance, _ := strconv.Atoi(string(currentBalanceBytes))

	// Calculate the updated balance.
	updatedBalance, err := sub(currentBalance, amount)
	if err != nil {
		return err
	}

	// Update the minter's account balance in the contract state.
	err = ctx.PutStateWithoutKYC(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

	// Get the current total token supply.
	totalSupplyBytes, err := ctx.GetState(totalSupplyKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	if totalSupplyBytes == nil {
		return errors.New("totalSupply does not exist")
	}

	totalSupply, _ := strconv.Atoi(string(totalSupplyBytes))

	// Calculate the updated total token supply.
	totalSupply, err = sub(totalSupply, amount)
	if err != nil {
		return err
	}

	// Update the total token supply in the contract state.
	err = ctx.PutStateWithoutKYC(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

	// Emit a Transfer event.
	transferEvent := event{minter, "0x0", amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

// Transfer transfers tokens from the caller's account to the recipient's account.
// It can only be called by a client with the MSPID "mailabs".
// It updates the caller's account balance and the recipient's account balance.
// It emits a Transfer event.
func (c *TokenERC20Contract) Transfer(ctx kalpsdk.TransactionContextInterface, recipient string, amount int) error {
	// Check if the contract is already initialized.
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// Get the caller's account ID.
	clientID, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	// Transfer tokens from the caller's account to the recipient's account.
	err = transferHelper(ctx, clientID, recipient, amount)
	if err != nil {
		return fmt.Errorf("failed to transfer: %v", err)
	}

	// Emit a Transfer event.
	transferEvent := event{clientID, recipient, amount}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

// BalanceOf returns the balance of the specified account.
// It takes a transaction context and the account address as parameters.
// It returns the balance as an integer and an error if any.
func (c *TokenERC20Contract) BalanceOf(ctx kalpsdk.TransactionContextInterface, account string) (int, error) {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return 0, fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	balanceBytes, err := ctx.GetState(account)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("the account %s does not exist", account)
	}

	balance, _ := strconv.Atoi(string(balanceBytes))
	return balance, nil
}

// ClientAccountBalance returns the balance of the client's account.
// It checks if the contract is already initialized and if the client's account exists.
// If the account exists, it retrieves the balance from the world state and returns it.
// If the account does not exist, it returns an error.
func (c *TokenERC20Contract) ClientAccountBalance(ctx kalpsdk.TransactionContextInterface) (int, error) {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return 0, fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	clientID, err := ctx.GetUserID()
	if err != nil {
		return 0, fmt.Errorf("failed to get client id: %v", err)
	}

	balanceBytes, err := ctx.GetState(clientID)
	if err != nil {
		return 0, fmt.Errorf("failed to read from world state: %v", err)
	}
	if balanceBytes == nil {
		return 0, fmt.Errorf("the account %s does not exist", clientID)
	}

	balance, _ := strconv.Atoi(string(balanceBytes))
	return balance, nil
}

// ClientAccountID returns the client account ID associated with the token ERC20 contract.
// It checks if the contract is initialized, retrieves the client account ID, and returns it.
// If there is an error during the process, it returns an empty string and the error.
func (c *TokenERC20Contract) ClientAccountID(ctx kalpsdk.TransactionContextInterface) (string, error) {
    // Check if the contract is initialized
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return "", fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    // Get the client account ID
    clientAccountID, err := ctx.GetUserID()
    if err != nil {
        return "", fmt.Errorf("failed to get client id: %v", err)
    }

    return clientAccountID, nil
}

// TotalSupply returns the total supply of tokens in the ERC20 contract.
// It checks if the contract is already initialized and retrieves the total token supply from the state.
// If the total supply is not found in the state, it returns 0.
// It returns the total supply and any error encountered during the process.
func (c *TokenERC20Contract) TotalSupply(ctx kalpsdk.TransactionContextInterface) (int, error) {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return 0, fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	totalSupplyBytes, err := ctx.GetState(totalSupplyKey)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve total token supply: %v", err)
	}

	var totalSupply int
	if totalSupplyBytes == nil {
		totalSupply = 0
	} else {
		totalSupply, _ = strconv.Atoi(string(totalSupplyBytes))
	}

	return totalSupply, nil
}


// Approve allows the owner of the token to approve the spender to spend a certain amount of tokens on their behalf.
// It updates the allowance state of the smart contract and emits an "Approval" event.
// If the contract is not initialized, it returns an error.
// If there is an error while checking the contract initialization, getting the client ID, creating the composite key,
// updating the state, encoding the event, or setting the event, it returns an error.
// Otherwise, it returns nil.

func (c *TokenERC20Contract) Approve(ctx kalpsdk.TransactionContextInterface, spender string, value int) error {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	owner, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	allowanceKey, err := ctx.CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	err = ctx.PutStateWithoutKYC(allowanceKey, []byte(strconv.Itoa(value)))
	if err != nil {
		return fmt.Errorf("failed to update state of smart contract for key %s: %v", allowanceKey, err)
	}

	approvalEvent := event{owner, spender, value}
	approvalEventJSON, err := json.Marshal(approvalEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.SetEvent("Approval", approvalEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}
func (c *TokenERC20Contract) Allowance(ctx kalpsdk.TransactionContextInterface, owner string, spender string) (int, error) {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return 0, fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	allowanceKey, err := ctx.CreateCompositeKey(allowancePrefix, []string{owner, spender})
	if err != nil {
		return 0, fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	allowanceBytes, err := ctx.GetState(allowanceKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read allowance for %s from world state: %v", allowanceKey, err)
	}

	var allowance int
	if allowanceBytes == nil {
		allowance = 0
	} else {
		allowance, err = strconv.Atoi(string(allowanceBytes))
		if err != nil {
			return 0, fmt.Errorf("failed to convert allowance: %v", err)
		}
	}

	return allowance, nil
}
func (c *TokenERC20Contract) TransferFrom(ctx kalpsdk.TransactionContextInterface, from string, to string, value int) error {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	spender, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	allowanceKey, err := ctx.CreateCompositeKey(allowancePrefix, []string{from, spender})
	if err != nil {
		return fmt.Errorf("failed to create the composite key for prefix %s: %v", allowancePrefix, err)
	}

	currentAllowanceBytes, err := ctx.GetState(allowanceKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve the allowance for %s from world state: %v", allowanceKey, err)
	}

	var currentAllowance int
	currentAllowance, _ = strconv.Atoi(string(currentAllowanceBytes))

	if currentAllowance < value {
		return fmt.Errorf("spender does not have enough allowance for transfer")
	}

	err = transferHelper(ctx, from, to, value)
	if err != nil {
		return fmt.Errorf("failed to transfer: %v", err)
	}

	updatedAllowance, err := sub(currentAllowance, value)
	if err != nil {
		return err
	}

	err = ctx.PutStateWithoutKYC(allowanceKey, []byte(strconv.Itoa(updatedAllowance)))
	if err != nil {
		return err
	}

	transferEvent := event{from, to, value}
	transferEventJSON, err := json.Marshal(transferEvent)
	if err != nil {
		return fmt.Errorf("failed to obtain JSON encoding: %v", err)
	}
	err = ctx.SetEvent("Transfer", transferEventJSON)
	if err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

func checkInitialized(ctx kalpsdk.TransactionContextInterface) (bool, error) {
	tokenName, err := ctx.GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("failed to get token name: %v", err)
	}
	if tokenName == nil {
		return false, nil
	}
	return true, nil
}


func transferHelper(ctx kalpsdk.TransactionContextInterface, from string, to string, value int) error {
	if from == to {
		return fmt.Errorf("cannot transfer to and from same client account")
	}
	if value < 0 {
		return fmt.Errorf("transfer amount cannot be negative")
	}

	fromCurrentBalanceBytes, err := ctx.GetState(from)
	if err != nil {
		return fmt.Errorf("failed to read client account %s from world state: %v", from, err)
	}
	if fromCurrentBalanceBytes == nil {
		return fmt.Errorf("client account %s has no balance", from)
	}

	fromCurrentBalance, _ := strconv.Atoi(string(fromCurrentBalanceBytes))
	if fromCurrentBalance < value {
		return fmt.Errorf("client account %s has insufficient funds", from)
	}

	toCurrentBalanceBytes, err := ctx.GetState(to)
	if err != nil {
		return fmt.Errorf("failed to read recipient account %s from world state: %v", to, err)
	}

	var toCurrentBalance int
	if toCurrentBalanceBytes == nil {
		toCurrentBalance = 0
	} else {
		toCurrentBalance, _ = strconv.Atoi(string(toCurrentBalanceBytes))
	}

	fromUpdatedBalance, err := sub(fromCurrentBalance, value)
	if err != nil {
		return err
	}

	toUpdatedBalance, err := add(toCurrentBalance, value)
	if err != nil {
		return err
	}

	err = ctx.PutStateWithoutKYC(from, []byte(strconv.Itoa(fromUpdatedBalance)))
	if err != nil {
		return err
	}

	err = ctx.PutStateWithoutKYC(to, []byte(strconv.Itoa(toUpdatedBalance)))
	if err != nil {
		return err
	}

	return nil
}