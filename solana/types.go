package solana

/**
 * These types are mostly sourced from github.com/gagliardetto/solana-go
 * But with less parsing and minor variations to align directly with RPC types
 * */

type Block struct {
	// The blockhash of this block.
	Blockhash string `json:"blockhash"`

	// The blockhash of this block's parent;
	// if the parent block is not available due to ledger cleanup,
	// this field will return "11111111111111111111111111111111".
	PreviousBlockhash string `json:"previousBlockhash"`

	// The slot index of this block's parent.
	ParentSlot uint64 `json:"parentSlot"`

	// Present if "full" transaction details are requested.
	Transactions []Transaction `json:"transactions"`

	// Present if "signatures" are requested for transaction details;
	// an array of signatures, corresponding to the transaction order in the block.
	Signatures []string `json:"signatures"`

	// Present if rewards are requested.
	Rewards []BlockReward `json:"rewards"`

	// Estimated production time, as Unix timestamp (seconds since the Unix epoch).
	// Nil if not available.
	BlockTime int64 `json:"blockTime"`

	// The number of blocks beneath this block.
	BlockHeight uint64 `json:"blockHeight"`
}

type Transaction struct {

	// The slot this transaction was processed in.
	Slot uint64 `json:"slot"`

	// Estimated production time, as Unix timestamp (seconds since the Unix epoch)
	// of when the transaction was processed.
	// Nil if not available.
	BlockTime int64 `json:"blockTime"`

	// The transaction data when the encoding is `json`
	Transaction *JSONTransaction `json:"transaction"`

	// Transaction status metadata object
	Meta *TransactionMeta `json:"meta,omitempty"`
	// Version TransactionVersion `json:"version"`
}

type JSONTransaction struct {
	Message Message `json:"message"`

	Signatures []string `json:"signatures"`
}

type Message struct {
	// List of base-58 encoded public keys used by the transaction,
	// including by the instructions and for signatures.
	// The first `message.header.numRequiredSignatures` public keys must sign the transaction.
	AccountKeys []string `json:"accountKeys"` // static keys; static keys + dynamic keys if after resolution (i.e. call to `ResolveLookups()`)

	// Details the account types and signatures required by the transaction.
	Header MessageHeader `json:"header"`

	// A base-58 encoded hash of a recent block in the ledger used to
	// prevent transaction duplication and to give transactions lifetimes.
	RecentBlockhash string `json:"recentBlockhash"`

	// List of program instructions that will be executed in sequence
	// and committed in one atomic transaction if all succeed.
	Instructions []CompiledInstruction `json:"instructions"`

	// List of address table lookups used to load additional accounts for this transaction.
	AddressTableLookups []MessageAddressTableLookup `json:"addressTableLookups"`

	// // The actual tables that contain the list of account pubkeys.
	// // NOTE: you need to fetch these from the chain, and then call `SetAddressTables`
	// // before you use this transaction -- otherwise, you will get a panic.
	// addressTables map[string]PublicKeySlice

	// resolved bool // if true, the lookups have been resolved, and the `AccountKeys` slice contains all the accounts (static + dynamic).
}

type MessageHeader struct {
	// The total number of signatures required to make the transaction valid.
	// The signatures must match the first `numRequiredSignatures` of `message.account_keys`.
	NumRequiredSignatures uint8 `json:"numRequiredSignatures"`

	// The last numReadonlySignedAccounts of the signed keys are read-only accounts.
	// Programs may process multiple transactions that load read-only accounts within
	// a single PoH entry, but are not permitted to credit or debit lamports or modify
	// account data.
	// Transactions targeting the same read-write account are evaluated sequentially.
	NumReadonlySignedAccounts uint8 `json:"numReadonlySignedAccounts"`

	// The last `numReadonlyUnsignedAccounts` of the unsigned keys are read-only accounts.
	NumReadonlyUnsignedAccounts uint8 `json:"numReadonlyUnsignedAccounts"`
}

type MessageAddressTableLookup struct {
	AccountKey      string  `json:"accountKey"` // The account key of the address table.
	WritableIndexes []uint8 `json:"writableIndexes"`
	ReadonlyIndexes []uint8 `json:"readonlyIndexes"`
}

