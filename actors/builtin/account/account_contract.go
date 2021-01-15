package account

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"

	"math/big"

	"github.com/filecoin-project/go-address"
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
	Code []byte
	Salt []byte
}

type ContractResult struct {
	Value   []byte
	GasUsed int64
}

// Creates new EVM configuration
func newEvmConfig(rt runtime.Runtime, params *ContractParams) *types.Config {
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
		BlockHash: types.BytesToHash([]byte{123}),
		TrxHash:   types.BytesToHash([]byte{121}),
		TrxIndex:  321,

		// Root hash of stateDB
		RootHash: root,
		// Database for EVM
		Database: db,
	}
}

// Creates new EVM contract
func (a Actor) CreateContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()
	if rt.OriginReciever().Protocol() != address.SECP256K1 {
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 addresses allowed! Current address protocol: %v", rt.OriginReciever().Protocol())
	}

	config := newEvmConfig(rt, params)

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	evm, err := evm.NewEVM(adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new EVM object %v", err)
		return nil
	}

	// instruct EVM to create the contract
	gasLimit := rt.GasLimit()
	value := rt.ValueReceived()
	result, err := evm.CreateContract(params.Code, uint64(gasLimit), value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = gasLimit - int64(result.GasLeft)

	// Save root to database
	if err := state.SaveRoot(config.Database, result.Root.FixedBytes()); err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed to save EVM root to database %v", err)
		return nil
	}

	// charge gas counted by EVM for contract creation
	rt.ChargeGas("evm", ret.GasUsed, 0)

	return ret
}

// Call EVM contract
func (a Actor) CallContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CallContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()
	if rt.OriginReciever().Protocol() != address.SECP256K1 {
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 addresses allowed! Current address protocol: %v", rt.OriginReciever().Protocol())
	}

	config := newEvmConfig(rt, params)

	// fetch EVM contract address from state
	var st State
	rt.StateReadonly(&st)

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	evm, err := evm.NewEVM(adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new EVM object %v", err)
		return nil
	}

	// instruct EVM to call the contract
	gasLimit := rt.GasLimit()
	value := rt.ValueReceived()
	receiver := types.BytesToAddress(rt.OriginReciever().Payload())
	result, err := evm.CallContract(receiver, params.Code, uint64(gasLimit), value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = gasLimit - int64(result.GasLeft)

	// Save root to database
	if err := state.SaveRoot(config.Database, result.Root.FixedBytes()); err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed to save EVM root to database %v", err)
		return nil
	}

	// charge gas counted by EVM for this call
	rt.ChargeGas("evm", ret.GasUsed, 0)

	return ret
}
