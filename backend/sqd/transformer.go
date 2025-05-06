package sqd

import (
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"

	"github.com/subquery/solana-takoyaki/solana"
)

func TransformBlock(sqdBlock SolanaBlockResponse) (out *solana.Block, err error) {

	out = &solana.Block{
		BlockHeight:       sqdBlock.Header.Height,
		Blockhash:         sqdBlock.Header.Hash,
		PreviousBlockhash: sqdBlock.Header.ParentHash,
		ParentSlot:        sqdBlock.Header.ParentSlot,
		BlockTime:         sqdBlock.Header.Timestamp,
	}

	// Transform Token Balances
	preTokenBalances, postTokenBalances, err := groupTokenBalances(sqdBlock.TokenBalances, sqdBlock.Transactions)
	if err != nil {
		return nil, err
	}

	// Transform Balances
	preBalances, postBalances, err := groupBalances(sqdBlock.Balances, sqdBlock.Transactions)
	if err != nil {
		return nil, err
	}

	// Transform instructions
	instructions, innerInstructions, err := groupInstructions(sqdBlock.Instructions, sqdBlock.Transactions)
	if err != nil {
		return nil, err
	}

	logs, err := groupLogs(sqdBlock.Logs)
	if err != nil {
		return nil, err
	}

	// Transform Transactions
	if out.Transactions == nil {
		out.Transactions = []solana.Transaction{}
	}
	for _, tx := range sqdBlock.Transactions {
		inner := innerInstructions[tx.TransactionIndex]
		if inner == nil {
			inner = []solana.InnerInstruction{}
		}
		txLogs := logs[tx.TransactionIndex]
		if txLogs == nil {
			txLogs = []solana.Log{}
		}

		solanaTx, err := TransformTransaction(
			tx,
			sqdBlock.Header,
			preBalances[tx.TransactionIndex],
			postBalances[tx.TransactionIndex],
			preTokenBalances[tx.TransactionIndex],
			postTokenBalances[tx.TransactionIndex],
			instructions[tx.TransactionIndex],
			inner,
			txLogs,
		)
		if err != nil {
			return nil, err
		}
		out.Transactions = append(out.Transactions, *solanaTx)
	}

	// Transform Rewards
	if out.Rewards == nil {
		out.Rewards = []solana.BlockReward{}
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
		out.Signatures = []string{}
	}
	// TODO fill signatures

	return out, nil
}

func TransformTransaction(
	in transaction,
	header blockHeader,
	preBalances []uint64,
	postBalances []uint64,
	preTokenBalance []solana.TokenBalance,
	postTokenBalance []solana.TokenBalance,
	instructions []solana.CompiledInstruction,
	innerInstructions []solana.InnerInstruction,
	logs []solana.Log,
) (out *solana.Transaction, err error) {
	fee, err := strconv.ParseUint(in.Fee, 10, 64)
	if err != nil {
		return nil, err
	}

	computeUnitsConsumed, err := strconv.ParseUint(in.ComputeUnitsConsumed, 10, 64)
	if err != nil {
		return nil, err
	}

	sort.Slice(preTokenBalance, func(i, j int) bool {
		return preTokenBalance[i].AccountIndex < preTokenBalance[j].AccountIndex
	})

	sort.Slice(postTokenBalance, func(i, j int) bool {
		return postTokenBalance[i].AccountIndex < postTokenBalance[j].AccountIndex
	})

	out = &solana.Transaction{
		Slot:      header.Slot,
		BlockTime: header.Timestamp,
		Meta: &solana.TransactionMeta{
			Err:                  in.Err,
			Fee:                  fee,
			PreBalances:          preBalances,  // Incomplete data, only included balances that change
			PostBalances:         postBalances, // Incomplete data, only included balances that change
			InnerInstructions:    innerInstructions,
			PreTokenBalances:     preTokenBalance,
			PostTokenBalances:    postTokenBalance,
			Logs:                 logs, // Missing program invoke, success and consumed compute units logs
			ComputeUnitsConsumed: &computeUnitsConsumed,
			LoadedAddresses: solana.LoadedAddresses{
				Readonly: in.LoadedAddresses.Readonly,
				Writable: in.LoadedAddresses.Writable,
			},
			Rewards: []solana.BlockReward{}, // TODO can we get this from the solana request?
		},
		Transaction: &solana.JSONTransaction{
			Signatures: in.Signatures,
			Message: solana.Message{
				AccountKeys:  in.AccountKeys,
				Instructions: instructions,
			},
		},
	}

	return out, nil
}

