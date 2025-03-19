package api

import (
	"fmt"

	"github.com/subquery/solana-takoyaki/backend/sqd"
	"github.com/subquery/solana-takoyaki/meta"
)

type SubqlApiService struct {
	networkMeta meta.NetworkMeta
	sqdClient   *sqd.Client
}

func NewSubqlApiService(
	networkMeta meta.NetworkMeta,
	sqdUrl string,
) (*SubqlApiService, error) {
	return &SubqlApiService{
		networkMeta,
		sqd.NewClient(sqdUrl),
	}, nil
}

func (s *SubqlApiService) FilterBlocksCapabilities() (*Capability, error) {
	currentHeight, err := s.sqdClient.CurrentHeight()
	if err != nil {
		return nil, err
	}

	capabilities := &Capability{
		AvailableBlocks: []AvailableBlocks{{
			s.networkMeta.EarliestSQDBlock,
			currentHeight,
		}},
		SupportedResponses: []string{"basic", "complete"},
		GenesisHash:        s.networkMeta.GenesisHash,
		ChainId:            s.networkMeta.ChainId,
		Filters: map[string][]string{
			"transactions": {"signerAccountKey"},
			"instructions": {"programId", "type", "isCommitted"},
			"logs":         {"programId", "kind"},
		},
	}
	return capabilities, nil
}

func (s *SubqlApiService) FilterBlocks() error {
	return fmt.Errorf("not implemented")
}
