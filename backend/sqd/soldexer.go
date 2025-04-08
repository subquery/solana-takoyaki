package sqd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/subquery/solana-takoyaki/meta"
)

// The legacy version had much more default to true values
var ALL_SOLDEXER_FIELDS = Fields{
	Instruction: map[string]bool{
		"transactionIndex":   true,
		"instructionAddress": true,
		"programId":          true,
		"data":               true,
		"accounts":           true,
		"isCommitted":        true,
	},
	Transaction: map[string]bool{
		"transactionIndex":            true,
		"accountKeys":                 true,
		"loadedAddresses":             true,
		"feePayer":                    true,
		"fee":                         true,
		"err":                         true,
		"signatures":                  true,
		"numReadonlySignedAccounts":   true,
		"numReadonlyUnsignedAccounts": true,
		"numRequiredSignatures":       true,
		"addressTableLookups":         true,
		"computeUnitsConsumed":        true,
		// "recentBlockhash":             true, // Doesn't work, RPC returns an error, possibly a typeo on the SQD query service: recentBlockhash should be recentBlockHash
	},
	Log: map[string]bool{
		"transactionIndex":   true,
		"logIndex":           true,
		"instructionAddress": true,
		"programId":          true,
		"kind":               true,
		"message":            true,
	},
	Reward: map[string]bool{
		"pubkey":      true,
		"lamports":    true,
		"rewardType":  true,
		"postBalance": true,
		"commission":  true,
	},
	Block: map[string]bool{
		"hash":         true,
		"number":       true, // Slot, differs from legacy archive
		"height":       true,
		"parentHash":   true,
		"parentNumber": true, // Parent Slot, differs from legacy archive
		"timestamp":    true,
	},
	TokenBalance: map[string]bool{
		"account":          true,
		"transactionIndex": true,
		"preMint":          true,
		"preDecimals":      true,
		"preOwner":         true,
		"preAmount":        true,
		"postMint":         true,
		"postDecimals":     true,
		"postOwner":        true,
		"postAmount":       true,
		"postProgramId":    true,
		"preProgramId":     true,
	},
	Balance: map[string]bool{
		"transactionIndex": true,
		"account":          true,
		"pre":              true,
		"post":             true,
	},
}

type headResponse struct {
	Number uint   `json:"number"`
	Hash   string `json:"hash"`
}

type metaResponse struct {
	Dataset    string   `json:"dataset"`
	Aliases    []string `json:"aliases"`
	RealTime   bool     `json:"real_time"`
	StartBlock uint     `json:"start_block"`
}

type SoldexerClient struct {
	baseUrl string
	meta    *NetworkMeta
}

func NewSoldexerClient(baseUrl string) *SoldexerClient {
	return &SoldexerClient{
		baseUrl,
		nil,
	}
}

func (c *SoldexerClient) GetAllFields() Fields {
	return ALL_SOLDEXER_FIELDS
}

func (c *SoldexerClient) CurrentHeight(ctx context.Context) (uint, error) {
	url, err := url.JoinPath(c.baseUrl, "/head")
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Bad response code: %s", res.Status)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	headRes := &headResponse{}
	err = json.Unmarshal(resBody, headRes)
	if err != nil {
		return 0, err
	}

	return headRes.Number, nil
}

func (c *SoldexerClient) Metadata(ctx context.Context) (*NetworkMeta, error) {
	if c.meta != nil {
		return c.meta, nil
	}
	url, err := url.JoinPath(c.baseUrl, "/metadata")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad response code: %s", res.Status)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	metaRes := &metaResponse{}
	err = json.Unmarshal(resBody, metaRes)
	if err != nil {
		return nil, err
	}

	meta := &NetworkMeta{
		StartBlock:  metaRes.StartBlock,
		ChainId:     metaRes.Aliases[0],
		GenesisHash: meta.MAINNET.GenesisHash, // TODO make configurable
	}

	c.meta = meta

	return meta, nil
}

func (c *SoldexerClient) Query(ctx context.Context, solReq SolanaRequest) ([]SolanaBlockResponse, error) {
	url, err := url.JoinPath(c.baseUrl, "/stream")
	if err != nil {
		return nil, err
	}

	rawReq, err := json.Marshal(solReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(rawReq))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to run query", "error", err)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		rawRes, err := io.ReadAll(res.Body)
		if err != nil {
			slog.Error("failed to read query body", "error", err)
			return nil, fmt.Errorf("Bad response code: %s\n%v", res.Status, "Failed to read body")
		}
		return nil, fmt.Errorf("Bad response code: %s\n%v", res.Status, string(rawRes))
	}

	solanaRes := []SolanaBlockResponse{}

	dec := json.NewDecoder(res.Body)

	// Read JSON values one at a time
	for {
		var item SolanaBlockResponse
		if err := dec.Decode(&item); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		solanaRes = append(solanaRes, item)
	}

	return solanaRes, nil
}