type TransactionMeta struct {
	// Error if transaction failed, null if transaction succeeded.
	// https://github.com/solana-labs/solana/blob/master/sdk/src/transaction.rs#L24
	Err interface{} `json:"err"`

	// Fee this transaction was charged
	Fee uint64 `json:"fee"`

	// Array of u64 account balances from before the transaction was processed
	PreBalances []uint64 `json:"preBalances"`

	// Array of u64 account balances after the transaction was processed
	PostBalances []uint64 `json:"postBalances"`

	// List of inner instructions or omitted if inner instruction recording
	// was not yet enabled during this transaction
	InnerInstructions []InnerInstruction `json:"innerInstructions"`

	// List of token balances from before the transaction was processed
	// or omitted if token balance recording was not yet enabled during this transaction
	PreTokenBalances []TokenBalance `json:"preTokenBalances"`

	// List of token balances from after the transaction was processed
	// or omitted if token balance recording was not yet enabled during this transaction
	PostTokenBalances []TokenBalance `json:"postTokenBalances"`

	// Array of string log messages or omitted if log message
	// recording was not yet enabled during this transaction
	// LogMessages []string `json:"logMessages"`
	Logs []Log `json:"logs"`

	// DEPRECATED: Transaction status.
	// Status DeprecatedTransactionMetaStatus `json:"status"`

	Rewards []BlockReward `json:"rewards"`

	LoadedAddresses LoadedAddresses `json:"loadedAddresses"`

	ReturnData *ReturnData `json:"returnData,omitempty"`

	ComputeUnitsConsumed *uint64 `json:"computeUnitsConsumed"`
}

type Log struct {
	Message   string `json:"message"`
	ProgramId string `json:"programId"`
	LogIndex  uint64 `json:"logIndex"`
	Kind      string `json:"kind"` // 'log' | 'data' | 'other'
}

type BlockReward struct {
	// The public key of the account that received the reward.
	Pubkey string `json:"pubkey"`

	// Number of reward lamports credited or debited by the account, as a i64.
	Lamports int64 `json:"lamports"`

	// Account balance in lamports after the reward was applied.
	PostBalance uint64 `json:"postBalance"`

	// Type of reward: "Fee", "Rent", "Voting", "Staking".
	RewardType RewardType `json:"rewardType"`

	// Vote account commission when the reward was credited,
	// only present for voting and staking rewards.
	Commission *uint8 `json:"commission,omitempty"`
}

type RewardType string

const (
	RewardTypeFee     RewardType = "Fee"
	RewardTypeRent    RewardType = "Rent"
	RewardTypeVoting  RewardType = "Voting"
	RewardTypeStaking RewardType = "Staking"
)

type InnerInstruction struct {
	// Index of the transaction instruction from which the inner instruction(s) originated
	Index uint64 `json:"index"`

	// Ordered list of inner program instructions that were invoked during a single transaction instruction.
	Instructions []CompiledInstruction `json:"instructions"`
}

type CompiledInstruction struct {
	// Index into the message.accountKeys array indicating the program account that executes this instruction.
	// NOTE: it is actually a uint8, but using a uint16 because uint8 is treated as a byte everywhere,
	// and that can be an issue.
	ProgramIDIndex uint16 `json:"programIdIndex"`

	// List of ordered indices into the message.accountKeys array indicating which accounts to pass to the program.
	// NOTE: it is actually a []uint8, but using a uint16 because []uint8 is treated as a []byte everywhere,
	// and that can be an issue.
	Accounts []uint16 `json:"accounts"`

	// The program input data encoded in a base-58 string.
	Data string `json:"data"`
}

type TokenBalance struct {
	// Index of the account in which the token balance is provided for.
	AccountIndex uint16 `json:"accountIndex"`

	// Pubkey of token balance's owner.
	Owner *string `json:"owner,omitempty"`
	// Pubkey of token program.
	ProgramId *string `json:"programId,omitempty"`

	// Pubkey of the token's mint.
	Mint          string         `json:"mint"`
	UiTokenAmount *UiTokenAmount `json:"uiTokenAmount"`
}

type UiTokenAmount struct {
	// Raw amount of tokens as a string, ignoring decimals.
	Amount string `json:"amount"`

	// TODO: <number> == int64 ???
	// Number of decimals configured for token's mint.
	Decimals uint8 `json:"decimals"`

	// DEPRECATED: Token amount as a float, accounting for decimals.
	UiAmount *float64 `json:"uiAmount"`

	// Token amount as a string, accounting for decimals.
	UiAmountString string `json:"uiAmountString"`
}

type LoadedAddresses struct {
	Readonly []string `json:"readonly"`
	Writable []string `json:"writable"`
}

type ReturnData struct {
	ProgramId string `json:"programId"`
	Data      string `json:"data"`
}
