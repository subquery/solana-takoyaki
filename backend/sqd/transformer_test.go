package sqd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/subquery/solana-takoyaki/meta"
	"github.com/subquery/solana-takoyaki/solana"
)

// https://solscan.io/block/327347682
const BLOCK = 305_604_799
const SLOT = 327_347_682
const RPC_ENDPOINT = "https://api.mainnet-beta.solana.com"

func compareAsJson(t *testing.T, expected, got interface{}, errorPrefix string) {
	aStr, _ := json.Marshal(expected)
	bStr, _ := json.Marshal(got)
	if string(aStr) != string(bStr) {
		t.Errorf("%s Mismatch\nexpected: %v\ngot: %v", errorPrefix, string(aStr), string(bStr))
	}
}

func TestTransforming(t *testing.T) {
	url, _ := GetSquidUrl(context.Background(), "solana-mainnet")
	client := NewClient(url, meta.MAINNET)

	rpcClient := rpc.NewWithHeaders(RPC_ENDPOINT, map[string]string{})

	maxVersion := uint64(0)

	// TODO parallelize RPC and SQD requests
	rpcBlock, err := rpcClient.GetBlockWithOpts(context.Background(), SLOT, &rpc.GetBlockOpts{
		Encoding:                       solanaGo.EncodingBase64,
		TransactionDetails:             "full",
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		t.Fatalf("Failed to get RPC block: %v", err)
	}

	res, err := client.Query(context.Background(), FULL_BLOCK_REQUEST)
	if err != nil {
		t.Fatalf("Failed to query SQD: %v", err)
	}

	transformedBlock, err := TransformBlock(res[0])
	if err != nil {
		t.Fatalf("Failed to transform block: %v", err)
	}

	compareBlock(t, *transformedBlock, *rpcBlock)
}

func TestTransformingSoldexer(t *testing.T) {
	client := NewSoldexerClient(SOLDEXER_URL)

	rpcClient := rpc.NewWithHeaders(RPC_ENDPOINT, map[string]string{})

	maxVersion := uint64(0)

	// TODO parallelize RPC and SQD requests
	rpcBlock, err := rpcClient.GetBlockWithOpts(context.Background(), SLOT, &rpc.GetBlockOpts{
		Encoding:                       solanaGo.EncodingBase64,
		TransactionDetails:             "full",
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		t.Fatalf("Failed to get RPC block: %v", err)
	}

	res, err := client.Query(context.Background(), SOLDEXER_FULL_BLOCK_REQUEST)
	if err != nil {
		t.Fatalf("Failed to query SQD: %v", err)
	}

	transformedBlock, err := TransformBlock(res[0])
	if err != nil {
		t.Fatalf("Failed to transform block: %v", err)
	}

	compareBlock(t, *transformedBlock, *rpcBlock)
}

func compareBlock(t *testing.T, sqdBlock solana.Block, rpcBlock rpc.GetBlockResult) {
	if sqdBlock.Blockhash != rpcBlock.Blockhash.String() {
		t.Errorf("Blockhash mismatch: %v != %v", sqdBlock.Blockhash, rpcBlock.Blockhash.String())
	}
	if sqdBlock.BlockHeight != *rpcBlock.BlockHeight {
		t.Errorf("Block height mismatch: %v != %v", sqdBlock.BlockHeight, *rpcBlock.BlockHeight)
	}
	if sqdBlock.PreviousBlockhash != rpcBlock.PreviousBlockhash.String() {
		t.Errorf("Previous block hash mismatch: %v != %v", sqdBlock.PreviousBlockhash, rpcBlock.PreviousBlockhash)
	}
	if sqdBlock.ParentSlot != rpcBlock.ParentSlot {
		t.Errorf("Parent slot mismatch: %v != %v", sqdBlock.ParentSlot, rpcBlock.ParentSlot)
	}
	if sqdBlock.BlockTime != rpcBlock.BlockTime.Time().Unix() {
		t.Errorf("Block time mismatch: %v != %v", sqdBlock.BlockTime, rpcBlock.BlockTime)
	}

	// SQD archive doesn't include voting transactions
	nonVotingTxs := []rpc.TransactionWithMeta{}
	for _, tx := range rpcBlock.Transactions {
		rpcTxInner, _ := solanaGo.TransactionFromBytes(tx.Transaction.GetBinary())
		if !IsVoteTx(rpcTxInner) {
			nonVotingTxs = append(nonVotingTxs, tx)
		}
	}
	if len(sqdBlock.Transactions) != len(nonVotingTxs) {
		t.Errorf("Tx count mismatch: %v != %v", len(sqdBlock.Transactions), len(nonVotingTxs))
	}

	// SQD doesnt include transactions from the vote program, this leads to different transaction indexes

	// https://solscan.io/tx/5LcWr1JK43dG7NNBXUw7qhouH4ewwoyo4idxgyf2BUtbzuKQpiv4Vn2w2c6sDWaejsmRzENLWdNxcA8mtaeXfS94
	compareTransactions(t, sqdBlock.Transactions[0], rpcBlock.Transactions[0], 0, []uint{0, 2})

	// https://solscan.io/tx/5vrB5e2Va47YcLvs2oVMeYfMdtmYJ2wJ7q2SQV9A7P8f1djzb9hMiwbPMajE5yaxPkYdch9QSzRzaiwHEvHgzCno
	// Failed transaction
	compareTransactions(t, sqdBlock.Transactions[2], rpcBlock.Transactions[3], 3, []uint{})

	// https://solscan.io/tx/2SB7fVzaUyU8knSbEa42c2BKQJXm1QPamtiFXKapX6YwLbbAm7dzezaGKyXfb6uGRH8a1xTeovSmWnbgav7jeKCS
	// Transaction with a program
	compareTransactions(t, sqdBlock.Transactions[5], rpcBlock.Transactions[16], 16, []uint{3, 4})

	if len(sqdBlock.Rewards) != len(rpcBlock.Rewards) {
		t.Errorf("Rewards count mismatch: %v != %v", len(sqdBlock.Rewards), len(rpcBlock.Rewards))
	}

	// This block only contains a single reward
	if len(rpcBlock.Rewards) > 0 {
		compareAsJson(t, rpcBlock.Rewards[0], sqdBlock.Rewards[0], "Reward 0")
	}
}

func compareTransactions(t *testing.T, sqdTx solana.Transaction, rpcTx rpc.TransactionWithMeta, index uint, checkInstructions []uint) {
	// RPC is missing both of these
	if sqdTx.Slot != SLOT {
		t.Errorf("Transaction Slot mismatch: %v != %v", sqdTx.Slot, SLOT)
	}
	// if sqdTx.BlockTime != rpcTx.BlockTime.Time().Unix() {
	// 	t.Errorf("Transaction BlockTime mismatch: %v != %v", sqdTx.BlockTime, rpcTx.BlockTime.Time().Unix())
	// }
	fmt.Printf("Checking tx sig: %v, idx: %v", sqdTx.Transaction.Signatures[0], index)

	rpcTxInner, err := solanaGo.TransactionFromBytes(rpcTx.Transaction.GetBinary())
	if err != nil {
		t.Fatalf("Failed to unmarshal RPC transaction: %v", err)
	}
	if len(sqdTx.Transaction.Signatures) != len(rpcTxInner.Signatures) {
		t.Errorf("Transaction[%v].Signatures count mismatch: %v != %v", index, len(sqdTx.Transaction.Signatures), len(rpcTxInner.Signatures))
	}
	// Most likely only one tx
	if sqdTx.Transaction.Signatures[0] != rpcTxInner.Signatures[0].String() {
		t.Errorf("Transaction[%v].Signatures mismatch: %v != %v", index, sqdTx.Transaction.Signatures[0], rpcTxInner.Signatures[0].String())
	}

	compareAsJson(t, rpcTx.Meta.Err, sqdTx.Meta.Err, fmt.Sprintf("Transaction[%v] Err", index))

	if sqdTx.Meta.Fee != rpcTx.Meta.Fee {
		t.Errorf("Transaction[%v].Fee mismatch: %v != %v", index, sqdTx.Meta.Fee, rpcTx.Meta.Fee)
	}

	if *sqdTx.Meta.ComputeUnitsConsumed != *rpcTx.Meta.ComputeUnitsConsumed {
		t.Errorf("Transaction[%v].ComputeUnitsConsumed mismatch: %v != %v", index, *sqdTx.Meta.ComputeUnitsConsumed, *rpcTx.Meta.ComputeUnitsConsumed)
	}

	compareAsJson(t, rpcTx.Meta.LoadedAddresses, sqdTx.Meta.LoadedAddresses, fmt.Sprintf("Transaction[%v] LoadedAddresses", index))
	compareAsJson(t, rpcTx.Meta.Rewards, sqdTx.Meta.Rewards, fmt.Sprintf("Transaction[%v] Rewards", index))

	// Only balance changes are included with SQD, not balances for all accounts in the tx
	sqdTokenBalIdx := 0
	for idx, preBal := range rpcTx.Meta.PreBalances {
		postBal := rpcTx.Meta.PostBalances[idx]

		if preBal != postBal {
			compareAsJson(t, preBal, sqdTx.Meta.PreBalances[sqdTokenBalIdx], fmt.Sprintf("Transaction[%v] Pre balance %v", index, idx))
			compareAsJson(t, postBal, sqdTx.Meta.PostBalances[sqdTokenBalIdx], fmt.Sprintf("Transaction[%v] Post balance %v", index, idx))
			sqdTokenBalIdx++
		}
	}

	if len(sqdTx.Meta.PreTokenBalances) != len(rpcTx.Meta.PreTokenBalances) {
		t.Errorf("Transaction[%v].PreTokenBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PreTokenBalances), len(rpcTx.Meta.PreTokenBalances))
	}
	if len(sqdTx.Meta.PostTokenBalances) != len(rpcTx.Meta.PostTokenBalances) {
		t.Errorf("Transaction[%v].PostTokenBalances count mismatch: %v != %v", index, len(sqdTx.Meta.PostTokenBalances), len(rpcTx.Meta.PostTokenBalances))
	}

	if len(rpcTx.Meta.PreTokenBalances) > 0 {
		compareAsJson(t, rpcTx.Meta.PreTokenBalances, sqdTx.Meta.PreTokenBalances, fmt.Sprintf("Transaction[%v] Pre token balance 0", index))
	}

	if len(rpcTx.Meta.PostTokenBalances) > 0 {
		compareAsJson(t, rpcTx.Meta.PostTokenBalances, sqdTx.Meta.PostTokenBalances, fmt.Sprintf("Transaction[%v] Post token balance 0", index))
	}

	if len(sqdTx.Transaction.Message.AccountKeys) != len(rpcTxInner.Message.AccountKeys) {
		t.Errorf("Transaction[%v].Message.AccountKeys count mismatch: %v != %v", index, len(sqdTx.Transaction.Message.AccountKeys), len(rpcTxInner.Message.AccountKeys))
	}
	// compareAsJson(t, rpcTx.Transaction.Message.AccountKeys, sqdTx.Transaction.Message.AccountKeys, fmt.Sprintf("Transaction[%v].Message.AccountKeys", index))

	if len(sqdTx.Transaction.Message.Instructions) != len(rpcTxInner.Message.Instructions) {
		t.Errorf("Transaction[%v].Message.Instructions count mismatch: %v != %v", index, len(sqdTx.Transaction.Message.Instructions), len(rpcTxInner.Message.Instructions))
	}
	for _, idx := range checkInstructions {
		compareInstructions(t, sqdTx.Transaction.Message.Instructions[idx], rpcTxInner.Message.Instructions[idx], idx)
	}

	if len(sqdTx.Meta.InnerInstructions) != len(rpcTx.Meta.InnerInstructions) {
		t.Errorf("Transaction[%v].Meta.InnerInstructions count mismatch: %v != %v", index, len(sqdTx.Meta.InnerInstructions), len(rpcTx.Meta.InnerInstructions))
	}

	for idx, innerInst := range sqdTx.Meta.InnerInstructions {
		rpcInnerInst := rpcTx.Meta.InnerInstructions[idx]
		if innerInst.Index != uint64(rpcInnerInst.Index) {
			t.Errorf("Transaction[%v].Meta.InnerInstructions[%v].Index mismatch: %v != %v", index, idx, innerInst.Index, rpcInnerInst.Index)
		}
		compareAsJson(t, rpcInnerInst.Instructions, innerInst.Instructions, fmt.Sprintf("Transaction[%v].Meta.InnerInstructions[%v]", index, idx))
	}

	// SQD only includes logs output by contracts, this excludes invoke, success and consumed compute units logs
	filteredLogs := []string{}
	for _, log := range rpcTx.Meta.LogMessages {
		if strings.HasPrefix(log, "Program log: ") || strings.HasPrefix(log, "Program data: ") || strings.HasPrefix(log, "Program returned:") {
			filteredLogs = append(filteredLogs, log)
		}
	}
	// if len(sqdTx.Meta.LogMessages) != len(filteredLogs) {
	// 	t.Errorf("Transaction[%v].Meta.LogMessages count mismatch: %v != %v", index, len(sqdTx.Meta.LogMessages), len(filteredLogs))
	// }
	// compareAsJson(t, sqdTx.Meta.LogMessages, filteredLogs, fmt.Sprintf("Transaction[%v].Meta.LogMessages", index))
}

func compareInstructions(t *testing.T, sqdInst solana.CompiledInstruction, rpcInst solanaGo.CompiledInstruction, index uint) {
	if sqdInst.ProgramIDIndex != rpcInst.ProgramIDIndex {
		t.Errorf("Instruction[%v].ProgramIDIndex mismatch: %v != %v", index, sqdInst.ProgramIDIndex, rpcInst.ProgramIDIndex)
	}
	if len(sqdInst.Accounts) != len(rpcInst.Accounts) {
		t.Errorf("Instruction[%v].Accounts count mismatch: %v != %v", index, len(sqdInst.Accounts), len(rpcInst.Accounts))
	}
	if sqdInst.Data != rpcInst.Data.String() {
		t.Errorf("Instruction[%v].Data mismatch: %v != %v", index, sqdInst.Data, rpcInst.Data.String())
	}
}

// Is part of github.com/gagliardetto/solana-go but not released at time of writing
func IsVoteTx(tx *solanaGo.Transaction) bool {
	// is vote if any of the instructions are of the vote program
	for _, inst := range tx.Message.Instructions {
		progKey, err := tx.ResolveProgramIDIndex(inst.ProgramIDIndex)
		if err == nil {
			if progKey.Equals(solanaGo.VoteProgramID) {
				return true
			}
		}
	}
	return false
}

func TestShiftDecimalPlacesLeft(t *testing.T) {
	shifted := shiftDecimalPlacesLeft(*big.NewFloat(101), 1)
	if big.NewFloat(10.1).Cmp(shifted) != 0 {
		t.Errorf("Expected 10.1, got %v", shifted)
	}

	shifted2 := shiftDecimalPlacesLeft(*big.NewFloat(987654321), 5)
	if big.NewFloat(9876.54321).Cmp(shifted2) != 0 {
		t.Errorf("Expected 9876.54321, got %v", shifted2)
	}
}
