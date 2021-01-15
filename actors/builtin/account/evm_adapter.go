package account

import (
	"encoding/hex"
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
