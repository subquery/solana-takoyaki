package sqd

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mr-tron/base58"
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

	// Transform Token Balances
	preTokenBalances, postTokenBalances, err := groupTokenBalances(sqdBlock.TokenBalances, sqdBlock.Transactions)
	if err != nil {
		return nil, err
	}

	// Transform Balances
	preBalances, postBalances, err := groupBalances(sqdBlock.Balances)
	if err != nil {
		return nil, err
	}

	// Transform instructions
	instructions, innerInstructions, err := groupInstructions(sqdBlock.Instructions)
	if err != nil {
		return nil, err
	}

	// Transform Transactions
	if out.Transactions == nil {
		out.Transactions = []rpc.ParsedTransactionWithMeta{}
	}
	for _, tx := range sqdBlock.Transactions {
		solanaTx, err := TransformTransaction(
			tx,
			sqdBlock.Header,
			preBalances[tx.TransactionIndex],
			postBalances[tx.TransactionIndex],
			preTokenBalances[tx.TransactionIndex],
			postTokenBalances[tx.TransactionIndex],
			instructions[tx.TransactionIndex],
			innerInstructions[tx.TransactionIndex],
		)
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

func TransformTransaction(
	in transaction,
	header blockHeader,
	preBalances []uint64,
	postBalances []uint64,
	preTokenBalance []rpc.TokenBalance,
	postTokenBalance []rpc.TokenBalance,
	instructions []*rpc.ParsedInstruction,
	innerInstructions []rpc.ParsedInnerInstruction,
) (out *rpc.ParsedTransactionWithMeta, err error) {
	timestamp := solana.UnixTimeSeconds(header.Timestamp)
	fee, err := strconv.ParseUint(in.Fee, 10, 64)
	if err != nil {
		return nil, err
	}

	sigs := []solana.Signature{}
	for _, sig := range in.Signatures {
		sigs = append(sigs, solana.MustSignatureFromBase58(sig))
	}

	sort.Slice(preTokenBalance, func(i, j int) bool {
		return preTokenBalance[i].AccountIndex < preTokenBalance[j].AccountIndex
	})

	sort.Slice(postTokenBalance, func(i, j int) bool {
		return postTokenBalance[i].AccountIndex < postTokenBalance[j].AccountIndex
	})

	out = &rpc.ParsedTransactionWithMeta{
		Slot:      header.Slot,
		BlockTime: &timestamp,
		Meta: &rpc.ParsedTransactionMeta{
			Err:               in.Err,
			Fee:               fee,
			PreBalances:       preBalances,  // Incomplete data
			PostBalances:      postBalances, // Incomplete data
			InnerInstructions: innerInstructions,
			PreTokenBalances:  preTokenBalance,
			PostTokenBalances: postTokenBalance,
			LogMessages:       []string{}, // TODO
		},
		Transaction: &rpc.ParsedTransaction{
			Signatures: sigs,
			Message: rpc.ParsedMessage{
				AccountKeys:     TransformParsedMessageAccount(in), // Missing data
				Instructions:    instructions,
				RecentBlockHash: "", // TODO
			},
		},
	}

	return out, nil
}

func TransformParsedMessageAccount(in transaction) (out []rpc.ParsedMessageAccount) {
	out = []rpc.ParsedMessageAccount{}

	if in.TransactionIndex == 16 {
		fmt.Println("ACCT KEYS", in.AccountKeys)
		fmt.Println("TALBE LOOKUPS", in.AddressTableLookups)
		fmt.Println("LOADED ADDRS", in.LoadedAddresses)
	}

	for _, accountKey := range in.AccountKeys {
		pubkey := solana.MustPublicKeyFromBase58(accountKey)
		out = append(out, rpc.ParsedMessageAccount{
			PublicKey: pubkey,
			// SQD assume the first account key is the fee payer. https://github.com/subsquid/archive.py/blob/a585e7121b32fe8dda7e723a7e7b626ecef851a3/sqa/solana/writer/parquet.py#L116
			Signer:   accountKey == in.FeePayer,
			Writable: slices.Contains(in.LoadedAddresses.Writable, accountKey),
		})
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

func TransformTokenBalance(in tokenBalance, tx transaction) (pre *rpc.TokenBalance, post *rpc.TokenBalance, err error) {
	parse := func(owner, programId *string, mint, amount string, decimals uint8) (*rpc.TokenBalance, error) {
		var _owner, _programId solana.PublicKey
		if owner != nil {
			_owner = solana.MustPublicKeyFromBase58(*owner)
		}
		if programId != nil {
			_programId = solana.MustPublicKeyFromBase58(*programId)
		}

		_mint := solana.MustPublicKeyFromBase58(mint)

		amountInt, err := strconv.ParseInt(amount, 10, 64)
		if err != nil {
			return nil, err
		}
		uiAmount := shiftDecimalPlaces(amountInt, int(decimals))

		idx := slices.Index(tx.AccountKeys, in.Account)
		// TODO restore this
		// if idx < 0 {
		// 	return nil, fmt.Errorf("Unable to find account key: %v", in.Account)
		// }

		return &rpc.TokenBalance{
			AccountIndex: uint16(idx),
			Owner:        &_owner,
			ProgramId:    &_programId,
			Mint:         _mint,
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:         amount,
				Decimals:       decimals,
				UiAmount:       &uiAmount,
				UiAmountString: strconv.FormatFloat(uiAmount, byte('f'), int(decimals), 64),
			},
		}, nil
	}

	// tokenBalance is generally one or the other
	if in.PreOwner != nil {
		pre, err = parse(
			in.PreOwner,
			in.PreProgramId,
			in.PreMint,
			in.PreAmount,
			in.PreDecimals,
		)
	}
	if in.PostOwner != nil {
		post, err = parse(
			in.PostOwner,
			in.PostProgramId,
			in.PostMint,
			in.PostAmount,
			in.PostDecimals,
		)
	}

	return pre, post, err
}

func TransformInstruction(in instruction) (out *rpc.ParsedInstruction, err error) {
	programId := solana.MustPublicKeyFromBase58(in.ProgramId)

	accounts := []solana.PublicKey{}
	for _, account := range in.Accounts {
		accounts = append(accounts, solana.MustPublicKeyFromBase58(account))
	}

	data := solana.Base58{}
	if len(in.Data) > 0 {
		dataB, err := base58.Decode(in.Data)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse instruction data: %v", err)
		}
		data = solana.Base58(dataB)
	}

	out = &rpc.ParsedInstruction{
		Program:     "", // TODO
		ProgramId:   programId,
		Parsed:      nil, // TODO
		Data:        data,
		Accounts:    accounts,
		StackHeight: 0, // TODO
	}
	return out, nil
}

func shiftDecimalPlaces(input int64, places int) float64 {
	return float64(input) / math.Pow10(places)
}

func groupInstructions(instructions []instruction) (out map[uint][]*rpc.ParsedInstruction, inner map[uint][]rpc.ParsedInnerInstruction, err error) {
	out = map[uint][]*rpc.ParsedInstruction{}
	innerInternal := map[uint]map[uint64]*rpc.ParsedInnerInstruction{}
	for _, instruction := range instructions {
		inst, err := TransformInstruction(instruction)
		if err != nil {
			return nil, nil, err
		}

		// Inner instructions have an array len > 1. See instruction struct definition for more info
		if len(instruction.InstructionAddress) == 1 {
			if out[instruction.TransactionIndex] == nil {
				out[instruction.TransactionIndex] = []*rpc.ParsedInstruction{}
			}
			out[instruction.TransactionIndex] = append(out[instruction.TransactionIndex], inst)
		} else {
			if innerInternal[instruction.TransactionIndex] == nil {
				innerInternal[instruction.TransactionIndex] = map[uint64]*rpc.ParsedInnerInstruction{}
			}
			innerIdx := instruction.InstructionAddress[0]
			if innerInternal[instruction.TransactionIndex][innerIdx] == nil {
				innerInternal[instruction.TransactionIndex][innerIdx] = &rpc.ParsedInnerInstruction{
					Index:        innerIdx,
					Instructions: []*rpc.ParsedInstruction{},
				}
			}

			innerInternal[instruction.TransactionIndex][innerIdx].Instructions = append(innerInternal[instruction.TransactionIndex][innerIdx].Instructions, inst)
		}
	}

	// Flatten inner instructions
	inner = map[uint][]rpc.ParsedInnerInstruction{}
	for txIdx, innerInst := range innerInternal {
		for _, inst := range innerInst {
			inner[txIdx] = append(inner[txIdx], *inst)
		}
	}

	return out, inner, nil
}

func groupBalances(in []balance) (preBalances, postBalances map[uint][]uint64, err error) {
	preBalances = map[uint][]uint64{}
	postBalances = map[uint][]uint64{}
	for _, balance := range in {
		if preBalances[balance.TransactionIndex] == nil {
			preBalances[balance.TransactionIndex] = []uint64{}
		}
		if postBalances[balance.TransactionIndex] == nil {
			postBalances[balance.TransactionIndex] = []uint64{}
		}

		preBal, err := strconv.ParseUint(balance.Pre, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to parse pre balance: %v", err)
		}
		postBal, err := strconv.ParseUint(balance.Post, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to parse post balance: %v", err)
		}

		preBalances[balance.TransactionIndex] = append(preBalances[balance.TransactionIndex], preBal)
		postBalances[balance.TransactionIndex] = append(postBalances[balance.TransactionIndex], postBal)
	}

	return preBalances, postBalances, nil
}

func groupTokenBalances(in []tokenBalance, txs []transaction) (preTokenBalances, postTokenBalances map[uint][]rpc.TokenBalance, err error) {
	preTokenBalances = map[uint][]rpc.TokenBalance{}
	postTokenBalances = map[uint][]rpc.TokenBalance{}
	for _, balance := range in {
		if preTokenBalances[balance.TransactionIndex] == nil {
			preTokenBalances[balance.TransactionIndex] = []rpc.TokenBalance{}
		}
		if postTokenBalances[balance.TransactionIndex] == nil {
			postTokenBalances[balance.TransactionIndex] = []rpc.TokenBalance{}
		}

		tx, err := getTransactionByIndex(txs, balance.TransactionIndex)
		if err != nil {
			return nil, nil, err
		}

		pre, post, err := TransformTokenBalance(balance, *tx)
		if err != nil {
			return nil, nil, fmt.Errorf("Error parsing token balances. Tx index: %v. Err: %v", balance.TransactionIndex, err)
		}
		if pre != nil {
			preTokenBalances[balance.TransactionIndex] = append(preTokenBalances[balance.TransactionIndex], *pre)
		}
		if post != nil {
			postTokenBalances[balance.TransactionIndex] = append(postTokenBalances[balance.TransactionIndex], *post)
		}
	}

	return preTokenBalances, postTokenBalances, nil
}

func getTransactionByIndex(txs []transaction, idx uint) (*transaction, error) {
	for _, tx := range txs {
		if tx.TransactionIndex == idx {
			return &tx, nil
		}
	}

	return nil, fmt.Errorf("Unable to find transaction with index: %v", idx)
}
