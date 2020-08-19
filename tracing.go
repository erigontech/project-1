package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ledgerwatch/turbo-geth/cmd/rpcdaemon/rpc"
	"github.com/ledgerwatch/turbo-geth/common"
	"github.com/ledgerwatch/turbo-geth/consensus"
	"github.com/ledgerwatch/turbo-geth/core/rawdb"
	"github.com/ledgerwatch/turbo-geth/core/state"
	"github.com/ledgerwatch/turbo-geth/core/types"
	"github.com/ledgerwatch/turbo-geth/eth"
	"github.com/ledgerwatch/turbo-geth/ethdb"
	"github.com/ledgerwatch/turbo-geth/params"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second
)

type API struct {
	db       ethdb.KV
	dbReader ethdb.Getter
}

func NewAPI(db ethdb.KV, dbReader ethdb.Getter) *API {
	return &API{
		db:       db,
		dbReader: dbReader,
	}
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *API) TraceTransaction(ctx context.Context, hash common.Hash, config *eth.TraceConfig) (interface{}, error) {
	// Retrieve the transaction and assemble its EVM context
	tx, blockHash, _, txIndex := rawdb.ReadTransaction(api.dbReader, hash)
	if tx == nil {
		return nil, fmt.Errorf("transaction %#x not found", hash)
	}
	msg, vmctx, ibs, _, err := eth.ComputeTxEnv(ctx, &blockGetter{api.dbReader}, params.MainnetChainConfig, &chainContext{db: api.dbReader}, api.db, blockHash, txIndex)
	if err != nil {
		return nil, err
	}
	// Trace the transaction and return
	return eth.TraceTx(ctx, msg, vmctx, ibs, config)
}

type chainContext struct {
	db rawdb.DatabaseReader
}

func NewChainContext(db rawdb.DatabaseReader) *chainContext {
	return &chainContext{
		db: db,
	}

}

type powEngine struct {
}

func (c *powEngine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	panic("must not be called")
}
func (c *powEngine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (func(), <-chan error) {
	panic("must not be called")
}
func (c *powEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	panic("must not be called")
}
func (c *powEngine) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	panic("must not be called")
}
func (c *powEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	panic("must not be called")
}
func (c *powEngine) Finalize(chainConfig *params.ChainConfig, header *types.Header, state *state.IntraBlockState, txs []*types.Transaction, uncles []*types.Header) {
	panic("must not be called")
}
func (c *powEngine) FinalizeAndAssemble(chainConfig *params.ChainConfig, header *types.Header, state *state.IntraBlockState, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	panic("must not be called")
}
func (c *powEngine) Seal(_ consensus.Cancel, chain consensus.ChainHeaderReader, block *types.Block, results chan<- consensus.ResultWithContext, stop <-chan struct{}) error {
	panic("must not be called")
}
func (c *powEngine) SealHash(header *types.Header) common.Hash {
	panic("must not be called")
}
func (c *powEngine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	panic("must not be called")
}
func (c *powEngine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	panic("must not be called")
}

func (c *powEngine) Close() error {
	panic("must not be called")
}

func (c *powEngine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (c *chainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	return rawdb.ReadHeader(c.db, hash, number)
}

func (c *chainContext) Engine() consensus.Engine {
	return &powEngine{}
}

type blockGetter struct {
	dbReader rawdb.DatabaseReader
}

func (g *blockGetter) GetBlockByHash(hash common.Hash) *types.Block {
	return rawdb.ReadBlockByHash(g.dbReader, hash)

}

func (g *blockGetter) GetBlock(hash common.Hash, number uint64) *types.Block {
	return rawdb.ReadBlock(g.dbReader, hash, number)
}
