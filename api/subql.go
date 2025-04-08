package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/subquery/solana-takoyaki/backend/sqd"
	"github.com/subquery/solana-takoyaki/meta"
	"github.com/subquery/solana-takoyaki/solana"
)

type TransactionsSelector struct {
	Instructions bool `json:"instructions"`
	Logs         bool `json:"logs"`
}

type InstructionsSelector struct {
	Transaction bool `json:"transaction"`
}

type LogsSelector struct {
	Transaction bool `json:"transaction"`
}

type FieldSelector struct {
	Instructions *InstructionsSelector `json:"instructions"`
	Transactions *TransactionsSelector `json:"transactions"`
	Logs         *LogsSelector         `json:"logs"`
}

type TxFilterQuery struct {
	SignerAccountKeys []string `json:"signerAccountKeys"`
}

type InstFilterQuery struct {
	ProgramIds     []string   `json:"programIds"`
	Accounts       [][]string `json:"accounts"`
	Discriminators []string   `json:"discriminators"`
	IsCommitted    bool       `json:"isCommitted"`
}

type LogFilterQuery struct {
	ProgramIds []string `json:"programIds"`
}

type BlockFilter struct {
	Transactions []TxFilterQuery
	Instructions []InstFilterQuery
	Logs         []LogFilterQuery
}

type BlockRequest struct {
	FromBlock     *big.Int
	ToBlock       *big.Int
	Limit         *big.Int
	BlockFilter   *BlockFilter
	FieldSelector *FieldSelector
}

// TODO BlockFilter json methods for bigints

type BlockResult struct {
	Blocks      []*solana.Block `json:"blocks"`
	BlockRange  [2]*big.Int     `json:"blockRange"` // Tuple [start, end]
	GenesisHash string          `json:"genesisHash"`
}

type SubqlApiService struct {
	// networkMeta meta.NetworkMeta
	sqdClient sqd.QueryClient
}

func NewSubqlApiService(
	networkMeta meta.NetworkMeta,
	sqdUrl string,
) (*SubqlApiService, error) {
	return &SubqlApiService{
		// networkMeta,
		sqd.NewSoldexerClient(sqdUrl),
	}, nil
}

func (s *SubqlApiService) FilterBlocksCapabilities(ctx context.Context) (*Capability, error) {
	currentHeight, err := s.sqdClient.CurrentHeight(ctx)
	if err != nil {
		return nil, err
	}

	meta, err := s.sqdClient.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	capabilities := &Capability{
		AvailableBlocks: []AvailableBlocks{{
			meta.StartBlock,
			// s.networkMeta.EarliestSQDBlock,
			currentHeight,
		}},
		SupportedResponses: []string{"basic", "complete"},
		GenesisHash:        meta.GenesisHash,
		ChainId:            meta.ChainId,
		Filters: map[string][]string{
			"transactions": {"signerAccountKeys"},
			"instructions": {"programIds", "discriminator", "accounts", "isCommitted"},
			"logs":         {"programIds", "kind"},
		},
	}
	return capabilities, nil
}

