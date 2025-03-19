package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/subquery/solana-takoyaki/api"
	"github.com/subquery/solana-takoyaki/backend/sqd"
	"github.com/subquery/solana-takoyaki/meta"
)

func main() {

	port := flag.Uint("port", 8080, "Port to listen on")
	// sqdEndpoint := flag.String("sqdEndpoint", "https://v2.archive.subsquid.io/network/solana-mainnet", "SQD archive endpoint")

	flag.Parse()

	sqdUrl, err := sqd.GetSquidUrl("solana-mainnet")
	if err != nil {
		fmt.Printf("Failed to get SQD url: %v", err)
		panic(1)
	}

	subqlApi, err := api.NewSubqlApiService(meta.MAINNET, sqdUrl)
	if err != nil {
		fmt.Println("Error creating subql rpc service", err)
		panic(1)
	}

	server := rpc.NewServer()
	err = server.RegisterName("subql", subqlApi)
	if err != nil {
		fmt.Println("Error registering subql rpc service", err)
		panic(1)
	}

	addr := fmt.Sprintf(":%v", *port)
	http.Handle("/", server)
	fmt.Printf("Starting HTTP server on %v\n", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("HTTP server failed: %v", err)
		panic(1)
	}
}
