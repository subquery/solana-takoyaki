package meta

type NetworkMeta struct {
	ChainId          string
	GenesisHash      string
	EarliestSQDBlock uint
}

// EarliestSQDBlock can be found here https://docs.sqd.ai/subsquid-network/reference/networks/#solana-and-compatibles

var MAINNET = NetworkMeta{
	ChainId:          "mainnet",
	GenesisHash:      "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d",
	EarliestSQDBlock: 269_828_500,
}

var ECLIPSE_MAINNET = NetworkMeta{
	ChainId:          "eclipse-mainnet",
	GenesisHash:      "", // TODO
	EarliestSQDBlock: 24_641_070,
}
