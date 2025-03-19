package sqd

import (
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/subquery/solana-takoyaki/utils"
)

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

	Transactions []TransactionRequest `json:"transactions,omitempty"`
	Instructions []InstructionRequest `json:"instructions,omitempty"`
	Logs         []LogRequest         `json:"logs,omitempty"`
	Rewards      []RewardRequest      `json:"rewards,omitempty"`

	/*Not implemented*/
	// Balances
	// TokenBalances
}

// Need custom serialization for sqd interface that differs from default json.Marshal
func (s SolanaRequest) MarshalJSON() ([]byte, error) {
	return utils.MarshalWithEmptySlices(s)
}

type SolanaBlockResponse struct {
	Header        blockHeader           `json:"header"`
	Transactions  []TransactionResponse `json:"transactions"`
	Instructions  []InstructionResponse `json:"instructions"`
	Logs          []LogResponse         `json:"logs"`
	Balances      []balance             `json:"balances"`
	TokenBalances []tokenBalance        `json:"tokenBalances"`
	Rewards       []reward              `json:"rewards"`
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
	Instructions bool `json:"instructions,omitempty"`
	Logs         bool `json:"logs,omitempty"`
}

type TransactionResponse struct {
	Transaction  transaction   `json:"transaction"`
	Instructions []instruction `json:"instructions"`
	Logs         []logMessage  `json:"logs"`
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
	TransactionTokenBalances bool `json:"transactionTokenbalances,omitempty"`
	Logs                     bool `json:"logs,omitempty"`
	InnerInstructions        bool `json:"innerInstructions,omitempty"`
}

type InstructionResponse struct {
	Instruction              instruction    `json:"instruction"`
	Transaction              *transaction   `json:"transaction"`
	TransactionTokenBalances []tokenBalance `json:"transactionTokenBalances"`
	Logs                     []logMessage   `json:"logs"`
	InnerInstructions        []instruction  `json:"innerInstructions"`
}

type LogRequest struct {
	/* Filters */
	ProgramId []string `json:"programId"`
	Kind      []string `json:"kind"` // 'log' | 'data' | 'other'

	/* Field Selection */
	Transaction bool `json:"transaction,omitempty"`
	Instruction bool `json:"instruction,omitempty"`
}

type LogResponse struct {
	Log         logMessage   `json:"log"`
	Transaction *transaction `json:"transaction"`
	Instruction *instruction `json:"instruction"`
}

type RewardRequest struct {
	PubKey []string `json:"pubkey,omitempty"`
}

type instruction struct {
	// independent of field selectors
	TransactionIndex   uint   `json:"transactionIndex"`
	InstructionAddress []uint `json:"instructionAddress"`

	// can be disabled with field selectors
	ProgramId   string   `json:"programId"`
	Accounts    []string `json:"accounts"`
	Data        string   `json:"data"`
	IsCommitted bool     `json:"isCommitted"`

	// can be enabled with field selectors
	ComputeUnitsConsumed  string  `json:"computeUnitsConsumed"` // TODO will need json parsing
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
	ComputeUnitsConsumed        uint64                 `json:"computeUnitsConsumed"` // TODO will need json parsing
	Fee                         uint64                 `json:"fee"`                  // TODO will need json parsing
	LoadedAddresses             loadedAddresses        `json:"loadedAddresses"`      // request the whole struct with loadedAddresses: true
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

type balance struct {
	// independent of field selectors
	TransactionIndex uint     `json:"transactionIndex"`
	Account          []string `json:"account"`

	// can be disabled with field selectors
	Pre  uint `json:"pre"`
	Post uint `json:"post"`
}

type loadedAddresses struct {
	Readonly []string `json:"readonly"`
	Writable []string `json:"writable"`
}

type addressTableLookup struct {
	// TODO
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
	Height     uint64 `json:"number"` // TODO needs parsing
	ParentHash string `json:"parentHash"`

	// can be disabled with field selectors
	Slot       uint64 `json:"slot"`
	ParentSlot uint64 `json:"parentSlot"` // TODO needs parsing
	Timestamp  int64  `json:"timestamp"`
}

type tokenBalance struct {
	// independent of field selectors
	TransactionIndex uint   `json:"transactionIndex"`
	Account          string `json:"account"`

	// can be disabled with field selectors
	PreMint      string  `json:"preMint"`
	PreDecimals  uint8   `json:"preDecimals"`
	PreOwner     *string `json:"preOwner"`
	PreAmount    uint    `json:"preAmount"`
	PostMint     string  `json:"postMint"`
	PostDecimals uint8   `json:"postDecimals"`
	PostOwner    *string `json:"postOwner"`
	PostAmount   uint    `json:"postAmount"`

	// can be enabled by field selectors
	PostProgramId *string `json:"postProgramId"`
	PreProgramId  *string `json:"preProgramId"`
}
