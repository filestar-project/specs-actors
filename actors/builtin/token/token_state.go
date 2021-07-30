package token

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

type State struct {
	Nonce			big.Int
	URIs			cid.Cid    // array, AMT[TokenID]string
	Creators 		cid.Cid    // array, AMT[TokenID]addr.address
	Balances 		cid.Cid    // array, AMT[TokenID]TokenAmountInAddressCid
	Approves		cid.Cid    // Map, HAMT[address]ApproveTargetAddressCid
}

type TokenURI struct {
	TokenURI string
}

type AddrTokenAmountMap struct {
	AddrTokenAmountMap 	cid.Cid 	// Map, HAMT[address]tokenAmount
}

type AddrApproveMap struct {
	AddrApproveMap 		cid.Cid   	// Map, HAMT[Address]index
}

func ConstructState(emptyArrayCid cid.Cid, emptyMapCid cid.Cid) *State {
	return &State {
		Nonce: big.Zero(),
		URIs: emptyArrayCid,
		Creators: emptyArrayCid,
		Balances: emptyArrayCid,
		Approves: emptyMapCid,
	}
}

func (s *State) GetCreatorAddress(creatorsArray *adt.Array, tokenID big.Int) (addr.Address, bool, error) {
	var creatorAddress addr.Address
	found, err := creatorsArray.Get(tokenID.Uint64(), &creatorAddress)
	if err != nil {
		return addr.Address{}, found, xerrors.Errorf("failed to get addrTokenAmountMap for tokenID: %v, err: %w", tokenID, err)
	}
	if !found {
		return addr.Address{}, found, nil
	}
	return creatorAddress, found, nil
}

func (s *State) setCreatorAddress(creatorsArray *adt.Array, creator addr.Address, tokenID big.Int) error {

	if err := creatorsArray.Set(tokenID.Uint64(), &creator); err != nil {
		return xerrors.Errorf("failed to put new creator for new Token: %w", err)
	}

	return nil
}

func (s *State) LoadTokenURI(urisArray *adt.Array, tokenID big.Int) (*TokenURI, bool, error) {
	var tokenURI TokenURI
	found, err := urisArray.Get(tokenID.Uint64(), &tokenURI)
	if err != nil {
		return &TokenURI{}, found, xerrors.Errorf("failed to get URI for tokenID: %v, err: %w", tokenID, err)
	}
	if !found {
		return &TokenURI{}, found, nil
	}
	return &tokenURI, found, nil
}

func (s *State) setTokenURI(urisArray *adt.Array, tokenUri string, tokenID big.Int) error {
	var tokenURI = TokenURI{TokenURI: tokenUri}
	if err := urisArray.Set(tokenID.Uint64(), &tokenURI); err != nil {
		return xerrors.Errorf("failed to put new URI for tokenID: v, err : %w", tokenID, err)
	}

	return nil
}

func (s *State) LoadAddrTokenAmountMap(store adt.Store, balanceArray *adt.Array, tokenID big.Int) (*AddrTokenAmountMap, bool, error) {

	var addrTokenAmountMapCborCid cbg.CborCid
	found, err := balanceArray.Get(tokenID.Uint64(), &addrTokenAmountMapCborCid)
	if err != nil {
		return nil, found, xerrors.Errorf("failed to get addrTokenAmountMap for tokenID: %v, err: %w", tokenID, err)
	}
	if !found {
		return nil, found, nil
	}
	addrTokenAmountMapCid := cid.Cid(addrTokenAmountMapCborCid)
	var addrTokenAmountMap AddrTokenAmountMap
	if err = store.Get(store.Context(), addrTokenAmountMapCid, &addrTokenAmountMap); err != nil {
		return nil, found, xerrors.Errorf("failed to load addrTokenAmountMap for tokenID: %v, err: %w", tokenID, err)
	}
	return &addrTokenAmountMap, found, nil
}

