package main

import (
	"os"

	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/cli"
	"github.com/ledgerwatch/turbo-geth/cmd/utils"
	"github.com/ledgerwatch/turbo-geth/log"
	"github.com/spf13/cobra"
)

func main() {
	cmd, cfg := cli.RootCommand()
	if err := utils.SetupCobra(cmd); err != nil {
		panic(err)
	}
	defer utils.StopDebug()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		db, txPool, err := cli.OpenDB(*cfg)
		if err != nil {
			log.Error("Could not connect to remoteDb", "error", err)
			return nil
		}

		var APIList = APIList(db, txPool, cfg)
		cli.StartRpcServer(cmd.Context(), *cfg, APIList)
		return nil
	}

	if err := cmd.ExecuteContext(utils.RootContext()); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
