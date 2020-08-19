package main

import (
	"context"
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/log"
	"github.com/ledgerwatch/turbo-geth/rpc"
)

// Create interface for your API
type ExampleAPI interface {
	TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error)
}

func GetAPI(kv ethdb.KV, eth ethdb.Backend, enabledApis []string, gascap uint64) []rpc.API {
	dbReader := ethdb.NewObjectDatabase(kv)
	api := NewAPI(kv, dbReader)

	var customAPIList []rpc.API

	for _, enabledAPI := range enabledApis {
		switch enabledAPI {
		case "example":
			customAPIList = append(customAPIList, rpc.API{
				Namespace: "example", // replace it by preferred namespace
				Public:    true,
				Service:   ExampleAPI(api),
				Version:   "1.0",
			})
		default:
			log.Error("Unrecognised", "api", enabledAPI)
		}
	}

	// Add default TurboGeth api's
	defaultAPIList := commands.GetAPI(kv, eth, enabledApis, gascap)
	return append(defaultAPIList, customAPIList...)
}
