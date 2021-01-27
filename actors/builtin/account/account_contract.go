package account

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"math/big"

	"github.com/filecoin-project/go-address"
	lotusbig "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/state"
	"github.com/filestar-project/evm-adapter/evm/types"

	rtt "github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
)

// PathToRepo path to repo from command params
var PathToRepo string

type databaseSingleton struct {
	levelDB types.Database
}

var database *databaseSingleton
var once sync.Once

type EvmLogs struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address types.Address
	// list of topics provided by the contract.
	Topics []byte
	// supplied by the contract, usually ABI-encoded
	Data []byte

	// The Removed field is true if this log was reverted due to a chain reorganisation.
	// You must pay attention to this field if you receive logs through a filter query.
	Removed bool
}

func newEvmLogs(logs []types.Log) []EvmLogs {
	result := make([]EvmLogs, len(logs))
	for i, log := range logs {
		result[i].Address = log.Address
		result[i].Data = log.Data
		result[i].Removed = log.Removed
		result[i].Topics = []byte{}
		for _, topic := range log.Topics {
			result[i].Topics = append(result[i].Topics, topic.Bytes()...)
		}
	}
	return result
}

func getFormatLogs(logs []types.Log) string {
	result := ""
	sep := ", "
	for _, logData := range logs {
		result += fmt.Sprintf("{Address:\"%x\"", logData.Address.Bytes()) + sep
		result += fmt.Sprintf("Data:%x", logData.Data) + sep
		result += fmt.Sprintf("Removed:%t", logData.Removed) + sep
		result += "{"
		for i, topic := range logData.Topics {
			result += fmt.Sprintf("Topic%v:%x", i, topic.Bytes()) + sep
		}
		result += "}}"
	}
	return result
}

func getLevelDB(rt runtime.Runtime) *databaseSingleton {
	once.Do(func() {
		if len(PathToRepo) == 0 {
			// if no path provided, open LevelDB in tmp random dir
			randBytes := make([]byte, 16)
			if _, err := rand.Read(randBytes); err != nil {
				rt.Abortf(exitcode.ErrIllegalState, "Failed open LevelDB for EVM %v", err)
				return
			}
			PathToRepo = filepath.Join(os.TempDir(), hex.EncodeToString(randBytes))
			if err := os.MkdirAll(PathToRepo, os.ModePerm); err != nil {
				rt.Abortf(exitcode.ErrIllegalState, "Failed open LevelDB for EVM %v", err)
				return
			}
		}
		db, err := state.OpenDatabase(PathToRepo)
		if err != nil {
			rt.Abortf(exitcode.ErrIllegalState, "Failed open LevelDB for EVM %v", err)
			return
		}
		rt.Log(rtt.DEBUG, "Opened LevelDB for Ethereum VM on path %v", PathToRepo)
		database = &databaseSingleton{db}
	})
	return database
}

type ContractParams struct {
	Code  []byte
	Salt  []byte
	Value lotusbig.Int
}

type ContractResult struct {
	Value   []byte
	GasUsed int64
	Logs    []EvmLogs
}

// Creates new EVM configuration
func newEvmConfig(rt runtime.Runtime, params *ContractParams, commitStatus bool) *types.Config {
	// fake for now
	caller := types.BytesToAddress(rt.Origin().Payload())
	db := getLevelDB(rt).levelDB
	root, err := state.GetRoot(db)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed open LevelDB for EVM (can't get root) %v", err)
		return nil
	}
	return &types.Config{
		// This fields used by Solidity opcodes:
		BlockNumber:     big.NewInt(100),
		BlockTime:       big.NewInt(0),
		BlockDifficulty: big.NewInt(0),
		BlockGasLimit:   1000000000,
		BlockCoinbase:   types.BytesToAddress([]byte{121}),
		Caller:          caller,
		GasPrice:        big.NewInt(1),

		// Used for contract creation
		Salt: params.Salt,

		// This fields used by logs:
		BlockHash:    types.BytesToHash([]byte{123}),
		TrxHash:      types.BytesToHash([]byte{121}),
		TrxIndex:     321,
		CommitStatus: commitStatus,
		// Root hash of stateDB
		RootHash: root,
		// Database for EVM
		Database: db,
	}
}

func (a Actor) CreateContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
	return a.createContract(rt, params, true)
}

func (a Actor) CreateContractWithoutCommit(rt runtime.Runtime, params *ContractParams) *ContractResult {
	return a.createContract(rt, params, false)
}

// Creates new EVM contract
func (a Actor) createContract(rt runtime.Runtime, params *ContractParams, commitStatus bool) *ContractResult {
	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()
	if rt.OriginReciever().Protocol() != address.SECP256K1 {
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 addresses allowed! Current address protocol: %v", rt.OriginReciever().Protocol())
	}

	config := newEvmConfig(rt, params, commitStatus)

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	evm, err := evm.NewEVM(adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new EVM object %v", err)
		return nil
	}

	// instruct EVM to create the contract
	gasLimit := rt.GasLimit()
	result, err := evm.CreateContract(params.Code, uint64(gasLimit), params.Value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = int64(gasLimit - result.GasLeft)
	// charge gas counted by EVM for contract creation
	rt.ChargeGas("OnCreateContract", ret.GasUsed, 0)
	// Add logs from evm
	ret.Logs = newEvmLogs(evm.GetLogs())
	rt.Log(rtt.INFO, getFormatLogs(evm.GetLogs()))
	if bytes.Equal(config.RootHash.Bytes(), result.Root.Bytes()) {
		return ret
	}
	// Save root to database
	if err := state.SaveRoot(config.Database, result.Root.FixedBytes()); err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed to save EVM root to database %v", err)
		return nil
	}
	return ret
}

func (a Actor) CallContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
	return a.callContract(rt, params, true)
}

func (a Actor) CallContractWithoutCommit(rt runtime.Runtime, params *ContractParams) *ContractResult {
	return a.callContract(rt, params, false)
}

// Call EVM contract
func (a Actor) callContract(rt runtime.Runtime, params *ContractParams, commitStatus bool) *ContractResult {
	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CallContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()
	if rt.OriginReciever().Protocol() != address.SECP256K1 {
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 addresses allowed! Current address protocol: %v", rt.OriginReciever().Protocol())
	}

	config := newEvmConfig(rt, params, commitStatus)

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	evm, err := evm.NewEVM(adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new EVM object %v", err)
		return nil
	}

	// instruct EVM to call the contract
	gasLimit := rt.GasLimit()
	receiver := types.BytesToAddress(rt.OriginReciever().Payload())
	result, err := evm.CallContract(receiver, params.Code, uint64(gasLimit), params.Value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed call contract, got %v", err)
		return nil
	}

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = int64(gasLimit - result.GasLeft)
	// charge gas counted by EVM for this call
	rt.ChargeGas("OnCallContract", ret.GasUsed, 0)
	// Add logs from evm
	ret.Logs = newEvmLogs(evm.GetLogs())
	rt.Log(rtt.INFO, getFormatLogs(evm.GetLogs()))
	if bytes.Equal(config.RootHash.Bytes(), result.Root.Bytes()) {
		return ret
	}
	// Save root to database
	if err := state.SaveRoot(config.Database, result.Root.FixedBytes()); err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed to save EVM root to database %v", err)
		return nil
	}
	return ret
}
