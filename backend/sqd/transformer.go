package sqd

import (
	"strconv"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TansformBlock(sqdBlock SolanaBlockResponse) (out *rpc.GetParsedBlockResult, err error) {

	timestamp := solana.UnixTimeSeconds(sqdBlock.Header.Timestamp)

	out = &rpc.GetParsedBlockResult{
		BlockHeight:       &sqdBlock.Header.Height,
		Blockhash:         solana.MustHashFromBase58(sqdBlock.Header.Hash),
		PreviousBlockhash: solana.MustHashFromBase58(sqdBlock.Header.ParentHash),
		ParentSlot:        sqdBlock.Header.ParentSlot,
		BlockTime:         &timestamp,
		// NumRewardPartitions: 0, // TODO
	}

	// Transform Transactions
	if out.Transactions == nil {
		out.Transactions = []rpc.ParsedTransactionWithMeta{}
	}
	for _, tx := range sqdBlock.Transactions {
		solanaTx, err := TransformTransaction(tx, sqdBlock.Header)
		if err != nil {
			return nil, err
		}
		out.Transactions = append(out.Transactions, *solanaTx)
	}

	// Transform Rewards
	if out.Rewards == nil {
		out.Rewards = []rpc.BlockReward{}
	}
	for _, reward := range sqdBlock.Rewards {
		solanaReward, err := TransformReward(reward)
		if err != nil {
			return nil, err
		}
		out.Rewards = append(out.Rewards, *solanaReward)
	}

	// Signatures
	if out.Signatures == nil {
		out.Signatures = []solana.Signature{}
	}
	// TODO fill signatures

	return out, nil
}

func TransformTransaction(in TransactionResponse, header blockHeader) (out *rpc.ParsedTransactionWithMeta, err error) {
	timestamp := solana.UnixTimeSeconds(header.Timestamp)

	out = &rpc.ParsedTransactionWithMeta{
		Slot:      header.Slot,
		BlockTime: &timestamp,
		Meta: &rpc.ParsedTransactionMeta{
			Err:          in.Transaction.Err,
			Fee:          in.Transaction.Fee,
			PreBalances:  []uint64{}, // TODO
			PostBalances: []uint64{}, // TODO
			// InnerInstructions: []rpc.InnerInstruction{}, // TODO
			PreTokenBalances:  []rpc.TokenBalance{}, // TODO
			PostTokenBalances: []rpc.TokenBalance{}, // TODO
			LogMessages:       []string{},           // TODO
			// Status:            nil,                      // TODO
			// Rewards:           []rpc.BlockReward{},      // TODO
			// LoadedAddresses:   TransformLoadedAddresses(in.Transaction.LoadedAddresses),
			// ReturnData:        rpc.ReturnData{
			// ProgramId: nil, // TODO
			// Data: nil, // TODO
			// },
			// ComputeUnitsConsumed: &in.Transaction.ComputeUnitsConsumed,
		},
		// Version:     in.Transaction.Version,
		Transaction: nil, // TODO
	}

	return out, nil
}

func TransformLoadedAddresses(in loadedAddresses) (out rpc.LoadedAddresses) {
	if out.ReadOnly == nil {
		out.ReadOnly = []solana.PublicKey{}
	}
	if out.Writable == nil {
		out.Writable = []solana.PublicKey{}
	}

	for _, addr := range in.Readonly {
		out.ReadOnly = append(out.ReadOnly, solana.MustPublicKeyFromBase58(addr))
	}
	for _, addr := range in.Writable {
		out.Writable = append(out.Writable, solana.MustPublicKeyFromBase58(addr))
	}

	return out
}

func TransformReward(in reward) (*rpc.BlockReward, error) {
	lamports, err := strconv.ParseInt(in.Lamports, 10, 64)
	if err != nil {
		return nil, err
	}
	postBalance, err := strconv.ParseUint(in.PostBalance, 10, 64)
	if err != nil {
		return nil, err
	}

	out := &rpc.BlockReward{
		Pubkey:      solana.MustPublicKeyFromBase58(in.Pubkey),
		Lamports:    lamports,
		PostBalance: postBalance,
		RewardType:  rpc.RewardType(*in.RewardType),
		Commission:  in.Commission,
	}

	return out, nil
}

func TransformTokenBalance(in tokenBalance, pre bool) rpc.TokenBalance {
	parse := func(owner, programId, mint, amount string, decimals uint8) rpc.TokenBalance {
		_owner := solana.MustPublicKeyFromBase58(owner)
		_programId := solana.MustPublicKeyFromBase58(programId)
		_mint := solana.MustPublicKeyFromBase58(mint)

		return rpc.TokenBalance{
			AccountIndex: 0, // TODO
			Owner:        &_owner,
			ProgramId:    &_programId,
			Mint:         _mint,
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:   amount,
				Decimals: decimals,
				// UiAmount: , // TODO float value
				// UiAmountString: , // TODO amount accounting for decimals
			},
		}
	}

	if pre {
		return parse(
			*in.PreOwner,
			*in.PreProgramId,
			in.PreMint,
			strconv.FormatUint(uint64(in.PreAmount), 10),
			in.PreDecimals,
		)
	} else {
		return parse(
			*in.PostOwner,
			*in.PostProgramId,
			in.PostMint,
			strconv.FormatUint(uint64(in.PostAmount), 10),
			in.PostDecimals,
		)
	}
}