func (s *SubqlApiService) FilterBlocks(ctx context.Context, blockReq BlockRequest) (*BlockResult, error) {
	slog.Debug("Filter Blocks")

	meta, err := s.sqdClient.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	blockResult := &BlockResult{
		GenesisHash: meta.ChainId,
	}

	// Create an initial request to get the relevant block numbers that match the filters
	// This is done because when filtering instructions we aren't able to get all the logs for the transaction, only the instruction
	req := sqd.SolanaRequest{
		Type:      "solana",
		FromBlock: uint(blockReq.FromBlock.Uint64()),
		ToBlock:   uint(blockReq.ToBlock.Uint64()),
		Fields: sqd.Fields{
			Block: map[string]bool{"number": true, "height": true},
		},
		// Empty item means no filter, these will get updated based on the block filters
		Transactions:  []sqd.TransactionRequest{},
		Instructions:  []sqd.InstructionRequest{},
		Rewards:       []sqd.RewardRequest{},
		TokenBalances: []sqd.TokenBalanceRequest{},
		Balances:      []sqd.BalancesRequest{},
		Logs:          []sqd.LogRequest{},
	}

	err = ApplyFiltersToSQDRequest(&req, *blockReq.BlockFilter)
	if err != nil {
		slog.Error("Failed to apply filters", "error", err)
		return nil, err
	}

	// This response always returns the first and last block in the range even if there is no match as a way to indicate the blocks searched.
	res, err := s.sqdClient.Query(ctx, req)
	if err != nil {
		slog.Error("Failed to run filter query", "error", err)
		return nil, err
	}

	slog.Info("Filter blocks", "num blocks", len(res))

	// Fetch all the matching blocks with full content
	blocks := make([]*solana.Block, len(res))
	wg := sync.WaitGroup{}
	errChan := make(chan error)
	for idx, block := range res {
		wg.Add(1)
		go func() {
			req := sqd.SolanaRequest{
				Type:      "solana",
				FromBlock: uint(block.Header.Slot), // Legacy should use height
				ToBlock:   uint(block.Header.Slot), // Legacy should use height
				Fields:    s.sqdClient.GetAllFields(),
				// Empty filters for all data to get a full block
				Transactions:  []sqd.TransactionRequest{{}},
				Instructions:  []sqd.InstructionRequest{{}},
				Rewards:       []sqd.RewardRequest{{}},
				TokenBalances: []sqd.TokenBalanceRequest{{}},
				Balances:      []sqd.BalancesRequest{{}},
				Logs:          []sqd.LogRequest{{}},
			}

			res, err := s.sqdClient.Query(ctx, req)
			if err != nil {
				errChan <- err
				return
			}
			fullBlock, err := sqd.TransformBlock(res[0])
			if err != nil {
				errChan <- fmt.Errorf("Failed to resolve block %v: %v", block.Header.Height, err)
				return
			}
			blocks[idx] = fullBlock
			defer wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err := <-errChan; err != nil {
		slog.Error("Failed to get full block", "error", err)
		return nil, err
	}

	// This response always returns the first and last block in the range even if there is no match as a way to indicate the blocks searched.
	blockResult.BlockRange = [2]*big.Int{
		big.NewInt(int64(res[0].Header.Slot)),
		big.NewInt(int64(res[len(res)-1].Header.Slot)),
	}
	blockResult.Blocks = blocks

	return blockResult, nil
}

func ApplyFiltersToSQDRequest(req *sqd.SolanaRequest, blockFilter BlockFilter) error {
	if len(blockFilter.Transactions) > 0 {
		req.Transactions = []sqd.TransactionRequest{}
		for _, tx := range blockFilter.Transactions {
			req.Transactions = append(req.Transactions, sqd.TransactionRequest{
				FeePayer: tx.SignerAccountKeys,

				// Instructions: true,
				// Logs:         true,
			})
		}
	}

	if len(blockFilter.Instructions) > 0 {
		req.Instructions = []sqd.InstructionRequest{}
		for _, inst := range blockFilter.Instructions {
			instReq := sqd.InstructionRequest{
				ProgramId: inst.ProgramIds,

				IsCommitted: inst.IsCommitted,

				// Transaction:              true,
				// TransactionBalances:      true,
				// TransactionTokenBalances: true,
				// TransactionInstructions:  true,
				// Logs:                     true,
				// InnerInstructions:        true,
			}

			for i, a := range inst.Accounts {
				instReq.SetAccounts(i, a)
			}

			req.Instructions = append(req.Instructions, instReq)
		}
	}

	if len(blockFilter.Logs) > 0 {
		req.Logs = []sqd.LogRequest{}
		for _, log := range blockFilter.Logs {
			req.Logs = append(req.Logs, sqd.LogRequest{
				ProgramId: log.ProgramIds,

				// Transaction: true,
				// Instruction: true,
			})
		}
	}

	return nil
}

func (b *BlockRequest) UnmarshalJSON(data []byte) error {
	type rawBlockFilter struct {
		FromBlock     *hexutil.Big   `json:"fromBlock"`
		ToBlock       *hexutil.Big   `json:"toBlock"`
		Limit         *hexutil.Big   `json:"limit"`
		BlockFilter   *BlockFilter   `json:"blockFilter"`
		FieldSelector *FieldSelector `json:"fieldSelector"`
	}

	var raw rawBlockFilter
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.FromBlock != nil {
		b.FromBlock = (*big.Int)(raw.FromBlock)
	}
	if raw.ToBlock != nil {
		b.ToBlock = (*big.Int)(raw.ToBlock)
	}
	if raw.Limit != nil {
		b.Limit = (*big.Int)(raw.Limit)
	}
	b.BlockFilter = raw.BlockFilter
	b.FieldSelector = raw.FieldSelector

	return nil
}