func (s *State) putAddrTokenAmountMap(store adt.Store, balanceArray *adt.Array, tokenID big.Int, addrTokenAmountMap *AddrTokenAmountMap) error {
	balanceArrayCid, err := store.Put(store.Context(), addrTokenAmountMap)
	if err != nil {
		return xerrors.Errorf("failed to save addrTokenAmountMap for tokenID: %v, err: %w", tokenID, err)
	}
	balanceArrayCborCid := cbg.CborCid(balanceArrayCid)
	if err = balanceArray.Set(tokenID.Uint64(), &balanceArrayCborCid); err != nil {
		return xerrors.Errorf("failed to put addrTokenAmountMap for tokenID: %v, err: %w", tokenID, err)
	}

	return nil
}

func (s *State) LoadAddrTokenAmount(balanceMap *adt.Map, tokenOperator addr.Address) (abi.TokenAmount, bool, error) {
	var amount abi.TokenAmount
	found, err := balanceMap.Get(abi.AddrKey(tokenOperator), &amount)
	if err != nil {
		return big.Zero(), found, xerrors.Errorf("failed to get available reward for %v: %w", tokenOperator, err)
	}
	if !found {
		amount = big.Zero()
	}

	return amount, found, err
}

func (s *State) putAddrTokenAmount(balanceMap *adt.Map, tokenOperator addr.Address, amount abi.TokenAmount) error {
	if err := balanceMap.Put(abi.AddrKey(tokenOperator), &amount); err != nil {
		return xerrors.Errorf("failed to put AddrTokenAmount: %w", err)
	}
	return nil
}

func (s *State) LoadAddrApproveMap(store adt.Store, isAllApproveMap *adt.Map, tokenOperator addr.Address) (*AddrApproveMap, bool, error) {

	var addrApproveMapCborCid cbg.CborCid
	found, err := isAllApproveMap.Get(abi.AddrKey(tokenOperator), &addrApproveMapCborCid)
	if err != nil {
		return nil, found, xerrors.Errorf("failed to get addrApproveMap for address: %v, err: %w", tokenOperator, err)
	}
	if !found {
		return nil, found, nil
	}
	addrApproveMapCid := cid.Cid(addrApproveMapCborCid)
	var addrApproveMap AddrApproveMap
	if err = store.Get(store.Context(), addrApproveMapCid, &addrApproveMap); err != nil {
		return nil, found, xerrors.Errorf("failed to load addrTokenMap for address: %v, err: %w", tokenOperator, err)
	}
	return &addrApproveMap, found, nil
}

func (s *State) putAddrApproveMap(store adt.Store, isAllApproveMap *adt.Map, tokenOperator addr.Address, addrApproveMap *AddrApproveMap) error {
	isAllapproveMapCid, err := store.Put(store.Context(), addrApproveMap)
	if err != nil {
		return xerrors.Errorf("failed to save addrApproveMap for address: %v, err: %w", tokenOperator, err)
	}
	isAllapproveMapCborCid := cbg.CborCid(isAllapproveMapCid)
	if err = isAllApproveMap.Put(abi.AddrKey(tokenOperator), &isAllapproveMapCborCid); err != nil {
		return xerrors.Errorf("failed to put addrApproveMap for address: %v, err: %w", tokenOperator, err)
	}

	return nil
}

func (s *State) LoadAddrApprove(approveMap *adt.Map, tokenOperator addr.Address) (bool, bool, error) {
	var isApprove big.Int
	found, err := approveMap.Get(abi.AddrKey(tokenOperator), &isApprove)
	if err != nil {
		return false, found, xerrors.Errorf("failed to get available reward for %v: %w", tokenOperator, err)
	}
	if !found {
		isApprove = big.Zero()
	}

	return isApprove != big.Zero(), found, err
}

func (s *State) putAddrApprove(approveMap *adt.Map, tokenOperator addr.Address, isApprove bool) error {
	var isApproved = big.Zero()
	if isApprove {
		isApproved = big.NewInt(1)
	}
	if err := approveMap.Put(abi.AddrKey(tokenOperator), &isApproved); err != nil {
		return xerrors.Errorf("failed to put AddrTokenAmount: %w", err)
	}
	return nil
}