func TransformInstruction(in instruction, tx transaction) (out *solana.CompiledInstruction, err error) {
	accounts := []uint16{}
	for _, account := range in.Accounts {
		idx, err := findAddressIndex(account, tx)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, uint16(idx))
	}

	programIdIndex, err := findAddressIndex(in.ProgramId, tx)
	if err != nil {
		return nil, fmt.Errorf("Unable to find programIdIndex: %v", tx)
	}

	out = &solana.CompiledInstruction{
		ProgramIDIndex: uint16(programIdIndex),
		Data:           in.Data,
		Accounts:       accounts,
	}
	return out, nil
}

func findAddressIndex(account string, tx transaction) (int, error) {
	idx := slices.Index(tx.AccountKeys, account)
	if idx >= 0 {
		return idx, nil
	}

	// Try the Loaded Addresses Writable with an offset of AccountKeys
	wLookupIdx := slices.Index(tx.LoadedAddresses.Writable, account)
	if wLookupIdx >= 0 {
		accountIdx := len(tx.AccountKeys) + wLookupIdx
		return accountIdx, nil
	}

	// Try the Loaded Addresses Readable with an offset of AccountKeys + Writable
	rLookupIdx := slices.Index(tx.LoadedAddresses.Readonly, account)
	if rLookupIdx >= 0 {
		accountIdx := len(tx.AccountKeys) + len(tx.LoadedAddresses.Writable) + rLookupIdx
		return accountIdx, nil
	}

	return -1, fmt.Errorf("Unable to find account key: %v", account)
}

func TransformReward(in reward) (*solana.BlockReward, error) {
	lamports, err := strconv.ParseInt(in.Lamports, 10, 64)
	if err != nil {
		return nil, err
	}
	postBalance, err := strconv.ParseUint(in.PostBalance, 10, 64)
	if err != nil {
		return nil, err
	}

	out := &solana.BlockReward{
		Pubkey:      in.Pubkey,
		Lamports:    lamports,
		PostBalance: postBalance,
		RewardType:  solana.RewardType(*in.RewardType),
		Commission:  in.Commission,
	}

	return out, nil
}

