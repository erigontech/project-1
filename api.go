package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/common/hexutil"
	"github.com/ledgerwatch/turbo-geth/core"
	"github.com/ledgerwatch/turbo-geth/core/rawdb"
	"github.com/ledgerwatch/turbo-geth/core/state"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/core/vm"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/params"
	"github.com/ledgerwatch/turbo-geth/rpc"
	"github.com/ledgerwatch/turbo-geth/turbo/adapter"
	"github.com/ledgerwatch/turbo-geth/turbo/rpchelper"
)

// API - implementation of ExampleApi
type API struct {
	kv ethdb.KV
	db ethdb.Database
}

func NewAPI(kv ethdb.KV, db ethdb.Database) *API {
	return &API{kv: kv, db: db}
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	value, _ := uint256.FromBig((*big.Int)(args.Value))
	gasPrice, _ := uint256.FromBig((*big.Int)(args.GasPrice))
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), value, uint64(*args.Gas), gasPrice, input)
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, value, uint64(*args.Gas), gasPrice, input)
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

// LocalForkRequest represents the request for a local fork
type LocalForkRequest struct {
	Block   rpc.BlockNumberOrHash `json:"block"`
	Txs     []SendTxArgs          `json:"txs"`
	Queries []CallArgs            `json:"queries"`
}

// LocalForkResponse for each transaction returns a receipt, and for each query it returns the result
type LocalForkResponse struct {
	TxResults    []*core.ExecutionResult `json:"txResults"`
	QueryResults []*core.ExecutionResult `json:"queryResults"`
	BlockNumber  uint64                  `json:"blockNumber"`
}

func (api *API) LocalFork(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, txs []*SendTxArgs, queries []*CallArgs) (interface{}, error) {
	tx, err1 := api.db.Begin(ctx, ethdb.RO)
	if err1 != nil {
		return nil, fmt.Errorf("call cannot open tx: %v", err1)
	}
	defer tx.Rollback()
	blockNumber, _, err := rpchelper.GetBlockNumber(blockNrOrHash, tx)
	if err != nil {
		return nil, err
	}
	var stateReader state.StateReader
	if num, ok := blockNrOrHash.Number(); ok && num == rpc.LatestBlockNumber {
		stateReader = state.NewPlainStateReader(tx)
	} else {
		stateReader = state.NewPlainDBState(tx.(ethdb.HasTx).Tx(), blockNumber)
	}
	prevHeaderHash, err := rawdb.ReadCanonicalHash(api.db, blockNumber)
	prevHeader := rawdb.ReadHeader(api.db, prevHeaderHash, blockNumber)
	ibs := state.New(stateReader)
	var header types.Header
	header.Number = big.NewInt(int64(blockNumber) + 1)
	header.Difficulty = big.NewInt(1000000)
	header.Time = prevHeader.Time + 14
	cc := adapter.NewChainContext(api.db)
	var txResults []*core.ExecutionResult
	for i, args := range txs {
		var input []byte
		if args.Input != nil {
			input = *args.Input
		} else if args.Data != nil {
			input = *args.Data
		}
		value, _ := uint256.FromBig((*big.Int)(args.Value))
		gasPrice, _ := uint256.FromBig((*big.Int)(args.GasPrice))
		var nonce uint64
		if args.Nonce != nil {
			nonce = uint64(*args.Nonce)
		}
		msg := types.NewMessage(args.From, args.To, nonce, value, uint64(*args.Gas), gasPrice, input, args.Nonce != nil /* checkNonce */)
		EVMcontext := core.NewEVMContext(msg, &header, cc, nil)
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(EVMcontext, ibs, params.MainnetChainConfig, vm.Config{})
		if execResult, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(uint64(*args.Gas))); err == nil {
			txResults = append(txResults, execResult)
		} else {
			return nil, fmt.Errorf("localFork: transaction %d failed: %v", i, err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		if err := ibs.FinalizeTx(vmenv.ChainConfig().WithEIPsFlags(context.Background(), header.Number), state.NewNoopWriter()); err != nil {
			return nil, fmt.Errorf("localFork: finalizeTx %d: %v\n", i, err)
		}
	}
	var queryResults []*core.ExecutionResult
	for i, args := range queries {
		ibsCopy := ibs.Copy()
		var input []byte
		if args.Input != nil {
			input = *args.Input
		} else if args.Data != nil {
			input = *args.Data
		}
		value, _ := uint256.FromBig((*big.Int)(args.Value))
		gasPrice, _ := uint256.FromBig((*big.Int)(args.GasPrice))
		msg := types.NewMessage(*args.From, args.To, 0, value, uint64(*args.Gas), gasPrice, input, false /* checkNonce */)
		EVMcontext := core.NewEVMContext(msg, &header, cc, nil)
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(EVMcontext, ibsCopy, params.MainnetChainConfig, vm.Config{})
		if execResult, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(uint64(*args.Gas))); err == nil {
			queryResults = append(queryResults, execResult)
		} else {
			return nil, fmt.Errorf("localFork: query %d failed: %v", i, err)
		}
	}
	return LocalForkResponse{TxResults: txResults, QueryResults: queryResults, BlockNumber: blockNumber}, nil
}
