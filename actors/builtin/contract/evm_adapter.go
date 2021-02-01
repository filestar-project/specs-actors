package contract

import (
	"math/big"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/types"

	"github.com/filecoin-project/go-address"
	stateBig "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
)

var log = logging.Logger("evm-adapter")

// evmAdapter implements interface evm.ChainAdapter
// providing access to FileStar chain for EVM contracts
type evmAdapter struct {
	runtime.Runtime
}

func newEvmAdapter(rt runtime.Runtime) *evmAdapter {
	r := &evmAdapter{}
	r.Runtime = rt
	return r
}

// Blockchain access
//// Get block hash by block number
func (e *evmAdapter) GetBlockHashByNum(num uint64) types.Hash {
	log.Debugf("evm-adapter::GetBlockHashByNum(%v)", num)
	return types.Hash{}
}

// Address call suicide
func (e *evmAdapter) DeleteAddress(addr types.Address) {
	a, err := e.tryGetActorAddress(addr)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot GetBalance(%x), error = %v", addr.Bytes(), err)
	}
	e.Runtime.DeleteContractActor(a)
}

// Balance managing

//// Get balance by address
func (e *evmAdapter) GetBalance(addr types.Address) *big.Int {
	a, err := e.tryGetActorAddress(addr)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot GetBalance(%x), error = %v", addr.Bytes(), err)
	}
	balance := e.Runtime.GetActorBalance(a)
	log.Debugf("evm-adapter::GetBalance(%x) => %v", addr.Bytes(), balance)
	return balance.Int
}

//// Transfer tokens
func (e *evmAdapter) TransferTokens(from, to types.Address, value *big.Int) {
	log.Debugf("evm-adapter::TransferTokens(%x, %x, %v)", from.FixedBytes(), to.FixedBytes(), value)
	msgValue := stateBig.NewFromGo(value)
	senderAddress, err := e.tryGetActorAddress(from)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot TransferTokens(%x, %x, %v), error = %v", from.FixedBytes(), to.FixedBytes(), value, value, err)
	}
	recipientAddress, err := e.tryGetActorAddress(to)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot TransferTokens(%x, %x, %v), error = %v", from.FixedBytes(), to.FixedBytes(), value, value, err)
	}
	e.Runtime.TransferTokens(senderAddress, recipientAddress, msgValue)
}

// try to get actor address by payload
// first try address.SECP256K1 protocol
// then try address.Actor protocol
// finally return error
func (e *evmAdapter) tryGetActorAddress(addr types.Address) (address.Address, error) {
	secpAddress, err := convertAddress(addr, address.SECP256K1)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrIllegalState, "cannot convert address from payload = %x to address.Address", addr.FixedBytes())
	}
	_, isSecp := e.Runtime.ResolveAddress(secpAddress)
	if isSecp {
		return secpAddress, nil
	}

	actorAddress, err := convertAddress(addr, address.Actor)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrIllegalState, "cannot convert address from payload = %x to address.Address", addr.FixedBytes())
	}
	_, isActor := e.Runtime.ResolveAddress(actorAddress)
	if isActor {
		return actorAddress, nil
	}

	return address.Address{}, xerrors.Errorf("address = %x not found", addr.FixedBytes())
}

func convertAddress(addr types.Address, protocol byte) (address.Address, error) {
	addrWithPrefix := append([]byte{protocol}, addr.Bytes()...)
	newAddress, err := address.NewFromBytes(addrWithPrefix)
	if err != nil {
		return address.Address{}, err
	}
	return newAddress, nil
}

// PrecomputeContractAddress - precompute contract address, based on caller address and contract code
// it will return new address and any errors
func PrecomputeContractAddress(caller address.Address, code []byte, salt []byte) (address.Address, error) {
	callerAddress := types.BytesToAddress(caller.Payload())
	precomputedAddress, err := evm.ComputeNewContractAddress(callerAddress, code, salt)
	if err != nil {
		return address.Address{}, err
	}
	newAddress, err := convertAddress(precomputedAddress, address.Actor)
	if err != nil {
		return address.Address{}, err
	}
	return newAddress, nil
}
