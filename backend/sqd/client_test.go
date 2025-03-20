package sqd

import (
	"fmt"
	"strings"
	"testing"
)

const ARCHIVE_URL = "https://v2.archive.subsquid.io/network/solana-mainnet"

var FULL_BLOCK_REQUEST = SolanaRequest{
	Type:      "solana",
	FromBlock: 305_604_799,
	ToBlock:   305_604_800,
	Fields: Fields{
		Instruction: map[string]bool{
			"programId": true,
			"data":      true,
			"accounts":  true,
			// "instructionAddress": true,
		},
		Transaction: map[string]bool{
			"accountKeys":                 true,
			"loadedAddresses":             true,
			"feePayer":                    true,
			"fee":                         true,
			"err":                         true,
			"signatures":                  true,
			"numReadonlySignedAccounts":   true,
			"numReadonlyUnsignedAccounts": true,
			"numRequiredSignatures":       true,
			"addressTableLookups":         true,
			// "recentBlockhash":             true, // Doesn't work, RPC returns an error
		},
		Log: map[string]bool{
			"kind": true,
		},
		Reward: map[string]bool{
			"rewardType":  true,
			"lamports":    true,
			"postBalance": true,
		},
		Block: map[string]bool{
			"parentHash": true,
			"slot":       true,
			"parentSlot": true,
			"timestamp":  true,
		},
		TokenBalance: map[string]bool{
			"preMint":       true,
			"preDecimals":   true,
			"preOwner":      true,
			"preAmount":     true,
			"postMint":      true,
			"postDecimals":  true,
			"postOwner":     true,
			"postAmount":    true,
			"postProgramId": true,
			"preProgramId":  true,
		},
		Balance: map[string]bool{
			"pre":  true,
			"post": true,
		},
	},
	Transactions: []TransactionRequest{{}}, // Empty item means no filter
	Instructions: []InstructionRequest{
		// {IsCommitted: true, Transaction: true},
		// {IsCommitted: false, Transaction: true},
		{},
	},
	Rewards:       []RewardRequest{{}},
	TokenBalances: []TokenBalanceRequest{{}},
	Balances:      []BalancesRequest{{}},
}

func TestGetDataSourceUrl(t *testing.T) {
	url, err := GetSquidUrl("solana-mainnet")
	if err != nil {
		t.Fatal(err)
	}

	if url != ARCHIVE_URL {
		t.Fatal("unexpected url")
	}
}

func TestGetCurrentHeight(t *testing.T) {
	client := Client{ARCHIVE_URL}

	height, err := client.CurrentHeight()
	if err != nil {
		t.Fatal(err)
	}

	if height < 305_604_799 {
		t.Fatal("unexpected height")
	}
}

func TestGetWorkerUrl(t *testing.T) {
	client := Client{ARCHIVE_URL}

	url, err := client.getWorkerUrl(305_604_799)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(url, "https://v2.archive.subsquid.io/network/solana-mainnet/") && strings.Contains(url, "/worker") {
		t.Fatal("unexpected url")
	}
}

func TestQuery(t *testing.T) {
	client := Client{ARCHIVE_URL}

	req := SolanaRequest{
		Type:      "solana",
		FromBlock: 305_604_799,
		ToBlock:   305_604_800,
		Fields: Fields{
			Instruction: map[string]bool{"programId": true},
			Transaction: map[string]bool{
				"accountKeys":         true,
				"addressTableLookups": true,
			},
			Log: map[string]bool{
				"kind": true,
			},
			Block: map[string]bool{
				"parentHash": true,
				"timestamp":  true,
				"slot":       true,
			},
		},
		Transactions: []TransactionRequest{{}}, // Empty item means no filter
		Instructions: []InstructionRequest{{}},
	}

	res, err := client.Query(req)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res[0].Header)

	// https://solscan.io/block/327347682
	block := res[0]
	if block.Header.Height != 305_604_799 {
		t.Errorf("Expected block height %v. Got: %v", 305_604_799, block.Header.Height)
	}

	if block.Header.Hash != "5FqMrgbiEmh22E9puyX4RV2EnARvwYBHgsTKYQ9a52Er" {
		t.Errorf("Expected block hash %v. Got: %v", "5FqMrgbiEmh22E9puyX4RV2EnARvwYBHgsTKYQ9a52Er", block.Header.Hash)
	}

	if len(block.Transactions) != 1_757 {
		t.Errorf("Expected %v transactions. Got %v transactions", 1_757, len(block.Transactions))
	}

	// TODO check this is correct
	if len(block.Instructions) != 3_568 {
		t.Errorf("Expected %v instructions. Got %v instructions", 3_568, len(block.Instructions))
	}

	// t.Fatal("Not enough validatiaon")
}
