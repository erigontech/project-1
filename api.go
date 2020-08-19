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

type MyAPI interface {
	TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error)
}

func GetAPI(db ethdb.KV, eth ethdb.Backend, enabledApis []string, gascap uint64) []rpc.API {
	dbReader := ethdb.NewObjectDatabase(db)
	api := NewAPI(db, dbReader)

	var customAPIList []rpc.API

	for _, enabledAPI := range enabledApis {
		switch enabledAPI {
		case "magic":
			customAPIList = append(customAPIList, rpc.API{
				Namespace: "eth",
				Public:    true,
				Service:   MyAPI(api),
				Version:   "1.0",
			})
		default:
			log.Error("Unrecognised", "api", enabledAPI)
		}
	}

	defaultAPIList := commands.GetAPI(db, eth, enabledApis, gascap)
	return append(defaultAPIList, customAPIList...)
}
