package account

import (
	"encoding/hex"
	"math/big"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/types"

	rtt "github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
)

type ContractParams struct {
	Code []byte
	Salt []byte
}

type ContractResult struct {
	Value   []byte
	GasUsed int64
}

var _ = &evm.EVM{}

// Creates new EVM configuration
func newEvmConfig(rt runtime.Runtime, params *ContractParams) *evm.Config {
	// fake for now
	caller := types.BytesToAddress(rt.Origin().Payload())
	return &evm.Config{
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

		// Logs config for StateDB
		LogsCfg: evm.LogsConfig{
			BlockHash: types.BytesToHash([]byte{123}),
			TrxHash:   types.BytesToHash([]byte{121}),
			TrxIndex:  321,
		},
	}
}

// Creates new EVM contract
func (a Actor) CreateContract(rt runtime.Runtime, params *ContractParams) *ContractResult {

	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()

	// construct proxy object and EVM
	adapter := newEvmAdapter(rt)
	evm := evm.NewEVM(adapter, newEvmConfig(rt, params))

	// instruct EVM to create the contract
	gasLimit := rt.GasLimit()
	value := abi.NewTokenAmount(0)
	result, err := evm.CreateContract(params.Code, uint64(gasLimit), value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}

	// save contract storage etc
	adapter.ApplyCreate(result.Address)

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = gasLimit - int64(result.GasLeft)

	// charge gas counted by EVM for contract creation
	rt.ChargeGas("evm", ret.GasUsed, 0)

	return ret
}

// Call EVM contract
func (a Actor) CallContract(rt runtime.Runtime, params *ContractParams) *ContractResult {

	// logs and call validation
	rt.Log(rtt.DEBUG, "accountActor.CallContract, code = %s", hex.EncodeToString(params.Code))
	rt.ValidateImmediateCallerAcceptAny()

	// fetch EVM contract address from state
	var st State
	rt.StateReadonly(&st)

	// construct proxy object and EVM
	adapter := newEvmAdapterWithState(rt, &st)
	evm := evm.NewEVM(adapter, newEvmConfig(rt, params))

	// instruct EVM to call the contract
	gasLimit := rt.GasLimit()
	value := rt.ValueReceived()
	result, err := evm.CallContract(st.Contract, params.Code, uint64(gasLimit), value.Int)
	if err != nil {
		rt.Abortf(exitcode.ErrForbidden, "Failed create contract, got %v", err)
		return nil
	}

	adapter.ApplyCall()

	// construct result which is being returned
	ret := &ContractResult{}
	ret.Value = result.Value
	ret.GasUsed = gasLimit - int64(result.GasLeft)

	// charge gas counted by EVM for this call
	rt.ChargeGas("evm", ret.GasUsed, 0)

	return ret
}
