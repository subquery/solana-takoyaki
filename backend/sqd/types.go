package sqd

import (
	"encoding/json"
	"fmt"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/subquery/solana-takoyaki/utils"
)

type NetworkMeta struct {
	GenesisHash string
	ChainId     string
	StartBlock  uint
}

/* Spec can be found here https://docs.sqd.ai/solana-indexing/network-api/solana-api/*/

type SolanaRequest struct {
	Type string `json:"type"` // Always "solana"

	/* FromBlock and ToBlock are the block numbers not the slot numbers*/
	FromBlock uint `json:"fromBlock"`
	ToBlock   uint `json:"toBlock"`

	IncludeAllBlocks *bool `json:"includeAllBlocks,omitempty"` // default: false
	//
	// Fields to.select, see the specific types for provided details
	Fields Fields `json:"fields,omitempty"`

	Transactions  []TransactionRequest  `json:"transactions,omitempty"`
	Instructions  []InstructionRequest  `json:"instructions,omitempty"`
	Logs          []LogRequest          `json:"logs,omitempty"`
	Rewards       []RewardRequest       `json:"rewards,omitempty"`
	TokenBalances []TokenBalanceRequest `json:"tokenBalances,omitempty"`
	Balances      []BalancesRequest     `json:"balances,omitempty"`
}

// Need custom serialization for sqd interface that differs from default json.Marshal
func (s SolanaRequest) MarshalJSON() ([]byte, error) {
	return utils.MarshalWithEmptySlices(s)
}

type SolanaBlockResponse struct {
	Header        blockHeader    `json:"header"`
	Transactions  []transaction  `json:"transactions"` // Excludes all Voting Program transactions
	Instructions  []instruction  `json:"instructions"`
	Logs          []logMessage   `json:"logs"`
	Balances      []balance      `json:"balances"` // Only seems to contain the balances of accounts where balances changed
	TokenBalances []tokenBalance `json:"tokenBalances"`
	Rewards       []reward       `json:"rewards"`
}

type Fields struct {
	Instruction  map[string]bool `json:"instruction,omitempty"`
	Transaction  map[string]bool `json:"transaction,omitempty"`
	Log          map[string]bool `json:"log,omitempty"`
	Balance      map[string]bool `json:"balance,omitempty"`
	TokenBalance map[string]bool `json:"tokenBalance,omitempty"`
	Reward       map[string]bool `json:"reward,omitempty"`
	Block        map[string]bool `json:"block,omitempty"`
}

type TransactionRequest struct {
	/* Filters */
	FeePayer []string `json:"feePayer,omitempty"`

	/* Filed Selection */
	Instructions  bool `json:"instructions,omitempty"`
	Logs          bool `json:"logs,omitempty"`
	Balances      bool `json:"balances,omitempty"`
	TokenBalances bool `json:"tokenBalances,omitempty"`
}

type InstructionRequest struct {
	/* Filters */
	ProgramId   []string `json:"programId,omitempty"`
	D1          []string `json:"d1,omitempty"`
	D2          []string `json:"d2,omitempty"`
	D3          []string `json:"d3,omitempty"`
	D4          []string `json:"d4,omitempty"`
	D8          []string `json:"d8,omitempty"`
	A0          []string `json:"a0,omitempty"`
	A1          []string `json:"a1,omitempty"`
	A2          []string `json:"a2,omitempty"`
	A3          []string `json:"a3,omitempty"`
	A4          []string `json:"a4,omitempty"`
	A5          []string `json:"a5,omitempty"`
	A6          []string `json:"a6,omitempty"`
	A7          []string `json:"a7,omitempty"`
	A8          []string `json:"a8,omitempty"`
	A9          []string `json:"a9,omitempty"`
	IsCommitted bool     `json:"isCommitted,omitempty"`

	/* Field Selection */
	Transaction              bool `json:"transaction,omitempty"`
	TransactionBalances      bool `json:"transactionBalances,omitempty"`
	TransactionTokenBalances bool `json:"transactionTokenBalances,omitempty"`
	TransactionInstructions  bool `json:"transactionInstructions,omitempty"`
	InnerInstructions        bool `json:"innerInstructions,omitempty"`
	Logs                     bool `json:"logs,omitempty"`
}

