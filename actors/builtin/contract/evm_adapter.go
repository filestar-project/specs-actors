package contract

import (
	"bytes"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	peer "github.com/libp2p/go-libp2p-peer"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	stateBig "github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"

	init0 "github.com/filecoin-project/specs-actors/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
	"github.com/filestar-project/evm-adapter/evm"
	"github.com/filestar-project/evm-adapter/evm/types"
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
	log.Debugf("evm-adapter::DeleteAddress(%x)", addr.Bytes())
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

func convertAddressTypes(addr address.Address) types.Address {
	newAddress := &types.Address{}
	newAddress.SetBytes(addr.Bytes()[1:])
	return *newAddress
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

func (e *evmAdapter) CallAddress(addr types.Address, method uint256.Int, params []byte) ([]byte, error) {
	log.Debugf("evm-adapter::CallAddress(%x, %x, %x)", addr.Bytes(), method.Bytes(), params)
	actorAddr, err := e.tryGetActorAddress(addr)
	if err != nil {
		return []byte{}, xerrors.Errorf("address = %x not found", addr.FixedBytes())
	}
	actorCode, ok := e.Runtime.GetActorCodeCID(actorAddr)
	if !ok {
		return []byte{}, xerrors.Errorf("can't get actor code, address = %x", addr.FixedBytes())
	}
	if actorCode == builtin.ContractActorCodeID {
		return []byte{}, xerrors.Errorf("call contract actor address = %x", addr.FixedBytes())
	}
	result, exitCode := e.Runtime.SendMarshalled(actorAddr, abi.MethodNum(method.Uint64()), abi.NewTokenAmount(0), params)
	if exitCode != 0 {
		return []byte{}, xerrors.Errorf("Unsuccessful call address %v", addr.FixedBytes())
	}
	return result, nil
}

func (e *evmAdapter) CreateContract(from types.Address, code []byte, salt []byte, amount *big.Int) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	log.Debugf("evm-adapter::CreateContract(%x, %x, %x, %x)", from.Bytes(), code, salt, amount.Bytes())
	value, err := stateBig.FromBytes(amount.Bytes())
	if err != nil {
		return []byte{}, from, 0, xerrors.Errorf("failed to convert bigInt to lotus stateBig: %w", err)
	}
	contractParams, err := actors.SerializeParams(&ContractParams{Code: code, Value: value, Salt: salt, Commit: true})
	if err != nil {
		return []byte{}, from, 0, xerrors.Errorf("failed to serialize contract create params: %w", err)
	}

	params, err := actors.SerializeParams(&init0.ExecParams{CodeCID: builtin.ContractActorCodeID, ConstructorParams: contractParams})
	if err != nil {
		return []byte{}, from, 0, xerrors.Errorf("failed to serialize exec contract create params: %w", err)
	}
	ret, errCode := e.Runtime.SendMarshalled(builtin.InitActorAddr, builtin.MethodsInit.ExecWithResult, abi.NewTokenAmount(int64(0)), params)
	if errCode != 0 {
		return []byte{}, from, 0, xerrors.Errorf("Unsuccessful send to initActor, error code = %v", errCode)
	}

	var result ContractResult
	if err := result.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		e.Runtime.Abortf(exitcode.ErrSerialization, "failed to unmarshal return value: %s", err)
	}
	return result.Value, types.Address(result.Address), uint64(result.GasUsed), nil
}

func (e *evmAdapter) GetNonce(addr types.Address) uint64 {
	a, err := e.tryGetActorAddress(addr)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot GetNonce(%x), error = %v", addr.Bytes(), err)
	}
	nonce := e.Runtime.GetNonce(a)
	log.Debugf("evm-adapter::GetNonce(%x) => %v", addr.Bytes(), nonce)
	return nonce
}

func (e *evmAdapter) SetNonce(addr types.Address, value uint64) {
	a, err := e.tryGetActorAddress(addr)
	if err != nil {
		e.Runtime.Abortf(exitcode.ErrForbidden, "cannot SetNonce(%x), error = %v", addr.Bytes(), err)
	}
	e.Runtime.SetNonce(a, value)
	log.Debugf("evm-adapter::SetNonce(%x, %v)", addr.Bytes(), value)
}

func (e *evmAdapter) ImportLocalStorage(path string, car bool) (multistore.StoreID, cid.Cid, error) {
	return e.Runtime.ImportLocalStorage(path, car)
}

func (e *evmAdapter) DropLocalStorage(ids []multistore.StoreID) error {
	return e.Runtime.DropLocalStorage(ids)
}

func (e *evmAdapter) ListLocalImports() ([]runtime.Import, error) {
	return e.Runtime.ListLocalImports()
}

func (e *evmAdapter) FindData(root cid.Cid, pieceCid *cid.Cid) ([]runtime.QueryOffer, bool, error) {
	return e.Runtime.FindData(root, pieceCid)
}

func (e *evmAdapter) RetrieveData(params runtime.RetrieveParams) error {
	return e.Runtime.RetrieveData(params)
}

func (e *evmAdapter) InitDeal(params runtime.InitDealParams) (*cid.Cid, error) {
	return e.Runtime.InitDeal(params)
}

func (e *evmAdapter) QueryAsk(maddr address.Address, pid peer.ID) (*storagemarket.StorageAsk, error) {
	return e.Runtime.QueryAsk(maddr, pid)
}

func (e *evmAdapter) ListDeals() ([]runtime.DealInfo, error) {
	return e.Runtime.ListDeals()
}

func (e *evmAdapter) GetDeal(propCid cid.Cid) (runtime.Deal, error) {
	return e.Runtime.GetDeal(propCid)
}

func (e *evmAdapter) ListAsks() ([]*storagemarket.StorageAsk, error) {
	return e.Runtime.ListAsks()
}
