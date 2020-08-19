package main

import (
	"context"

	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/cli"
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/rpc"
)

// Create interface for your API
type ExampleAPI interface {
	TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error)
}

func APIList(kv ethdb.KV, eth ethdb.Backend, cfg *cli.Flags) []rpc.API {
	dbReader := ethdb.NewObjectDatabase(kv)
	api := NewAPI(kv, dbReader)

	customAPIList := []rpc.API{
		{
			Namespace: "example", // replace it by preferred namespace
			Public:    true,
			Service:   ExampleAPI(api),
			Version:   "1.0",
		},
	}

	// Add default TurboGeth api's
	return commands.APIList(kv, eth, *cfg, customAPIList)
}
