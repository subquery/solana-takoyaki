package api

// type BlockResult struct {
// 	Blocks      []*Block        `json:"blocks"`
// 	BlockRange  [2]*hexutil.Big `json:"blockRange"` // Tuple [start, end]
// 	GenesisHash string          `json:"genesisHash"`
// }

// type Block struct {
// 	Header       *types.Header            `json:"header"`
// 	Transactions []*ethapi.RPCTransaction `json:"transactions,omitempty"`
// 	Logs         []*types.Log             `json:"logs,omitempty"`
// }

// type BlockRequest struct {
// 	FromBlock     *rpc.BlockNumber `json:"fromBlock"`
// 	ToBlock       *rpc.BlockNumber `json:"toBlock"`
// 	Limit         *hexutil.Big     `json:"limit"`
// 	BlockFilter   EntityFilter     `json:"blockFilter,omitempty"`
// 	FieldSelector *FieldSelector   `json:"fieldSelector"`
// }

// type LogsSelector struct {
// 	Transaction bool `json:"transaction"`
// 	// TODO add specific fields
// }

// type TransactionsSelector struct {
// 	Log bool `json:"log"`
// 	// TODO add specific fields
// }

// type FieldSelector struct {
// 	Logs         *LogsSelector         `json:"logs"`
// 	Transactions *TransactionsSelector `json:"transactions"`
// }

// type FieldFilter map[string][]interface{}

// type EntityFilter map[string][]FieldFilter

type AvailableBlocks struct {
	StartHeight uint `json:"startHeight"`
	EndHeight   uint `json:"endHeight"`
}

type Capability struct {
	AvailableBlocks    []AvailableBlocks   `json:"availableBlocks"`
	Filters            map[string][]string `json:"filters"`
	SupportedResponses []string            `json:"supportedResponses"`
	GenesisHash        string              `json:"genesisHash"`
	ChainId            string              `json:"chainId"`
}
