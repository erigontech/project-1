package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/node"
	"github.com/ledgerwatch/turbo-geth/rpc"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

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

func daemon(cmd *cobra.Command, cfg commands.Config) {
	vhosts := splitAndTrim(cfg.HttpVirtualHost)
	cors := splitAndTrim(cfg.HttpCORSDomain)
	enabledApis := splitAndTrim(cfg.API)

	var db ethdb.KV
	var txPool ethdb.Backend
	var err error
	if cfg.PrivateApiAddr != "" {
		db, txPool, err = ethdb.NewRemote().Path(cfg.PrivateApiAddr).Open()
		if err != nil {
			log.Error("Could not connect to remoteDb", "error", err)
			return
		}
	} else if cfg.Chaindata != "" {
		if database, errOpen := ethdb.Open(cfg.Chaindata); errOpen == nil {
			db = database.KV()
		} else {
			err = errOpen
		}
	} else {
		err = fmt.Errorf("either remote db or bolt db must be specified")
	}

	if err != nil {
		log.Error("Could not connect to remoteDb", "error", err)
		return
	}

	var rpcAPI = GetAPI(db, txPool, enabledApis)

	httpEndpoint := fmt.Sprintf("%s:%d", cfg.HttpListenAddress, cfg.HttpPort)

	// register apis and create handler stack
	srv := rpc.NewServer()
	err = node.RegisterApisFromWhitelist(rpcAPI, enabledApis, srv, false)
	if err != nil {
		log.Error("Could not start register RPC apis", "error", err)
		return
	}
	handler := node.NewHTTPHandlerStack(srv, cors, vhosts)

	listener, _, err := node.StartHTTPEndpoint(httpEndpoint, rpc.DefaultHTTPTimeouts, handler)
	if err != nil {
		log.Error("Could not start RPC api", "error", err)
		return
	}
	extapiURL := fmt.Sprintf("http://%s", httpEndpoint)
	log.Info("HTTP endpoint opened", "url", extapiURL)

	defer func() {
		listener.Close()
		log.Info("HTTP endpoint closed", "url", httpEndpoint)
	}()

	sig := <-cmd.Context().Done()
	log.Info("Exiting...", "signal", sig)
}

var (
	cfg commands.Config
)

func init() {
	rootCmd.Flags().StringVar(&cfg.PrivateApiAddr, "private.api.addr", "", "private api network address, for example: 127.0.0.1:9090, empty string means not to start the listener. do not expose to public network. serves remote database interface")
	rootCmd.Flags().StringVar(&cfg.Chaindata, "chaindata", "", "path to the database")
	rootCmd.Flags().StringVar(&cfg.HttpListenAddress, "http.addr", node.DefaultHTTPHost, "HTTP-RPC server listening interface")
	rootCmd.Flags().IntVar(&cfg.HttpPort, "http.port", node.DefaultHTTPPort, "HTTP-RPC server listening port")
	rootCmd.Flags().StringVar(&cfg.HttpCORSDomain, "http.corsdomain", "", "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	rootCmd.Flags().StringVar(&cfg.HttpVirtualHost, "http.vhosts", strings.Join(node.DefaultConfig.HTTPVirtualHosts, ","), "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.")
	rootCmd.Flags().StringVar(&cfg.API, "http.api", "", "API's offered over the HTTP-RPC interface")
	rootCmd.Flags().Uint64Var(&cfg.Gascap, "rpc.gascap", 0, "Sets a cap on gas that can be used in eth_call/estimateGas")
}

var rootCmd = &cobra.Command{
	Use:   "rpcdaemon",
	Short: "rpcdaemon is JSON RPC server that connects to turbo-geth node for remote DB access",
	RunE: func(cmd *cobra.Command, args []string) error {
		daemon(cmd, cfg)
		return nil
	},
}

func main() {
	var (
		ostream log.Handler
		glogger *log.GlogHandler
	)

	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger = log.NewGlogHandler(ostream)
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.LvlInfo)

	commands.Execute()
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}
