package token

import (
    "encoding/json"
    "fmt"
    "github.com/p2eengineering/kalp-sdk-public/kalpsdk"
)

const balancePrefix = "balance"
const nftPrefix = "nft"
const approvalPrefix = "approval"
const nameKey = "name"
const symbolKey = "symbol"

type Nft struct {
    TokenId  string `json:"tokenId"`
    Owner    string `json:"owner"`
    TokenURI string `json:"tokenURI"`
    Approved string `json:"approved"`
}

type Approval struct {
    Owner    string `json:"owner"`
    Operator string `json:"operator"`
    Approved bool   `json:"approved"`
}

type Transfer struct {
    From    string `json:"from"`
    To      string `json:"to"`
    TokenId string `json:"tokenId"`
}

type TokenERC721Contract struct {
    kalpsdk.Contract
}

func _readNFT(ctx kalpsdk.TransactionContextInterface, tokenId string) (*Nft, error) {
    nftKey, err := ctx.CreateCompositeKey(nftPrefix, []string{tokenId})
    if err != nil {
        return nil, fmt.Errorf("failed to CreateCompositeKey %s: %v", tokenId, err)
    }

    nftBytes, err := ctx.GetState(nftKey)
    if err != nil {
        return nil, fmt.Errorf("failed to GetState %s: %v", tokenId, err)
    }

    nft := new(Nft)
    err = json.Unmarshal(nftBytes, nft)
    if err != nil {
        return nil, fmt.Errorf("failed to Unmarshal nftBytes: %v", err)
    }

    return nft, nil
}

func _nftExists(ctx kalpsdk.TransactionContextInterface, tokenId string) bool {
    nftKey, err := ctx.CreateCompositeKey(nftPrefix, []string{tokenId})
    if err != nil {
        panic("error creating CreateCompositeKey:" + err.Error())
    }

    nftBytes, err := ctx.GetState(nftKey)
    if err != nil {
        panic("error GetState nftBytes:" + err.Error())
    }

    return len(nftBytes) > 0
}

func (c *TokenERC721Contract) BalanceOf(ctx kalpsdk.TransactionContextInterface, owner string) int {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        panic("failed to check if contract is already initialized:" + err.Error())
    }
    if !initialized {
        panic("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    iterator, err := ctx.GetStateByPartialCompositeKey(balancePrefix, []string{owner})
    if err != nil {
        panic("Error creating asset chaincode:" + err.Error())
    }

    balance := 0
    for iterator.HasNext() {
        _, err := iterator.Next()
        if err != nil {
            return 0
        }
        balance++
    }
    return balance
}
