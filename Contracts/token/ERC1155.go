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