func (ir *InstructionRequest) SetAccounts(idx int, accounts []string) error {

	accountFields := [][]string{
		ir.A0, ir.A1, ir.A2, ir.A3, ir.A4,
		ir.A5, ir.A6, ir.A7, ir.A8, ir.A9,
	}

	if idx < 0 {
		return fmt.Errorf("Account index must be >= 0")
	}

	if idx >= len(accountFields) {
		return fmt.Errorf("Account filter length is limited to %v", len(accountFields))
	}

	accountFields[idx] = accounts
	return nil
}

type LogRequest struct {
	/* Filters */
	ProgramId []string `json:"programId,omitempty"`
	Kind      []string `json:"kind,omitempty"` // 'log' | 'data' | 'other'

	/* Field Selection */
	Transaction bool `json:"transaction,omitempty"`
	Instruction bool `json:"instruction,omitempty"`
}

type RewardRequest struct {
	PubKey []string `json:"pubkey,omitempty"`
}

type BalancesRequest struct {
	Account []string `json:"account,omitempty"`

	Transaction             bool `json:"transaction,omitempty"`
	TransactionInstructions bool `json:"transactionInstructions,omitempty"`
}

type TokenBalanceRequest struct {
	Account       []string `json:"account,omitempty"`
	PreProgramId  []string `json:"preProgramId,omitempty"`
	PostProgramId []string `json:"postProgramId,omitempty"`
	PreMint       []string `json:"preMint,omitempty"`
	PostMint      []string `json:"postMint,omitempty"`
	PreOwner      []string `json:"preOwner,omitempty"`
	PostOwner     []string `json:"postOwner,omitempty"`

	Transaction             *bool `json:"transaction,omitempty"`
	TransactionInstructions *bool `json:"transactionInstructions,omitempty"`
}

type instruction struct {
	// independent of field selectors
	TransactionIndex uint `json:"transactionIndex"`
	// Used to identify inner instructions. https://docs.sqd.ai/solana-indexing/sdk/solana-batch/field-selection/#instruction
	InstructionAddress []uint64 `json:"instructionAddress"`

	// can be disabled with field selectors
	ProgramId   string   `json:"programId"`
	Accounts    []string `json:"accounts"`
	Data        string   `json:"data"`
	IsCommitted bool     `json:"isCommitted"`

	// can be enabled with field selectors
	ComputeUnitsConsumed  string  `json:"computeUnitsConsumed"`
	Error                 *string `json:"error"`
	HasDroppedLogMessages bool    `json:"hasDroppedLogMessages"`
}

type transaction struct {
	// independent of field selectors
	TransactionIndex uint `json:"transactionIndex"`

	// can be disabled with field selectors
	Signatures []string    `json:"signatures"`
	Err        interface{} `json:"err"` // null | object

	// can be requested with field selectors
	Version                     rpc.TransactionVersion `json:"version"` //'legacy' | number // TODO
	AccountKeys                 []string               `json:"accountKeys"`
	AddressTableLookups         []addressTableLookup   `json:"addressTableLookups"`
	NumReadonlySignedAccounts   uint                   `json:"numReadonlySignedAccounts"`
	NumReadonlyUnsignedAccounts uint                   `json:"numReadonlyUnsignedAccounts"`
	NumRequiredSignatures       uint                   `json:"numRequiredSignatures"`
	RecentBlockhash             string                 `json:"recentBlockhash"`
	ComputeUnitsConsumed        string                 `json:"computeUnitsConsumed"`
	Fee                         string                 `json:"fee"`
	FeePayer                    string                 `json:"feePayer"`        // Undocumented
	LoadedAddresses             loadedAddresses        `json:"loadedAddresses"` // request the whole struct with loadedAddresses: true
	HasDroppedLogMessages       bool                   `json:"hasDroppedLogMessages"`
}

type logMessage struct {
	// independent of field selectors
	TransactionIndex   uint   `json:"transactionIndex"`
	LogIndex           uint   `json:"logIndex"`
	InstructionAddress []uint `json:"instructionAddress"`

	// can be disabled with field selectors
	ProgramId string `json:"programId"`
	Kind      string `json:"kind"` //'log' | 'data' | 'other'
	Message   string `json:"message"`
}

