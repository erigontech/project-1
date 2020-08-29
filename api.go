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
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/eth/stagedsync/stages"
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
}

func (api *API) LocalFork(ctx context.Context, number rpc.BlockNumber, txs []*SendTxArgs, queries []*CallArgs) (interface{}, error) {
	var blockNum uint64
	if number == rpc.LatestBlockNumber {
		var err error
		blockNum, _, err = stages.GetStageProgress(api.db, stages.Execution)
		if err != nil {
			return nil, fmt.Errorf("lockFork, getting latest block number: %v", err)
		}
	} else if number == rpc.PendingBlockNumber || number == rpc.EarliestBlockNumber {
		return nil, fmt.Errorf("localFork, pending and earliest blocks are not supported")
	} else {
		blockNum = uint64(number.Int64())
	}
	fmt.Printf("Blocknum: %d\n", blockNum)
	prevHeaderHash := rawdb.ReadCanonicalHash(api.db, blockNum)
	prevHeader := rawdb.ReadHeader(api.db, prevHeaderHash, blockNum)
	reader := adapter.NewStateReader(api.kv, blockNum)
	ibs := state.New(reader)
	var header types.Header
	header.Number = big.NewInt(int64(blockNum) + 1)
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
		msg := types.NewMessage(args.From, args.To, uint64(*args.Nonce), value, uint64(*args.Gas), gasPrice, input, true /* checkNonce */)
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
		if err := ibs.FinalizeTx(vmenv.ChainConfig().WithEIPsFlags(context.Background(), header.Number), reader); err != nil {
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
	return LocalForkResponse{TxResults: txResults, QueryResults: queryResults}, nil
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
