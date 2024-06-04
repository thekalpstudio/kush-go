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
func (c *TokenERC721Contract) OwnerOf(ctx kalpsdk.TransactionContextInterface, tokenId string) (string, error) {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return "", fmt.Errorf("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    nft, err := _readNFT(ctx, tokenId)
    if err != nil {
        return "", fmt.Errorf("could not process OwnerOf for tokenId: %w", err)
    }

    return nft.Owner, nil
}

func (c *TokenERC721Contract) Approve(ctx kalpsdk.TransactionContextInterface, operator string, tokenId string) (bool, error) {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return false, fmt.Errorf("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    sender, err := ctx.GetUserID()
    if err != nil {
        return false, fmt.Errorf("failed to GetClientIdentity: %v", err)
    }

    nft, err := _readNFT(ctx, tokenId)
    if err != nil {
        return false, fmt.Errorf("failed to _readNFT: %v", err)
    }

    owner := nft.Owner
    operatorApproval, err := c.IsApprovedForAll(ctx, owner, sender)
    if err != nil {
        return false, fmt.Errorf("failed to get IsApprovedForAll: %v", err)
    }
    if owner != sender && !operatorApproval {
        return false, fmt.Errorf("the sender is not the current owner nor an authorized operator")
    }

    nft.Approved = operator
    nftKey, err := ctx.CreateCompositeKey(nftPrefix, []string{tokenId})
    if err != nil {
        return false, fmt.Errorf("failed to CreateCompositeKey %s: %v", nftKey, err)
    }

    nftBytes, err := json.Marshal(nft)
    if err != nil {
        return false, fmt.Errorf("failed to marshal nftBytes: %v", err)
    }

    err = ctx.PutStateWithoutKYC(nftKey, nftBytes)
    if err != nil {
        return false, fmt.Errorf("failed to PutState for nftKey: %v", err)
    }

    return true, nil
}

func (c *TokenERC721Contract) SetApprovalForAll(ctx kalpsdk.TransactionContextInterface, operator string, approved bool) (bool, error) {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return false, fmt.Errorf("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    sender, err := ctx.GetUserID()
    if err != nil {
        return false, fmt.Errorf("failed to GetClientIdentity: %v", err)
    }

    nftApproval := new(Approval)
    nftApproval.Owner = sender
    nftApproval.Operator = operator
    nftApproval.Approved = approved

    approvalKey, err := ctx.CreateCompositeKey(approvalPrefix, []string{sender, operator})
    if err != nil {
        return false, fmt.Errorf("failed to CreateCompositeKey: %v", err)
    }

    approvalBytes, err := json.Marshal(nftApproval)
    if err != nil {
        return false, fmt.Errorf("failed to marshal approvalBytes: %v", err)
    }

    err = ctx.PutStateWithoutKYC(approvalKey, approvalBytes)
    if err != nil {
        return false, fmt.Errorf("failed to PutState approvalBytes: %v", err)
    }

    err = ctx.SetEvent("ApprovalForAll", approvalBytes)
    if err != nil {
        return false, fmt.Errorf("failed to SetEvent ApprovalForAll: %v", err)
    }

    return true, nil
}


func (c *TokenERC721Contract) IsApprovedForAll(ctx kalpsdk.TransactionContextInterface, owner string, operator string) (bool, error) {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return false, fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return false, fmt.Errorf("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    approvalKey, err := ctx.CreateCompositeKey(approvalPrefix, []string{owner, operator})
    if err != nil {
        return false, fmt.Errorf("failed to CreateCompositeKey: %v", err)
    }
    approvalBytes, err := ctx.GetState(approvalKey)
    if err != nil {
        return false, fmt.Errorf("failed to GetState approvalBytes %s: %v", approvalBytes, err)
    }

    if len(approvalBytes) < 1 {
        return false, nil
    }

    approval := new(Approval)
    err = json.Unmarshal(approvalBytes, approval)
    if err != nil {
        return false, fmt.Errorf("failed to Unmarshal: %v, string %s", err, string(approvalBytes))
    }

    return approval.Approved, nil
}

func (c *TokenERC721Contract) GetApproved(ctx kalpsdk.TransactionContextInterface, tokenId string) (string, error) {
    initialized, err := checkInitialized(ctx)
    if err != nil {
        return "false", fmt.Errorf("failed to check if contract is already initialized: %v", err)
    }
    if !initialized {
        return "false", fmt.Errorf("Contract options need to be set before calling any function, call Initialize() to initialize contract")
    }

    nft, err := _readNFT(ctx, tokenId)
    if err != nil {
        return "false", fmt.Errorf("failed GetApproved for tokenId : %v", err)
    }
    return nft.Approved, nil
}