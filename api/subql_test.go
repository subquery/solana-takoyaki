package api

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/subquery/solana-takoyaki/backend/sqd"
	"github.com/subquery/solana-takoyaki/meta"
)

const BLOCK = 305_604_799

var BLOCK_BN = big.NewInt(BLOCK)

func compareAsJson(t *testing.T, expected, got interface{}, errorPrefix string) {
	aStr, _ := json.Marshal(expected)
	bStr, _ := json.Marshal(got)
	if string(aStr) != string(bStr) {
		// t.Errorf("%s Data missmatch", errorPrefix)
		t.Errorf("%s Mismatch\nexpected: %v\ngot: %v", errorPrefix, string(aStr), string(bStr))
	}
}

func TestFilterFullBlock(t *testing.T) {

	sqdUrl, err := sqd.GetSquidUrl(context.Background(), "solana-mainnet")
	if err != nil {
		t.Fatalf("Failed to get SQD url: %v", err)
	}

	apiService, err := NewSubqlApiService(meta.MAINNET, sqdUrl)
	if err != nil {
		t.Fatal(err)
	}

	res, err := apiService.FilterBlocks(context.Background(), BlockFilter{
		FromBlock: BLOCK_BN,
		ToBlock:   BLOCK_BN,
		Limit:     1,
		Instructions: []InstFilterQuery{
			{ProgramIds: []string{"675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8"}},
		},
		Logs: []LogFilterQuery{
			{ProgramIds: []string{"675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8"}},
		},
	})

	if err != nil {
		t.Fatalf("Failed to filter blocks: %v", err)
	}

	rawFullBlock, err := apiService.sqdClient.Query(context.Background(), sqd.SolanaRequest{
		Type:          "solana",
		FromBlock:     BLOCK,
		ToBlock:       BLOCK,
		Fields:        sqd.ALL_FIELDS,
		Transactions:  []sqd.TransactionRequest{{}}, // Empty item means no filter
		Instructions:  []sqd.InstructionRequest{{}},
		Rewards:       []sqd.RewardRequest{{}},
		TokenBalances: []sqd.TokenBalanceRequest{{}},
		Balances:      []sqd.BalancesRequest{{}},
		Logs:          []sqd.LogRequest{{}},
	})
	if err != nil {
		t.Fatalf("Failed to get full block to compare: %v", err)
	}

	fullBlock, err := sqd.TransformBlock(rawFullBlock[0])
	if err != nil {
		t.Fatalf("Failed to parse full block to compare: %v", err)
	}

	if len(res.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %v", len(res.Blocks))
	}

	if res.BlockRange[0] != BLOCK_BN {
		t.Errorf("Expected block range start %v, got %v", BLOCK, res.BlockRange[0].String())
	}

	if res.BlockRange[1] != BLOCK_BN {
		t.Errorf("Expected block range end %v, got %v", BLOCK, res.BlockRange[1].String())
	}

	block := res.Blocks[0]

	if block.BlockHeight != BLOCK {
		t.Errorf("Expected block height %v, got %v", BLOCK, block.BlockHeight)
	}

	// fmt.Printf("NUM TX expected %v, got %v\n", len(fullBlock.Transactions), len(block.Transactions))

	// for idx, tx := range fullBlock.Transactions {
	// 	if tx.Transaction.Signatures[0] == "2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS" {
	// 		fmt.Println("FULL BLOCK INDEX", idx)
	// 		break
	// 	}
	// }

	// for idx, tx := range block.Transactions {
	// 	if tx.Transaction.Signatures[0] == "2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS" {
	// 		fmt.Println("Filter BLOCK INDEX", idx)
	// 		break
	// 	}
	// }

	if len(fullBlock.Transactions) != len(block.Transactions) {
		t.Errorf("Expected %v transactions, got %v", len(fullBlock.Transactions), len(block.Transactions))
	}

	compareAsJson(t, fullBlock.Transactions[5], block.Transactions[5], "Transaction 2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS")

	// compareAsJson(t, fullBlock, block, fmt.Sprintf("Block %v", BLOCK))
}
