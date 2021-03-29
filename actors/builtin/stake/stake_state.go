package stake

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cid "github.com/ipfs/go-cid"
)

var InflationDividend = big.NewInt(10000)

type State struct {
	RootKey         addr.Address
	TotalStakePower abi.StakePower

	MaturePeriod          abi.ChainEpoch
	RoundPeriod           abi.ChainEpoch
	PrincipalLockDuration abi.ChainEpoch
	MinDepositAmount      abi.TokenAmount
	MaxRewardPerRound     abi.TokenAmount
	InflationFactor       big.Int
	LastRoundReward       abi.TokenAmount
	NextRoundEpoch        abi.ChainEpoch

	LockedPrincipalMap    cid.Cid // Map, (HAMT[address]LockedPrincipalsCid)
	AvailablePrincipalMap cid.Cid // Map, (HAMT[address]TokenAmount)
	StakePowerMap         cid.Cid // Map, (HAMT[address]StakePower)
	VestingRewardMap      cid.Cid // Map, (HAMT[address]VestingFundsCid)
	AvailableRewardMap    cid.Cid // Map, (HAMT[address]TokenAmount)
}

func ConstructState(params *ConstructorParams, emptyMapCid cid.Cid) *State {
	return &State{
		RootKey:               params.RootKey,
		TotalStakePower:       big.Zero(),
		MaturePeriod:          params.MaturePeriod,
		RoundPeriod:           params.RoundPeriod,
		PrincipalLockDuration: params.PrincipalLockDuration,
		MinDepositAmount:      params.MinDepositAmount,
		MaxRewardPerRound:     params.MaxRewardPerRound,
		InflationFactor:       params.InflationFactor,
		LastRoundReward:       abi.NewTokenAmount(0),
		NextRoundEpoch:        params.FirstRoundEpoch,
		LockedPrincipalMap:    emptyMapCid,
		AvailablePrincipalMap: emptyMapCid,
		StakePowerMap:         emptyMapCid,
		VestingRewardMap:      emptyMapCid,
		AvailableRewardMap:    emptyMapCid,
	}
}

func (st *State) LoadLockedPrincipals(store adt.Store, lockedPrincipalMap *adt.Map, staker addr.Address) (*LockedPrincipals, bool, error) {
	var lockedPrincipalsCid cid.Cid
	var lockedPrincipalsCborCid cbg.CborCid
	found, err := lockedPrincipalMap.Get(abi.AddrKey(staker), &lockedPrincipalsCborCid)
	if err != nil {
		return nil, found, xerrors.Errorf("failed to get locked principalsCid for %v: %w", staker, err)
	}
	if !found {
		return nil, found, nil
	}
	lockedPrincipalsCid = cid.Cid(lockedPrincipalsCborCid)
	var lockedPrincipals LockedPrincipals
	if err = store.Get(store.Context(), lockedPrincipalsCid, &lockedPrincipals); err != nil {
		return nil, found, xerrors.Errorf("failed to load locked principals for %v: %w", staker, err)
	}
	return &lockedPrincipals, found, nil
}

func (st *State) putLockedPrincipals(store adt.Store, lockedPrincipalMap *adt.Map, staker addr.Address, lockedPrincipals *LockedPrincipals) error {
	lockedPrincipalsCid, err := store.Put(store.Context(), lockedPrincipals)
	if err != nil {
		return xerrors.Errorf("failed to save locked principals for %v: %w", staker, err)
	}
	lockedPrincipalsCborCid := cbg.CborCid(lockedPrincipalsCid)
	if err = lockedPrincipalMap.Put(abi.AddrKey(staker), &lockedPrincipalsCborCid); err != nil {
		return xerrors.Errorf("failed to put locked principals for %v: %w", staker, err)
	}
	return nil
}

