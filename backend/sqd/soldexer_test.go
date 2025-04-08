package sqd

import (
	"context"
	"testing"
)

const SOLDEXER_URL = "https://portal.sqd.dev/datasets/solana-beta"

var SOLDEXER_FULL_BLOCK_REQUEST = SolanaRequest{
	Type:          "solana",
	FromBlock:     327_347_682, // Slot
	ToBlock:       327_347_682, // Slot
	Fields:        ALL_SOLDEXER_FIELDS,
	Transactions:  []TransactionRequest{{}}, // Empty item means no filter
	Instructions:  []InstructionRequest{{}},
	Rewards:       []RewardRequest{{}},
	TokenBalances: []TokenBalanceRequest{{}},
	Balances:      []BalancesRequest{{}},
	Logs:          []LogRequest{{}},
}

func TestSoldexerGetCurrentHeight(t *testing.T) {
	client := NewSoldexerClient(SOLDEXER_URL)

	ctx := context.Background()

	height, err := client.CurrentHeight(ctx)
	if err != nil {
		t.Fatalf("Failed to get current height: %v", err)
	}

	if height == 0 {
		t.Fatal("Expected non-zero height")
	}
}

func TestSoldexerGetMeta(t *testing.T) {
	client := NewSoldexerClient(SOLDEXER_URL)

	ctx := context.Background()

	meta, err := client.Metadata(ctx)
	if err != nil {
		t.Fatalf("Failed to get current height: %v", err)
	}

	if meta.StartBlock <= 0 {
		t.Fatal("Expected non-zero start block")
	}

	// if meta.Aliases[0] != "solana-mainnet" {
	// 	t.Fatal("Expected non-zero height")
	// }
}

func TestSoldexerQuery(t *testing.T) {
	client := NewSoldexerClient(SOLDEXER_URL)

	req := SolanaRequest{
		Type:      "solana",
		FromBlock: 327_347_682,
		ToBlock:   327_347_682,
		Fields:    ALL_SOLDEXER_FIELDS, /*Fields{
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
			},
		},*/
		Transactions: []TransactionRequest{{}}, // Empty item means no filter
		Instructions: []InstructionRequest{{}},
	}

	res, err := client.Query(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// https://solscan.io/block/327347682 (slot 327_347_682 = block 305_604_799)
	block := res[0]
	if block.Header.Height != 305_604_799 {
		t.Errorf("Expected block height %v. Got: %v", 305_604_799, block.Header.Height)
	}

	if block.Header.Hash != "5FqMrgbiEmh22E9puyX4RV2EnARvwYBHgsTKYQ9a52Er" {
		t.Errorf("Expected block hash %v. Got: %v", "5FqMrgbiEmh22E9puyX4RV2EnARvwYBHgsTKYQ9a52Er", block.Header.Hash)
	}

	// Doesn't include voting program transactions
	if len(block.Transactions) != 441 {
		t.Errorf("Expected %v transactions. Got %v transactions", 441, len(block.Transactions))
	}

	// TODO check this is correct
	if len(block.Instructions) != 3_568 {
		t.Errorf("Expected %v instructions. Got %v instructions", 3_568, len(block.Instructions))
	}
}
