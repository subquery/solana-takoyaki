package sqd

import (
	"context"
	"encoding/json"
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

	sqdBlock := res[0]

	transformedBlock, err := TansformBlock(sqdBlock)

	if !transformedBlock.Blockhash.Equals(rpcBlock.Blockhash) {
		t.Errorf("Blockhash mismatch: %v != %v", transformedBlock.Blockhash, rpcBlock.Blockhash)
	}
	if transformedBlock.BlockHeight != rpcBlock.BlockHeight {
		t.Errorf("Block height mismatch: %v != %v", transformedBlock.BlockHeight, rpcBlock.BlockHeight)
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

	compareTransactions(t, transformedBlock.Transactions[0], rpcBlock.Transactions[0])
	// TODO compare failed transaction

	if len(transformedBlock.Rewards) != len(rpcBlock.Rewards) {
		t.Errorf("Rewards count mismatch: %v != %v", len(transformedBlock.Rewards), len(rpcBlock.Rewards))
	}

	// This block only contains a single reward
	rpcRewardStr, _ := json.Marshal(rpcBlock.Rewards[0])
	sqdRewardStr, _ := json.Marshal(transformedBlock.Rewards[0])
	if string(rpcRewardStr) != string(sqdRewardStr) {
		t.Errorf("Reward 0 mismatch expected: %v\ngot: %v", string(rpcRewardStr), string(sqdRewardStr))
	}

}

func compareTransactions(t *testing.T, sqdTx rpc.ParsedTransactionWithMeta, rpcTx rpc.ParsedTransactionWithMeta) {
	// RPC is missing both of these
	// if sqdTx.Slot != rpcTx.Slot {
	// 	t.Errorf("Transaction Slot mismatch: %v != %v", sqdTx.Slot, rpcTx.Slot)
	// }
	// if sqdTx.BlockTime.Time() != rpcTx.BlockTime.Time() {
	// 	t.Errorf("Transaction BlockTime mismatch: %v != %v", sqdTx.BlockTime, rpcTx.BlockTime)
	// }
	if sqdTx.Meta.Err != rpcTx.Meta.Err {
		t.Errorf("Transaction Error mismatch: %v != %v", sqdTx.Meta.Err, rpcTx.Meta.Err)
	}
	if sqdTx.Meta.Fee != rpcTx.Meta.Fee {
		t.Errorf("Transaction Fee mismatch: %v != %v", sqdTx.Meta.Fee, rpcTx.Meta.Fee)
	}
	if len(sqdTx.Meta.PreBalances) != len(rpcTx.Meta.PreBalances) {
		t.Errorf("Transaction PreBalances count mismatch: %v != %v", len(sqdTx.Meta.PreBalances), len(rpcTx.Meta.PreBalances))
	}
	if len(sqdTx.Meta.PostBalances) != len(rpcTx.Meta.PostBalances) {
		t.Errorf("Transaction PostBalances count mismatch: %v != %v", len(sqdTx.Meta.PostBalances), len(rpcTx.Meta.PostBalances))
	}
	// if len(sqdTx.Meta.Signatures) != len(rpcTx.Meta.Signatures) {
	// 	t.Errorf("Signatures count mismatch: %v != %v", len(sqdTx.Meta.Signatures), len(rpcTx.Meta.Signatures))
	// }
	// if len(sqdTx.Meta.SignaturesV2) != len(rpcTx.Meta.SignaturesV2) {
	// 	t.Errorf("SignaturesV2 count mismatch: %v != %v", len(sqdTx.Meta.SignaturesV2), len(rpcTx.Meta.SignaturesV2))
	// }
	// if sqdTx.Meta.Transaction.Message.Header.NumRequiredSignatures != rpcTx.Meta.Transaction.Message.Header.NumRequiredSignatures {
	// 	t.Errorf("NumRequiredSignatures mismatch: %v != %v", sqdTx.Meta.Transaction.Message.Header.NumRequiredSignatures, rpcTx.Meta.Transaction.Message.Header.NumRequiredSignatures)
	// }
	// if sqdTx.Meta.Transaction.Message.Header.NumReadonlySignedAccounts != rpcTx.Meta.Transaction.Message.Header.NumReadonlySignedAccounts {
	// 	t.Errorf("NumReadonlySignedAccounts mismatch: %v != %v", sqdTx.Meta.Transaction.Message.Header.NumReadonlySignedAccounts, rpcTx.Meta.Transaction.Message.Header.NumReadonlySignedAccounts)
	// }
	// if sqdTx.Meta.Transaction.Message.Header.NumReadonlyUnsignedAccounts != rpcTx.Meta.Transaction.Message.Header.NumReadonlyUnsignedAccounts {
	// 	t.Errorf("NumReadonlyUnsignedAccounts mismatch")

	// }
}