func (st *State) updateAvailablePrincipal(availablePrincipalMap *adt.Map, staker addr.Address, amountDelta abi.TokenAmount) (abi.TokenAmount, error) {
	var amount abi.TokenAmount
	found, err := availablePrincipalMap.Get(abi.AddrKey(staker), &amount)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to get available principal for %v: %w", staker, err)
	}
	if !found {
		amount = big.Zero()
	}
	if amountDelta.IsZero() {
		return amount, nil
	}
	newAmount := big.Add(amount, amountDelta)
	if newAmount.LessThan(big.Zero()) {
		return amount, xerrors.Errorf("available principal cannot be negative %s", newAmount)
	}
	if err = availablePrincipalMap.Put(abi.AddrKey(staker), &newAmount); err != nil {
		return amount, xerrors.Errorf("failed to put available principal: %w", err)
	}
	return newAmount, nil
}

func (st *State) updateAvailableReward(availableRewardMap *adt.Map, staker addr.Address, amountDelta abi.TokenAmount) error {
	var amount abi.TokenAmount
	found, err := availableRewardMap.Get(abi.AddrKey(staker), &amount)
	if err != nil {
		return xerrors.Errorf("failed to get available reward for %v: %w", staker, err)
	}
	if !found {
		amount = big.Zero()
	}
	newAmount := big.Add(amount, amountDelta)
	if newAmount.LessThan(big.Zero()) {
		return xerrors.Errorf("available reward cannot be negative %s", newAmount)
	}
	if err = availableRewardMap.Put(abi.AddrKey(staker), &newAmount); err != nil {
		return xerrors.Errorf("failed to put available reward: %w", err)
	}
	return nil
}

func (st *State) updateStakePower(stakePowerMap *adt.Map, staker addr.Address, powerDelta abi.StakePower) error {
	var power abi.StakePower
	found, err := stakePowerMap.Get(abi.AddrKey(staker), &power)
	if err != nil {
		return xerrors.Errorf("failed to get stake power for %v: %w", staker, err)
	}
	if !found {
		power = big.Zero()
	}
	newPower := big.Add(power, powerDelta)
	if newPower.LessThan(big.Zero()) {
		return xerrors.Errorf("stake power cannot be negative %s", newPower)
	}
	if newPower.IsZero() {
		if err = stakePowerMap.Delete(abi.AddrKey(staker)); err != nil {
			return xerrors.Errorf("failed to remove staker: %w", err)
		}
	} else {
		if err = stakePowerMap.Put(abi.AddrKey(staker), &newPower); err != nil {
			return xerrors.Errorf("failed to put stake power: %w", err)
		}
	}
	return nil
}

func (st *State) LoadVestingFunds(store adt.Store, vestingRewardMap *adt.Map, staker addr.Address) (*VestingFunds, bool, error) {
	var vestingFundsCid cid.Cid
	var vestingFundsCborCid cbg.CborCid
	found, err := vestingRewardMap.Get(abi.AddrKey(staker), &vestingFundsCborCid)
	if err != nil {
		return nil, found, xerrors.Errorf("failed to get vesting funds for %v: %w", staker, err)
	}
	if !found {
		return nil, found, nil
	}
	vestingFundsCid = cid.Cid(vestingFundsCborCid)
	var vestingFunds VestingFunds
	if err = store.Get(store.Context(), vestingFundsCid, &vestingFunds); err != nil {
		return nil, found, xerrors.Errorf("failed to load vesting rewards for %v: %w", staker, err)
	}
	return &vestingFunds, found, nil
}

func (st *State) putVestingFunds(store adt.Store, vestingRewardMap *adt.Map, staker addr.Address, vestingFunds *VestingFunds) error {
	vestingFundsCid, err := store.Put(store.Context(), vestingFunds)
	if err != nil {
		return xerrors.Errorf("failed to save vesting funds for %v: %w", staker, err)
	}
	vestingFundsCborCid := cbg.CborCid(vestingFundsCid)
	if err = vestingRewardMap.Put(abi.AddrKey(staker), &vestingFundsCborCid); err != nil {
		return xerrors.Errorf("failed to put vesting rewards for %v: %w", staker, err)
	}
	return nil
}

func init() {
	// Check that ChainEpoch is indeed a signed integer to confirm that epochKey is making the right interpretation.
	var e abi.ChainEpoch
	if reflect.TypeOf(e).Kind() != reflect.Int64 {
		panic("incorrect chain epoch encoding")
	}

}
