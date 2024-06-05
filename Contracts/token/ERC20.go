package token

import (
    "encoding/json"
    "fmt"
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