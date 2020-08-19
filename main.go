package main

import (
	"context"
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/cli"
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/log"
	"github.com/ledgerwatch/turbo-geth/rpc"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	rpc.SetupDefaultLogger(log.LvlInfo)

	cmd, cfg := rpc.RootCommand()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		db, txPool, err := rpc.DefaultConnection(cfg)
		if err != nil {
			log.Error("Could not connect to remoteDb", "error", err)
			return nil
		}

		var rpcAPI = GetAPI(db, txPool, cfg.API, cfg.Gascap)
		rpc.StartRpcServer(cfg, rpcAPI)
		sig := <-cmd.Context().Done()
		log.Info("Exiting...", "signal", sig)
		return nil
	}

	if err := cmd.ExecuteContext(rpc.RootContext()); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

type MyAPI interface {
	TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error)
}

func GetAPI(db ethdb.KV, eth ethdb.Backend, enabledApis []string) []rpc.API {
	var rpcAPI []rpc.API
	dbReader := ethdb.NewObjectDatabase(db)
	api := NewAPI(db, dbReader)

	for _, enabledAPI := range enabledApis {
		switch enabledAPI {
		case "magic":
			rpcAPI = append(rpcAPI, rpc.API{
				Namespace: "eth",
				Public:    true,
				Service:   MyAPI(api),
				Version:   "1.0",
			})

		default:
			log.Error("Unrecognised", "api", enabledAPI)
		}
	}
	return rpcAPI
}