func TransformTokenBalance(in tokenBalance, tx transaction) (pre *solana.TokenBalance, post *solana.TokenBalance, err error) {
	parse := func(owner, programId *string, mint, amount string, decimals uint8) (*solana.TokenBalance, error) {

		amountBF, success := new(big.Float).SetString(amount)
		if !success {
			return nil, fmt.Errorf("Unable to parse amount %v", amount)
		}

		uiTokenAmount := &solana.UiTokenAmount{
			Amount:         amount,
			Decimals:       decimals,
			UiAmountString: "0",
		}

		// Values are only set for non-zero amounts
		if amount != "0" {
			uiAmount := shiftDecimalPlacesLeft(*amountBF, int64(decimals))
			floatAmount, _ := uiAmount.Float64()
			uiTokenAmount.UiAmount = &floatAmount
			uiTokenAmount.UiAmountString = uiAmount.Text(byte('f'), -1)
		}

		idx, err := findAddressIndex(in.Account, tx)
		if err != nil {
			return nil, err
		}

		return &solana.TokenBalance{
			AccountIndex:  uint16(idx),
			Owner:         owner,
			ProgramId:     programId,
			Mint:          mint,
			UiTokenAmount: uiTokenAmount,
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

func TransformLog(log logMessage) (out solana.Log) {
	return solana.Log{
		Message:   log.Message,
		ProgramId: log.ProgramId,
		LogIndex:  uint64(log.LogIndex),
		Kind:      log.Kind,
	}
}

func shiftDecimalPlacesLeft(input big.Float, places int64) *big.Float {
	// Compute 10^places as a *big.Float
	exp := new(big.Float).SetFloat64(1)
	ten := big.NewFloat(10)

	for i := int64(0); i < places; i++ {
		exp.Mul(exp, ten)
	}
	return input.Quo(&input, exp)
}

func groupInstructions(instructions []instruction, txs []transaction) (out map[uint][]solana.CompiledInstruction, inner map[uint][]solana.InnerInstruction, err error) {
	out = map[uint][]solana.CompiledInstruction{}
	innerInternal := map[uint]map[uint64]*solana.InnerInstruction{}
	for _, instruction := range instructions {
		tx, err := getTransactionByIndex(txs, instruction.TransactionIndex)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to find transaction for instruction: %v", err)
		}
		inst, err := TransformInstruction(instruction, *tx)
		if err != nil {
			return nil, nil, err
		}

		// Inner instructions have an array len > 1. See instruction struct definition for more info
		if len(instruction.InstructionAddress) == 1 {
			if out[instruction.TransactionIndex] == nil {
				out[instruction.TransactionIndex] = []solana.CompiledInstruction{}
			}
			out[instruction.TransactionIndex] = append(out[instruction.TransactionIndex], *inst)
		} else {
			if innerInternal[instruction.TransactionIndex] == nil {
				innerInternal[instruction.TransactionIndex] = map[uint64]*solana.InnerInstruction{}
			}
			innerIdx := instruction.InstructionAddress[0]
			if innerInternal[instruction.TransactionIndex][innerIdx] == nil {
				innerInternal[instruction.TransactionIndex][innerIdx] = &solana.InnerInstruction{
					Index:        innerIdx,
					Instructions: []solana.CompiledInstruction{},
				}
			}

			innerInternal[instruction.TransactionIndex][innerIdx].Instructions = append(innerInternal[instruction.TransactionIndex][innerIdx].Instructions, *inst)
		}
	}

	// Flatten inner instructions
	inner = map[uint][]solana.InnerInstruction{}
	for txIdx, innerInst := range innerInternal {
		for _, inst := range innerInst {
			inner[txIdx] = append(inner[txIdx], *inst)
		}
	}

	return out, inner, nil
}

func groupBalances(in []balance, txs []transaction) (preBalances, postBalances map[uint][]uint64, err error) {
	preBalances = map[uint][]uint64{}
	postBalances = map[uint][]uint64{}

	// Group balances by the tx so we can sort them
	bxs := map[uint][]balance{}
	for _, balance := range in {
		bxs[balance.TransactionIndex] = append(bxs[balance.TransactionIndex], balance)
	}

	for txIdx, txBals := range bxs {
		tx, err := getTransactionByIndex(txs, txIdx)
		if err != nil {
			return nil, nil, err
		}

		if tx == nil {
			return nil, nil, fmt.Errorf("Unable to find transaction for index %v", txIdx)
		}

		// Sort the balances by the account index in the transaction
		slices.SortFunc(txBals, func(i, j balance) int {
			accountIdxI, _ := findAddressIndex(i.Account, *tx)
			accountIdxJ, _ := findAddressIndex(j.Account, *tx)

			return int(accountIdxI) - int(accountIdxJ)
		})

		// Split the balances into pre and post balances
		for _, bal := range txBals {
			if preBalances[txIdx] == nil {
				preBalances[txIdx] = []uint64{}
			}
			if postBalances[txIdx] == nil {
				postBalances[txIdx] = []uint64{}
			}

			preBal, err := strconv.ParseUint(bal.Pre, 10, 64)
			if err != nil {
				fmt.Println("Transaction", txIdx, tx.Signatures)
				return nil, nil, fmt.Errorf("Unable to parse pre balance. Err=%v. Value=%v", err, bal)
			}
			postBal, err := strconv.ParseUint(bal.Post, 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("Unable to parse post balance: %v", err)
			}

			preBalances[txIdx] = append(preBalances[txIdx], preBal)
			postBalances[txIdx] = append(postBalances[txIdx], postBal)
		}
	}

	return preBalances, postBalances, nil
}

func groupTokenBalances(in []tokenBalance, txs []transaction) (preTokenBalances, postTokenBalances map[uint][]solana.TokenBalance, err error) {
	preTokenBalances = map[uint][]solana.TokenBalance{}
	postTokenBalances = map[uint][]solana.TokenBalance{}
	for _, balance := range in {
		if preTokenBalances[balance.TransactionIndex] == nil {
			preTokenBalances[balance.TransactionIndex] = []solana.TokenBalance{}
		}
		if postTokenBalances[balance.TransactionIndex] == nil {
			postTokenBalances[balance.TransactionIndex] = []solana.TokenBalance{}
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

func groupLogs(in []logMessage) (out map[uint][]solana.Log, err error) {
	out = map[uint][]solana.Log{}
	for _, log := range in {
		if out[log.TransactionIndex] == nil {
			out[log.TransactionIndex] = []solana.Log{}
		}
		out[log.TransactionIndex] = append(out[log.TransactionIndex], TransformLog(log))
	}

	return out, nil
}

func getTransactionByIndex(txs []transaction, idx uint) (*transaction, error) {
	for _, tx := range txs {
		if tx.TransactionIndex == idx {
			return &tx, nil
		}
	}

	return nil, fmt.Errorf("Unable to find transaction with index: %v", idx)
}
