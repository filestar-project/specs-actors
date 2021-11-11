package contract

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
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
		3: a.GetCode,
		4: a.GetStorageAt,
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
	Address address.Address
}

// PathToRepo path to repo from command params
var PathToRepo string

func setRandomPath() error {
	if len(PathToRepo) == 0 {
		// if no path provided, open LevelDB in tmp random dir
		randBytes := make([]byte, 16)
		if _, err := rand.Read(randBytes); err != nil {
			return err
		}
		PathToRepo = filepath.Join(os.TempDir(), hex.EncodeToString(randBytes))
		return os.MkdirAll(PathToRepo, os.ModePerm)
	}
	return nil
}

type EvmLogs struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address types.Address
	// list of topics provided by the contract.
	Topics []types.Hash
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
		result[i].Topics = log.Topics
	}
	return result
}

func ConvertEvmLogs(logs []EvmLogs) []types.Log {
	evmLogs := make([]types.Log, len(logs))
	for i, log := range logs {
		evmLogs[i] = types.Log{
			Address: log.Address,
			Topics:  log.Topics,
			Data:    log.Data,
			Removed: log.Removed,
		}
	}
	return evmLogs
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

type StateContractInfo struct {
	Root   types.Hash
	Height int
}

type stateDBManager struct {
	StateContractInfo
	LevelDB types.Database
	Statedb types.StateDB
}

var dbManager stateDBManager

func InitStateDBManager() error {
	dbManager.Statedb = nil
	err := dbManager.setLevelDB()
	if err != nil {
		return err
	}
	dbManager.Root, err = state.GetRoot(dbManager.LevelDB)
	return err
}

func InitStateDB(root types.Hash, adapter *evmAdapter, config *types.Config) error {
	statedb, err := state.New(root.FixedBytes(), dbManager.LevelDB, adapter, config)
	dbManager.Statedb = statedb
	return err
}

func (manager *stateDBManager) setLevelDB() error {
	err := setRandomPath()
	if err != nil {
		return err
	}
	db, err := state.OpenDatabase(PathToRepo)
	if err != nil {
		return err
	}
	manager.LevelDB = db
	return nil
}

func SaveContractInfo() error {
	if dbManager.Statedb != nil {
		root, err := dbManager.Statedb.Commit(false)
		dbManager.Root = types.ConvertHash(root)
		if err != nil {
			return err
		}
	}
	return state.SaveRoot(dbManager.LevelDB, dbManager.Root.FixedBytes())
}

func GetCurrentHeight() int {
	return dbManager.Height
}

func IsMaxHeight(height int) bool {
	return dbManager.Height < height
}

func UpdateCurrentHeight(height int) {
	if IsMaxHeight(height) {
		dbManager.Height = height
	}
}

func ReInitStateDBForHeight(height int64) error {
	rootManager, err := GetStateRootManager()
	if err != nil {
		return err
	}
	// Empty root, if root in the current height doesn't exist
	// Otherwise copy root to the manager
	root, err := rootManager.GetRoot(strconv.FormatInt(int64(height), 16))
	if err == nil {
		dbManager.Root = types.BytesToHash(root)
	}
	if dbManager.Statedb != nil {
		return dbManager.Statedb.Reset(common.BytesToHash(root))
	}
	return nil
}

type ContractParams struct {
	Code   []byte
	Value  lotusbig.Int
	Salt   []byte
	Commit bool
}

type ContractResult struct {
	Value   []byte
	GasUsed int64
	Address types.Address
	Logs    []EvmLogs
}

type StorageInfo struct {
	Address  address.Address
	Position string
	Root     []byte
}

type StorageResult struct {
	Value []byte
}

type GetCodeResult struct {
	Code string
}

// Creates new EVM configuration
func newEvmConfig(rt runtime.Runtime, root types.Hash, params *ContractParams) *types.Config {
	// fake for now
	caller := types.BytesToAddress(rt.Origin().Payload())
	return &types.Config{
		// This fields used by Solidity opcodes:
		BlockNumber:     big.NewInt(100),
		BlockTime:       big.NewInt(0),
		BlockDifficulty: big.NewInt(0),
		BlockGasLimit:   100000000000,
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
		// CommitStatus: params.Commit,
	}
}

func constructEvmObject(rt runtime.Runtime, params *ContractParams) *evm.EVM {
	root := dbManager.Root
	config := newEvmConfig(rt, root, params)

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	// Init only once, when daemon starts
	if dbManager.Statedb == nil {
		err := InitStateDB(config.RootHash, adapter, config)
		if err != nil {
			rt.Abortf(exitcode.ErrIllegalState, "Failed init of new StateDB object %v", err)
			return nil
		}
	}
	evmObj, err := evm.NewEVM(adapter, dbManager.Statedb, config)

	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new EVM object %v", err)
		return nil
	}
	return evmObj
}

func getCurrentStateRoot() types.Hash {
	return dbManager.Root
}