func (l *logMessage) String() string {
	switch l.Kind {
	case "log":
		return fmt.Sprintf("Program log: %v", l.Message)
	case "data":
		return fmt.Sprintf("Program data: %v", l.Message) // Example https://solscan.io/tx/57moryaDygCpmdG5Bx9DTpwdrkz7QYVhD4Gqf8srN2PX4ZXp5SQDemMwRzgFj6dtbG7W5RUv4S2KadetiB4NJtqV `Program data: GmTE6l15n9+1K0GKz0oSSO8oves0qt09GsKz1QNA3hkOpcvC0rPMywt4KffaIJMAVQlyjQhUVOXGyn09Lxu29Ty1k5m72ijBAIkB9C0BAAAAAAAjpZzi/////6zzsgcAAAAAAAAAAAAAAAAA`
	case "other":
		return l.Message // Example https://solscan.io/tx/5skPYHKGg46Bv271wERLGxD9pPR3KnyovLqhRtyw6N3vUZybskWijJHFXWyaqHbjjrdtxirDGdQ71KfpCgRuegZ8 `Program return: JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4 RJG4pwkAAAA=`
	}
	return ""
}

type balance struct {
	// independent of field selectors
	TransactionIndex uint   `json:"transactionIndex"`
	Account          string `json:"account"`

	// can be disabled with field selectors
	Pre  string `json:"pre"`
	Post string `json:"post"`
}

type loadedAddresses struct {
	Readonly []string `json:"readonly"`
	Writable []string `json:"writable"`
}

// These values always seem to be null unable to determine types
type addressTableLookup struct {
	AccountKey      string  `json:"accountKey"`
	ReadonlyIndexes []uint8 `json:"readonlyIndexes"`
	WritableIndexes []uint8 `json:"writableIndexes"`
}

type reward struct {
	// independent of field selectors
	Pubkey string `json:"pubkey"`

	// can be disabled with field selectors
	Lamports   string  `json:"lamports"`
	RewardType *string `json:"rewardType"`

	// can be enabled by field selectors
	PostBalance string `json:"postBalance"`
	Commission  *uint8 `json:"commission"`
}

type blockHeader struct {
	// independent of field selectors
	Hash       string `json:"hash"`
	Height     uint64 `json:"number"`
	ParentHash string `json:"parentHash"`

	// can be disabled with field selectors
	Slot       uint64 `json:"slot"`
	ParentSlot uint64 `json:"parentSlot"`
	Timestamp  int64  `json:"timestamp"`
}

func (b *blockHeader) UnmarshalJSON(data []byte) error {
	type Alias blockHeader

	type rawHeader struct {
		Alias
		Slot         *uint64 `json:"slot"`
		ParentSlot   *uint64 `json:"parentSlot"`
		Height       *uint64 `json:"height"`
		Number       *uint64 `json:"number"`
		ParentNumber *uint64 `json:"parentNumber"`
	}

	var raw rawHeader
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	b.Hash = raw.Hash
	b.ParentHash = raw.ParentHash
	b.Timestamp = raw.Timestamp

	// Legacy archive uses slot, parentSlot while soldexer uses number and parent number
	// They also mix number and height
	if raw.Slot != nil || raw.ParentSlot != nil {
		b.Height = *raw.Number
		b.Slot = *raw.Slot
		if raw.ParentSlot != nil {
			b.ParentSlot = *raw.ParentSlot
		}
	} else {
		b.Height = *raw.Height
		b.Slot = *raw.Number
		if raw.ParentNumber != nil {
			b.ParentSlot = *raw.ParentNumber
		}
	}

	return nil
}

type tokenBalance struct {
	// independent of field selectors
	TransactionIndex uint   `json:"transactionIndex"`
	Account          string `json:"account"`

	// can be disabled with field selectors
	PreMint      string  `json:"preMint"`
	PreDecimals  uint8   `json:"preDecimals"`
	PreOwner     *string `json:"preOwner"`
	PreAmount    string  `json:"preAmount"`
	PostMint     string  `json:"postMint"`
	PostDecimals uint8   `json:"postDecimals"`
	PostOwner    *string `json:"postOwner"`
	PostAmount   string  `json:"postAmount"`

	// can be enabled by field selectors
	PostProgramId *string `json:"postProgramId"`
	PreProgramId  *string `json:"preProgramId"`
}
