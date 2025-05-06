package sqd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/subquery/solana-takoyaki/meta"
)

type Client struct {
	url  string
	meta meta.NetworkMeta
}

type QueryClient interface {
	Metadata(ctx context.Context) (*NetworkMeta, error)
	CurrentHeight(ctx context.Context) (uint, error)
	Query(ctx context.Context, solReq SolanaRequest, limit *int) ([]SolanaBlockResponse, error)
	GetAllFields() Fields
}

func NewClient(url string, meta meta.NetworkMeta) *Client {
	return &Client{
		url,
		meta,
	}
}

var ALL_FIELDS = Fields{
	Instruction: map[string]bool{
		"programId": true,
		"data":      true,
		"accounts":  true,
		// "instructionAddress": true,
	},
	Transaction: map[string]bool{
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
		// "recentBlockHash":             true, // Doesn't work, RPC returns an error, possibly a typeo on the SQD query service: recentBlockhash should be recentBlockHash
	},
	Log: map[string]bool{
		"kind":      true,
		"programId": true,
		"message":   true,
	},
	Reward: map[string]bool{
		"rewardType":  true,
		"lamports":    true,
		"postBalance": true,
	},
	Block: map[string]bool{
		"parentHash": true,
		"slot":       true,
		"parentSlot": true,
		"timestamp":  true,
	},
	TokenBalance: map[string]bool{
		"preMint":       true,
		"preDecimals":   true,
		"preOwner":      true,
		"preAmount":     true,
		"postMint":      true,
		"postDecimals":  true,
		"postOwner":     true,
		"postAmount":    true,
		"postProgramId": true,
		"preProgramId":  true,
	},
	Balance: map[string]bool{
		"pre":  true,
		"post": true,
	},
}

func (c *Client) GetAllFields() Fields {
	return ALL_FIELDS
}

func (c *Client) Metadata(ctx context.Context) *NetworkMeta {
	return &NetworkMeta{
		GenesisHash: c.meta.GenesisHash,
		StartBlock:  c.meta.EarliestSQDBlock,
		ChainId:     c.meta.ChainId,
	}
}

// Gets the current height of the dataset
// NOTE this returns the block height not the slot
func (c *Client) CurrentHeight(ctx context.Context) (uint, error) {
	url := fmt.Sprint(c.url, "/height")

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
	bodyString := string(resBody)

	currentHeight, err := strconv.ParseUint(bodyString, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(currentHeight), nil
}

func (c *Client) Query(ctx context.Context, solReq SolanaRequest, limit *int) ([]SolanaBlockResponse, error) {
	workerUrl, err := c.getWorkerUrl(ctx, solReq.FromBlock)
	if err != nil {
		slog.Error("Failed to get worker url", "error", err)
		return nil, err
	}

	rawReq, err := json.Marshal(solReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", workerUrl, bytes.NewBuffer(rawReq))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("request failed: %v", err)
		return nil, err
	}

	defer res.Body.Close()

	rawRes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("request failed to read body: %v", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad response code: %s\n%v", res.Status, string(rawRes))
	}

	// fmt.Println("RAW RES", string(rawRes))

	solanaRes := &[]SolanaBlockResponse{}
	err = json.Unmarshal(rawRes, solanaRes)
	if err != nil {
		return nil, err
	}

	return *solanaRes, nil
}

func (c *Client) getWorkerUrl(ctx context.Context, startBlock uint) (string, error) {

	url := fmt.Sprintf("%s/%v/worker", c.url, startBlock)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Failed to get worker url", "error", err, "url", url)
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read worker url response", "error", err)
		return "", err
	}
	bodyString := string(resBody)

	if res.StatusCode != http.StatusOK {
		slog.Error("Bad response code", "error", err, "url", url, "status", res.Status)
		return "", fmt.Errorf("Bad response code: %s, message: %s", res.Status, bodyString)
	}

	return bodyString, nil
}

func GetSquidUrl(ctx context.Context, network string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", EvmRegistry, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	archiveRes := &ArchiveRegistryResponse{}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(resBody, archiveRes)
	if err != nil {
		return "", err
	}
	for i := range archiveRes.Archives {
		archive := archiveRes.Archives[i]
		if archive.Network == network {
			for j := range archive.Providers {
				p := archive.Providers[j]
				// if p.Release == string(release) {
				return p.DataSourceUrl, nil
				// }
			}
		}
	}
	return "", fmt.Errorf("not found")
}