func (a Actor) Constructor(rt runtime.Runtime, params *ContractParams) *ContractResult {
	// Account actors are created implicitly by sending a message to a pubkey-style address.
	// This constructor is not invoked by the InitActor, but by the system.
	rt.ValidateImmediateCallerIs(builtin.InitActorAddr)
	// logs and call validation
	rt.Log(rtt.DEBUG, "contractActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	addr, err := PrecomputeContractAddress(rt.Origin(), params.Code, params.Salt)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Can't Precompute contract address!!!")
		return nil
	}
	st := State{Address: addr}
	rt.StateCreate(&st)
	switch rt.Origin().Protocol() {
	case address.SECP256K1:
	case address.Actor:
		break
	default:
		rt.Abortf(exitcode.ErrForbidden, "Only Secp256k1 or Actor addresses allowed in Caller! Current address protocol: %v", rt.Origin().Protocol())
	}
	evmObj := constructEvmObject(rt, params)
	snap := dbManager.Statedb.Snapshot()
	// instruct EVM to create the contract
	gasLimit := rt.GasLimit()
	result, err := evmObj.CreateContract(params.Code, uint64(gasLimit), params.Value.Int)
	if err != nil {
		dbManager.Statedb.RevertToSnapshot(snap)
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}
	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = int64(gasLimit - result.GasLeft)
	ret.Address = result.Address
	// charge gas counted by EVM for contract creation
	if !dbManager.Statedb.Exist(ret.Address.GetCommonAddress()) {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, addr = %v, got %v", ret.Address, err)
	}
	rt.ChargeGas("OnCreateContract", ret.GasUsed, 0)
	// Add logs from evm
	ret.Logs = newEvmLogs(evmObj.GetLogs())
	rt.Log(rtt.INFO, getFormatLogs(evmObj.GetLogs()))
	if params.Commit {
		appendLogs(ret.Logs)
		//Compute root and save changes for intermediate state in Runtime
		dbManager.Statedb.IntermediateRoot(false)
		return ret
	}
	dbManager.Statedb.RevertToSnapshot(snap)
	return ret
}

func (a Actor) CallContract(rt runtime.Runtime, params *ContractParams) *ContractResult {
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
	if rt.RecieverAddress().Protocol() != address.Actor {
		rt.Abortf(exitcode.ErrForbidden, "Only Actor addresses allowed in Reciever! Current address protocol: %v", rt.RecieverAddress().Protocol())
	}
	evmObj := constructEvmObject(rt, params)
	snap := dbManager.Statedb.Snapshot()
	// instruct EVM to call the contract
	gasLimit := rt.GasLimit()
	receiver := types.BytesToAddress(rt.RecieverAddress().Payload())
	result, err := evmObj.CallContract(receiver, params.Code, uint64(gasLimit), params.Value.Int)
	if err != nil {
		dbManager.Statedb.RevertToSnapshot(snap)
		rt.Abortf(exitcode.ErrForbidden, "Failed call contract, got %v", err)
		return nil
	}
	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	if ret.Value == nil {
		ret.Value = make([]byte, 0)
	}
	ret.GasUsed = int64(gasLimit - result.GasLeft)
	// charge gas counted by EVM for this call
	rt.ChargeGas("OnCallContract", ret.GasUsed, 0)
	// Add logs from evm
	ret.Logs = newEvmLogs(evmObj.GetLogs())
	rt.Log(rtt.INFO, getFormatLogs(evmObj.GetLogs()))
	if params.Commit {
		appendLogs(ret.Logs)
		//Compute root and save changes for intermediate state in Runtime
		dbManager.Statedb.IntermediateRoot(false)
		return ret
	}
	dbManager.Statedb.RevertToSnapshot(snap)
	return ret
}

func (a Actor) GetCode(rt runtime.Runtime, _ *abi.EmptyValue) *GetCodeResult {
	receiver := types.BytesToAddress(rt.RecieverAddress().Payload())
	rt.Log(rtt.DEBUG, "contractActor.GetCode, addr = %s", hex.EncodeToString(receiver.Bytes()))
	rt.ValidateImmediateCallerAcceptAny()
	config := newEvmConfig(rt, dbManager.Root, &ContractParams{})
	adapter := newEvmAdapter(rt)
	statedb, err := state.New(config.RootHash.FixedBytes(), dbManager.LevelDB, adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new StateDB object %v", err)
		return nil
	}
	return &GetCodeResult{Code: hex.EncodeToString(statedb.GetCode(receiver.GetCommonAddress()))}
}

func (a Actor) GetStorageAt(rt runtime.Runtime, params *StorageInfo) *StorageResult {
	receiver := types.BytesToAddress(rt.RecieverAddress().Payload())
	rt.Log(rtt.DEBUG, "contractActor.GetCode, addr = %s", hex.EncodeToString(receiver.Bytes()))
	rt.ValidateImmediateCallerAcceptAny()
	root := types.BytesToHash(params.Root)
	if len(params.Root) == 0 {
		root = dbManager.Root
	}
	rt.Log(rtt.DEBUG, "contractActor.GetCode, root = %s", hex.EncodeToString(root[:]))
	config := newEvmConfig(rt, root, &ContractParams{})
	adapter := newEvmAdapter(rt)
	statedb, err := state.New(config.RootHash.FixedBytes(), dbManager.LevelDB, adapter, config)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "Failed creation of new StateDB object %v", err)
		return nil
	}
	value := statedb.GetState(convertAddressTypes(params.Address).FixedBytes(), common.HexToHash(params.Position))
	return &StorageResult{Value: value[:]}
}
