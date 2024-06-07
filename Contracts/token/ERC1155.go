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