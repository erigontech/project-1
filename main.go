package main

import (
	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/rpc"
	"github.com/ledgerwatch/turbo-geth/log"
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

		var APIList = GetAPI(db, txPool, cfg.API, cfg.Gascap)
		rpc.StartRpcServer(cfg, APIList)
		sig := <-cmd.Context().Done()
		log.Info("Exiting...", "signal", sig)
		return nil
	}

	if err := cmd.ExecuteContext(rpc.RootContext()); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
