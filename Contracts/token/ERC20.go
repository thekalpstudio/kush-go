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


type event struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value int    `json:"value"`
}

func (c *TokenERC20Contract) Initialize(ctx kalpsdk.TransactionContextInterface, name, symbol string, decimals int) (bool, error) {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "mailabs" {
		return false, fmt.Errorf("client is not authorized to initialize contract")
	}

	bytes, err := ctx.GetState(nameKey)
	if err != nil {
		return false, fmt.Errorf("failed to get Name: %v", err)
	}
	if bytes != nil {
		return false, fmt.Errorf("contract options are already set, client is not authorized to change them")
	}

	err = ctx.PutStateWithoutKYC(nameKey, []byte(name))
	if err != nil {
		return false, fmt.Errorf("failed to set token name: %v", err)
	}

	err = ctx.PutStateWithoutKYC(symbolKey, []byte(symbol))
	if err != nil {
		return false, fmt.Errorf("failed to set symbol: %v", err)
	}

	err = ctx.PutStateWithoutKYC(decimalsKey, []byte(strconv.Itoa(decimals)))
	if err != nil {
		return false, fmt.Errorf("failed to set decimals: %v", err)
	}

	return true, nil
}

func (c *TokenERC20Contract) Mint(ctx kalpsdk.TransactionContextInterface, amount int) error {
	initialized, err := checkInitialized(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if contract is already initialized: %v", err)
	}
	if !initialized {
		return fmt.Errorf("contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != "mailabs" {
		return fmt.Errorf("client is not authorized to mint new tokens")
	}

	minter, err := ctx.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if amount <= 0 {
		return fmt.Errorf("mint amount must be a positive integer")
	}

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

	updatedBalance, err := add(currentBalance, amount)
	if err != nil {
		return err
	}

	err = ctx.PutStateWithoutKYC(minter, []byte(strconv.Itoa(updatedBalance)))
	if err != nil {
		return err
	}

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

	totalSupply, err = add(totalSupply, amount)
	if err != nil {
		return err
	}

	err = ctx.PutStateWithoutKYC(totalSupplyKey, []byte(strconv.Itoa(totalSupply)))
	if err != nil {
		return err
	}

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