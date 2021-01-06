package account

import (
	"encoding/hex"
	"fmt"
	"math/big"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
	"github.com/filestar-project/evm-adapter/crypto"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/types"

	"github.com/filecoin-project/go-address"
)

var log = logging.Logger("evm-adapter")

// evmAdapter implements interface evm.ChainAdapter
// providing access to FileStar chain for EVM contracts
type evmAdapter struct {
	runtime.Runtime
	storage map[string]StorageValue
}

var _ evm.ChainAdapter = &evmAdapter{}

func newEvmAdapter(rt runtime.Runtime) *evmAdapter {
	r := &evmAdapter{}
	r.Runtime = rt
	r.storage = make(map[string]StorageValue)
	return r
}

func newEvmAdapterWithState(rt runtime.Runtime, s *State) *evmAdapter {
	r := &evmAdapter{}
	r.Runtime = rt
	r.storage = s.Storage
	return r
}

// Blockchain access
//// Get block hash by block number
func (e *evmAdapter) GetBlockHashByNum(num uint64) types.Hash {
	log.Debugf("evm-adapter::GetBlockHashByNum(%v)", num)
	return types.Hash{}
}

// Storage manager
// Look to core/state/database.go for more info about Trie logic
func (e *evmAdapter) Get(key [types.HashLength]byte) (r []byte, err error) {
	ks := hex.EncodeToString(key[:])
	if v, ok := e.storage[ks]; ok {
		r = v.Value
	} else {
		err = fmt.Errorf("key not found")
	}
	log.Debugf("evm-adapter::Get(%v) => %v, %v", ks, hex.EncodeToString(r), err)
	return
}

func (e *evmAdapter) Put(key [types.HashLength]byte, value []byte) error {
	ks := hex.EncodeToString(key[:])
	e.storage[ks] = StorageValue{value}
	log.Debugf("evm-adapter::Put(%v, %v)", ks, hex.EncodeToString(value))
	return nil
}

//// Remove by key
func (e *evmAdapter) Remove(key [types.HashLength]byte) error {
	ks := hex.EncodeToString(key[:])
	delete(e.storage, ks)
	log.Debugf("evm-adapter::Remove(%v)", ks)
	return nil
}

// Balance managing

//// Get balance by address
func (e *evmAdapter) GetBalance(addr types.Address) *big.Int {
	result := big.NewInt(0)
	log.Debugf("evm-adapter::GetBalance(%v) => %v", hex.EncodeToString(addr.Bytes()), result)
	return result
}

//// Add balance by address
func (e *evmAdapter) AddBalance(addr types.Address, value *big.Int) {
	log.Debugf("evm-adapter::AddBalance(%v, %v)", hex.EncodeToString(addr.Bytes()), value)
}

//// Sub balance by address
func (e *evmAdapter) SubBalance(addr types.Address, value *big.Int) {
	log.Debugf("evm-adapter::SetBalance(%v, %v)", hex.EncodeToString(addr.Bytes()), value)
}

func (e *evmAdapter) apply(state *State) {
	state.Storage = e.storage
}

func (e *evmAdapter) ApplyCall() {
	var state State
	e.StateTransaction(&state, func() {
		e.apply(&state)
	})
}

func (e *evmAdapter) ApplyCreate(addr types.Address) {
	var state State
	e.StateTransaction(&state, func() {
		e.apply(&state)
		state.Contract = addr
	})
}

// PrecomputeContractAddress - precompute contract address, based on caller address and contract code
// it will return new address, salt (used for address generation) and any errors
func PrecomputeContractAddress(caller address.Address, code []byte) (address.Address, []byte, error) {
	salt := crypto.GenerateSalt()
	callerAddress := types.BytesToAddress(caller.Payload())
	precomputedAddress, err := evm.ComputeNewContractAddress(callerAddress, code, salt)
	if err != nil {
		return address.Address{}, salt, err
	}
	prAddrWithPrefix := append([]byte{address.SECP256K1}, precomputedAddress.Bytes()...)
	newAddress, err := address.NewFromBytes(prAddrWithPrefix)
	if err != nil {
		return address.Address{}, salt, err
	}
	return newAddress, salt, nil
}
