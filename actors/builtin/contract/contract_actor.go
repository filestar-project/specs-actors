package contract

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/filecoin-project/go-address"
	addr "github.com/filecoin-project/go-address"
	lotusbig "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/state"
	"github.com/filestar-project/evm-adapter/evm/types"
	"github.com/ipfs/go-cid"

	rtt "github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
)

type Actor struct{}

func (a Actor) Exports() []interface{} {
	return []interface{}{
		1: a.Constructor,
		2: a.CallContract,
		3: a.CallContractWithoutCommit,
	}
}

func (a Actor) Code() cid.Cid {
	return builtin.ContractActorCodeID
}

func (a Actor) State() cbor.Er {
	return new(State)
}

var _ runtime.VMActor = Actor{}

type State struct {
	Address addr.Address
}

// PathToRepo path to repo from command params
var PathToRepo string

type databaseSingleton struct {
	levelDB types.Database
}

var database *databaseSingleton
var once sync.Once

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

type ContractParams struct {
	Code         []byte
	Value        lotusbig.Int
	CommitStatus bool
}

type ContractResult struct {
	Value   []byte
	GasUsed int64
	Address address.Address
	Logs    []EvmLogs
}

// Creates new EVM configuration
func newEvmConfig(rt runtime.Runtime, params *ContractParams, salt []byte) *types.Config {
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
		Salt: salt,

		// This fields used by logs:
		BlockHash:    types.BytesToHash([]byte{123}),
		TrxHash:      types.BytesToHash([]byte{121}),
		TrxIndex:     321,
		CommitStatus: params.CommitStatus,
		// Root hash of stateDB
		RootHash: root,
		// Database for EVM
		Database: db,
	}
}

func (a Actor) Constructor(rt runtime.Runtime, params *ContractParams) *ContractResult {
	// Account actors are created implicitly by sending a message to a pubkey-style address.
	// This constructor is not invoked by the InitActor, but by the system.
	rt.ValidateImmediateCallerIs(builtin.InitActorAddr)
	// logs and call validation
	rt.Log(rtt.DEBUG, "contractActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	addr, salt, err := PrecomputeContractAddress(rt.Origin(), params.Code)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Cannot compute contract address, caller = %x, err = ", rt.Origin(), err)
	}
	rt.CreateActor(builtin.ContractActorCodeID, addr)
	st := State{Address: addr}
	rt.StateCreate(&st)
	switch rt.Origin().Protocol() {
	case address.SECP256K1:
	case address.Actor:
		break
	default:
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 or Actor addresses allowed in Caller! Current address protocol: %v", rt.Origin().Protocol())
	}

	config := newEvmConfig(rt, params, salt)

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
	params.CommitStatus = true
	return a.callContract(rt, params)
}

func (a Actor) CallContractWithoutCommit(rt runtime.Runtime, params *ContractParams) *ContractResult {
	params.CommitStatus = false
	return a.callContract(rt, params)
}

// Call EVM contract
func (a Actor) callContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
	// logs and call validation
	rt.Log(rtt.DEBUG, "contractActor.CallContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()
	switch rt.Origin().Protocol() {
	case address.SECP256K1:
	case address.Actor:
		break
	default:
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 or Actor addresses allowed in Caller! Current address protocol: %v", rt.Origin().Protocol())
	}
	if rt.OriginReciever().Protocol() != address.Actor {
		rt.Abortf(exitcode.ErrForbidden, "Only Actor addresses allowed in Reciever! Current address protocol: %v", rt.OriginReciever().Protocol())
	}

	config := newEvmConfig(rt, params, []byte{})

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
