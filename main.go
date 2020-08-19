package main

import (
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/rpc"
	"github.com/ledgerwatch/turbo-geth/cmd/utils"
	"github.com/ledgerwatch/turbo-geth/log"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	cmd, cfg := rpc.RootCommand()
	if err := utils.SetupCobra(cmd); err != nil {
		panic(err)
	}
	defer utils.StopDebug()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		db, txPool, err := rpc.OpenDB(*cfg)
		if err != nil {
			log.Error("Could not connect to remoteDb", "error", err)
			return nil
		}

		var APIList = GetAPI(db, txPool, cfg.API, cfg.Gascap)
		rpc.StartRpcServer(cmd.Context(), *cfg, APIList)
		return nil
	}

	if err := cmd.ExecuteContext(utils.RootContext()); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
