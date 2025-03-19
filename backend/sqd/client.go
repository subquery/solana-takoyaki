package sqd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type Client struct {
	url string
}

func NewClient(url string) *Client {
	return &Client{
		url,
	}
}

// Gets the current height of the dataset
func (c *Client) CurrentHeight() (uint, error) {
	url := fmt.Sprint(c.url, "/height")

	res, err := http.Get(url)
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

func (c *Client) Query(req SolanaRequest) ([]SolanaBlockResponse, error) {
	workerUrl, err := c.getWorkerUrl(req.FromBlock)
	if err != nil {
		return nil, err
	}

	rawReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// fmt.Println("RAW REQ", string(rawReq))

	res, err := http.Post(workerUrl, "application/json", bytes.NewBuffer(rawReq))
	if err != nil {
		return nil, err
	}

	rawRes, err := io.ReadAll(res.Body)
	if err != nil {
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

func (c *Client) getWorkerUrl(startBlock uint) (string, error) {

	url := fmt.Sprintf("%s/%v/worker", c.url, startBlock)

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad response code: %s", res.Status)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	bodyString := string(resBody)
	return bodyString, nil
}

func GetSquidUrl(network string) (string, error) {
	res, err := http.Get(EvmRegistry)
	if err != nil {
		return "", err
	}
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
