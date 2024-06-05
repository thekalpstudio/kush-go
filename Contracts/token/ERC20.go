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
