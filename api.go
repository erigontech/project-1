package main

import (
	"context"
	"fmt"

	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/core/rawdb"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/params"
	"github.com/ledgerwatch/turbo-geth/rpc"
	"github.com/ledgerwatch/turbo-geth/turbo/adapter"
	"github.com/ledgerwatch/turbo-geth/turbo/adapter/ethapi"
	"github.com/ledgerwatch/turbo-geth/turbo/transactions"
)

// API - implementation of ExampleApi
type API struct {
	kv ethdb.KV
	db ethdb.Getter
}

func NewAPI(kv ethdb.KV, db ethdb.Getter) *API {
	return &API{kv: kv, db: db}
}

// GetBlockByNumber see https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber
// see internal/ethapi.PublicBlockChainAPI.GetBlockByNumber
func (api *API) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	additionalFields := make(map[string]interface{})

	block := rawdb.ReadBlockByNumber(api.db, uint64(number.Int64()))
	if block == nil {
		return nil, fmt.Errorf("block not found: %d", number.Int64())
	}

	additionalFields["totalDifficulty"] = rawdb.ReadTd(api.db, block.Hash(), uint64(number.Int64()))
	response, err := ethapi.RPCMarshalBlock(block, true, fullTx, additionalFields)

	if err == nil && number == rpc.PendingBlockNumber {
		// Pending blocks need to nil out a few fields
		for _, field := range []string{"hash", "nonce", "miner"} {
			response[field] = nil
		}
	}
	return response, err
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *API) TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error) {
	// Retrieve the transaction and assemble its EVM context
	tx, blockHash, _, txIndex := rawdb.ReadTransaction(api.db, hash)
	if tx == nil {
		return nil, fmt.Errorf("transaction %#x not found", hash)
	}
	bc := adapter.NewBlockGetter(api.db)
	cc := adapter.NewChainContext(api.db)
	msg, vmctx, ibs, _, err := transactions.ComputeTxEnv(ctx, bc, params.MainnetChainConfig, cc, api.kv, blockHash, txIndex)
	if err != nil {
		return nil, err
	}
	// Trace the transaction and return
	return transactions.TraceTx(ctx, msg, vmctx, ibs, config)
}
