package sqd

import (
	"context"
	"encoding/json"
	"fmt"

	// "fmt"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
)

// https://solscan.io/block/327347682
const BLOCK = 305_604_799
const SLOT = 327_347_682

func TestTransforming(t *testing.T) {
	url, _ := GetSquidUrl("solana-mainnet")
	client := NewClient(url)

	rpcClient := rpc.NewWithHeaders("https://api.mainnet-beta.solana.com", map[string]string{})

	maxVersion := uint64(0)

	// TODO parallelize RPC and SQD requests
	rpcBlock, err := rpcClient.GetParsedBlockWithOpts(context.Background(), SLOT, &rpc.GetBlockOpts{
		// Encoding:                       "jsonParsed",
		TransactionDetails:             "full",
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		t.Fatalf("Failed to get RPC block: %v", err)
	}

	res, err := client.Query(FULL_BLOCK_REQUEST)
	if err != nil {
		t.Fatalf("Failed to query SQD: %v", err)
	}

	transformedBlock, err := TansformBlock(res[0])
	if err != nil {
		t.Fatalf("Failed to transform block: %v", err)
	}

	if !transformedBlock.Blockhash.Equals(rpcBlock.Blockhash) {
		t.Errorf("Blockhash mismatch: %v != %v", transformedBlock.Blockhash, rpcBlock.Blockhash)
	}
	if *transformedBlock.BlockHeight != *rpcBlock.BlockHeight {
		t.Errorf("Block height mismatch: %v != %v", *transformedBlock.BlockHeight, *rpcBlock.BlockHeight)
	}
	if transformedBlock.PreviousBlockhash != rpcBlock.PreviousBlockhash {
		t.Errorf("Previous block hash mismatch: %v != %v", transformedBlock.PreviousBlockhash, rpcBlock.PreviousBlockhash)
	}
	if transformedBlock.ParentSlot != rpcBlock.ParentSlot {
		t.Errorf("Parent slot mismatch: %v != %v", transformedBlock.ParentSlot, rpcBlock.ParentSlot)
	}
	if transformedBlock.BlockTime.Time() != rpcBlock.BlockTime.Time() {
		t.Errorf("Block time mismatch: %v != %v", transformedBlock.BlockTime, rpcBlock.BlockTime)
	}

	if len(transformedBlock.Transactions) != len(rpcBlock.Transactions) {
		t.Errorf("Tx count mismatch: %v != %v", len(transformedBlock.Transactions), len(rpcBlock.Transactions))
	}

	// SQD doesnt include transactions from the vote program, this leads to different transaction indexes

	// https://solscan.io/tx/5LcWr1JK43dG7NNBXUw7qhouH4ewwoyo4idxgyf2BUtbzuKQpiv4Vn2w2c6sDWaejsmRzENLWdNxcA8mtaeXfS94
	// compareTransactions(t, transformedBlock.Transactions[0], rpcBlock.Transactions[0], 0, []uint{0, 2})

	// https://solscan.io/tx/5vrB5e2Va47YcLvs2oVMeYfMdtmYJ2wJ7q2SQV9A7P8f1djzb9hMiwbPMajE5yaxPkYdch9QSzRzaiwHEvHgzCno
	// Failed transaction
	// compareTransactions(t, transformedBlock.Transactions[2], rpcBlock.Transactions[3], 3, []uint{})

	//https://solscan.io/tx/2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS
	// Transaction with a program
	compareTransactions(t, transformedBlock.Transactions[5], rpcBlock.Transactions[16], 16, []uint{3, 4})

	for i, tx := range rpcBlock.Transactions {
		if tx.Transaction.Signatures[0].String() == "2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS" {
			fmt.Println("Found transaction 5 at index", i)
		}
	}

	if len(transformedBlock.Rewards) != len(rpcBlock.Rewards) {
		t.Errorf("Rewards count mismatch: %v != %v", len(transformedBlock.Rewards), len(rpcBlock.Rewards))
	}

	// This block only contains a single reward
	if len(rpcBlock.Rewards) > 0 {
		rpcRewardStr, _ := json.Marshal(rpcBlock.Rewards[0])
		sqdRewardStr, _ := json.Marshal(transformedBlock.Rewards[0])
		if string(rpcRewardStr) != string(sqdRewardStr) {
			t.Errorf("Reward 0 mismatch expected: %v\ngot: %v", string(rpcRewardStr), string(sqdRewardStr))
		}
	}

}

func compareTransactions(t *testing.T, sqdTx, rpcTx rpc.ParsedTransactionWithMeta, index uint, checkInstructions []uint) {
	// RPC is missing both of these
	// if sqdTx.Slot != rpcTx.Slot {
	// 	t.Errorf("Transaction Slot mismatch: %v != %v", sqdTx.Slot, rpcTx.Slot)
	// }
	// if sqdTx.BlockTime.Time() != rpcTx.BlockTime.Time() {
	// 	t.Errorf("Transaction BlockTime mismatch: %v != %v", sqdTx.BlockTime, rpcTx.BlockTime)
	// }
	fmt.Println("Checking tx", sqdTx.Transaction.Signatures[0].String())
	if len(sqdTx.Transaction.Signatures) != len(rpcTx.Transaction.Signatures) {
		t.Errorf("Transaction[%v].Signatures count mismatch: %v != %v", index, len(sqdTx.Transaction.Signatures), len(rpcTx.Transaction.Signatures))
	}
	// Most likely only one tx
	if !sqdTx.Transaction.Signatures[0].Equals(rpcTx.Transaction.Signatures[0]) {
		t.Errorf("Transaction[%v].Signatures mismatch: %v != %v", index, sqdTx.Transaction.Signatures[0].String(), rpcTx.Transaction.Signatures[0].String())
	}

	// TODO check doesn't seem to work
	if sqdTx.Meta.Err != rpcTx.Meta.Err {
		t.Errorf("Transaction[%v].Error mismatch: %v != %v", index, sqdTx.Meta.Err, rpcTx.Meta.Err)
	}
	if sqdTx.Meta.Fee != rpcTx.Meta.Fee {
		t.Errorf("Transaction[%v].Fee mismatch: %v != %v", index, sqdTx.Meta.Fee, rpcTx.Meta.Fee)
	}
	if len(sqdTx.Meta.PreBalances) != len(rpcTx.Meta.PreBalances) {
		t.Errorf("Transaction[%v].PreBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PreBalances), len(rpcTx.Meta.PreBalances))
	}
	if len(sqdTx.Meta.PostBalances) != len(rpcTx.Meta.PostBalances) {
		t.Errorf("Transaction[%v].PostBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PostBalances), len(rpcTx.Meta.PostBalances))
	}

	if len(sqdTx.Meta.PreTokenBalances) != len(rpcTx.Meta.PreTokenBalances) {
		t.Errorf("Transaction[%v].PreTokenBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PreTokenBalances), len(rpcTx.Meta.PreTokenBalances))
	}
	if len(sqdTx.Meta.PostTokenBalances) != len(rpcTx.Meta.PostTokenBalances) {
		t.Errorf("Transaction[%v].PostTokenBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PostTokenBalances), len(rpcTx.Meta.PostTokenBalances))
	}

	// TODO probably needs ordering
	if len(rpcTx.Meta.PreTokenBalances) > 0 {
		sqdTbStr, _ := json.Marshal(sqdTx.Meta.PreTokenBalances)
		rpcTbStr, _ := json.Marshal(rpcTx.Meta.PreTokenBalances)
		if string(sqdTbStr) != string(rpcTbStr) {
			t.Errorf("Transaction[%v] Pre token balance 0 mismatch\n\texpected: %v\n\tgot: %v", index, string(rpcTbStr), string(sqdTbStr))
		}
	}

	// TODO probably needs ordering
	if len(rpcTx.Meta.PostTokenBalances) > 0 {
		sqdTbStr, _ := json.Marshal(sqdTx.Meta.PostTokenBalances)
		rpcTbStr, _ := json.Marshal(rpcTx.Meta.PostTokenBalances)
		if string(sqdTbStr) != string(rpcTbStr) {
			t.Errorf("Transaction[%v] Post token balance 0 mismatch\n\texpected: %v\n\tgot: %v", index, string(rpcTbStr), string(sqdTbStr))
		}
	}

	if len(sqdTx.Transaction.Message.AccountKeys) != len(rpcTx.Transaction.Message.AccountKeys) {
		t.Errorf("Transaction[%v].Message.AccountKeys count mismatch: %v != %v", index, len(sqdTx.Transaction.Message.AccountKeys), len(rpcTx.Transaction.Message.AccountKeys))
	}

	sqdString, _ := json.Marshal(sqdTx.Transaction.Message.AccountKeys)
	rpcString, _ := json.Marshal(rpcTx.Transaction.Message.AccountKeys)
	if string(sqdString) != string(rpcString) {
		t.Errorf("Transaction[%v].Message.AccountKeys mismatch\n\texpected: %v\n\tgot: %v", index, string(rpcString), string(sqdString))
	}

	if sqdTx.Transaction.Message.RecentBlockHash != rpcTx.Transaction.Message.RecentBlockHash {
		t.Errorf("Transaction[%v].Message.RecentBlockHash mismatch: %v != %v", index, sqdTx.Transaction.Message.RecentBlockHash, rpcTx.Transaction.Message.RecentBlockHash)
	}

	if len(sqdTx.Transaction.Message.Instructions) != len(rpcTx.Transaction.Message.Instructions) {
		t.Errorf("Transaction[%v].Message.Instructions count mismatch: %v != %v", index, len(sqdTx.Transaction.Message.Instructions), len(rpcTx.Transaction.Message.Instructions))
	}

	for _, idx := range checkInstructions {
		compareInstructions(t, sqdTx.Transaction.Message.Instructions[idx], rpcTx.Transaction.Message.Instructions[idx], idx)
	}
	// TODO find instruction that failed
	// TODO find instruction that is parsed
	// TODO find instruction that has a program set

	if len(sqdTx.Meta.InnerInstructions) != len(rpcTx.Meta.InnerInstructions) {
		t.Errorf("Transaction[%v].Meta.InnerInstructions count mismatch: %v != %v", index, len(sqdTx.Meta.InnerInstructions), len(rpcTx.Meta.InnerInstructions))
	}

	// TODO compare inner instructions
}

func compareInstructions(t *testing.T, sqdInst, rpcInst *rpc.ParsedInstruction, index uint) {
	if sqdInst.Program != rpcInst.Program {
		t.Errorf("Instruction[%v].Program mismatch: %v != %v", index, sqdInst.Program, rpcInst.Program)
	}
	if sqdInst.ProgramId != rpcInst.ProgramId {
		t.Errorf("Instruction[%v].ProgramId mismatch: %v != %v", index, sqdInst.ProgramId, rpcInst.ProgramId)
	}
	if len(sqdInst.Accounts) != len(rpcInst.Accounts) {
		t.Errorf("Instruction[%v].Accounts count mismatch: %v != %v", index, len(sqdInst.Accounts), len(rpcInst.Accounts))
	}
	if sqdInst.Data.String() != rpcInst.Data.String() {
		t.Errorf("Instruction[%v].Data mismatch: %v != %v", index, sqdInst.Data.String(), rpcInst.Data.String())
	}
	if sqdInst.StackHeight != rpcInst.StackHeight {
		t.Errorf("Instruction[%v].StackHeight mismatch: %v != %v", index, sqdInst.StackHeight, rpcInst.StackHeight)
	}
	// if sqdInst.Parsed != rpcInst.Parsed {
	// 	t.Errorf("Parsed mismatch: %v != %v", sqdInst.Parsed, rpcInst.Parsed)
	// }
}
