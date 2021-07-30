package stake_test

import (
	"context"
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/stake"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/filecoin-project/specs-actors/v3/support/mock"
	tutil "github.com/filecoin-project/specs-actors/v3/support/testing"
)

func init() {

}

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, stake.Actor{})
}

func TestConstructor(t *testing.T) {
	actor := stakeHarness{stake.Actor{}, t}
	admin := tutil.NewIDAddr(t, 100)

	t.Run("construct", func(t *testing.T) {
		rt := mock.NewBuilder(context.Background(), builtin.StakeActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)
		params := stake.ConstructorParams{
			RootKey:               admin,
			MaturePeriod:          abi.ChainEpoch(10),
			RoundPeriod:           abi.ChainEpoch(20),
			PrincipalLockDuration: abi.ChainEpoch(30),
			FirstRoundEpoch:       abi.ChainEpoch(3),
			MinDepositAmount:      abi.NewTokenAmount(100_000_000),
			MaxRewardPerRound:     abi.NewTokenAmount(100_000_000_000),
			InflationFactor:       big.NewInt(100),
		}
		actor.constructAndVerify(rt, &params)
		st := getState(rt)

		assert.Equal(t, admin, st.RootKey)
		assert.Equal(t, big.Zero(), st.TotalStakePower)
		assert.Equal(t, abi.ChainEpoch(10), st.MaturePeriod)
		assert.Equal(t, abi.ChainEpoch(20), st.RoundPeriod)
		assert.Equal(t, abi.ChainEpoch(30), st.PrincipalLockDuration)
		assert.Equal(t, abi.ChainEpoch(3), st.NextRoundEpoch)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), st.MinDepositAmount)
		assert.Equal(t, abi.NewTokenAmount(100_000_000_000), st.MaxRewardPerRound)
		assert.Equal(t, big.NewInt(100), st.InflationFactor)
		assert.Equal(t, big.Zero(), st.LastRoundReward)
	})
}

func TestStake(t *testing.T) {
	actor := stakeHarness{stake.Actor{}, t}
	admin := tutil.NewIDAddr(t, 100)
	staker1 := tutil.NewIDAddr(t, 101)
	staker2 := tutil.NewIDAddr(t, 102)

	t.Run("deposit-withdraw", func(t *testing.T) {
		rt := mock.NewBuilder(context.Background(), builtin.StakeActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			WithEpoch(abi.ChainEpoch(0)).
			Build(t)
		params := stake.ConstructorParams{
			RootKey:               admin,
			MaturePeriod:          abi.ChainEpoch(10),
			RoundPeriod:           abi.ChainEpoch(20),
			PrincipalLockDuration: abi.ChainEpoch(30),
			FirstRoundEpoch:       abi.ChainEpoch(3),
			MinDepositAmount:      abi.NewTokenAmount(100_000_000),
			MaxRewardPerRound:     abi.NewTokenAmount(100_000_000_000),
			InflationFactor:       big.NewInt(100),
		}
		actor.constructAndVerify(rt, &params)

		actor.onEpochTickEnd(rt, abi.ChainEpoch(1))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(2))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(3))

		st := getState(rt)
		assert.Equal(t, abi.ChainEpoch(3+20), st.NextRoundEpoch)
		assert.Equal(t, abi.NewStoragePower(0), st.TotalStakePower)

		actor.deposit(rt, abi.ChainEpoch(4), staker1, abi.NewTokenAmount(100_000_000))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(4))
		st = getState(rt)
		lockedPrincipalMap, err := adt.AsMap(rt.AdtStore(), st.LockedPrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		lockedPrincipals, found, err := st.LoadLockedPrincipals(rt.AdtStore(), lockedPrincipalMap, staker1)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(lockedPrincipals.Data))
		assert.Equal(t, abi.ChainEpoch(4), lockedPrincipals.Data[0].Epoch)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), lockedPrincipals.Data[0].Amount)
		assert.Equal(t, abi.NewStoragePower(0), st.TotalStakePower)

		actor.deposit(rt, abi.ChainEpoch(5), staker2, abi.NewTokenAmount(200_000_000))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(5))
		st = getState(rt)
		lockedPrincipalMap, err = adt.AsMap(rt.AdtStore(), st.LockedPrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		lockedPrincipals, found, err = st.LoadLockedPrincipals(rt.AdtStore(), lockedPrincipalMap, staker1)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(lockedPrincipals.Data))
		assert.Equal(t, abi.ChainEpoch(4), lockedPrincipals.Data[0].Epoch)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), lockedPrincipals.Data[0].Amount)
		assert.Equal(t, abi.NewStoragePower(0), st.TotalStakePower)
		lockedPrincipals, found, err = st.LoadLockedPrincipals(rt.AdtStore(), lockedPrincipalMap, staker2)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(lockedPrincipals.Data))
		assert.Equal(t, abi.ChainEpoch(5), lockedPrincipals.Data[0].Epoch)
		assert.Equal(t, abi.NewTokenAmount(200_000_000), lockedPrincipals.Data[0].Amount)
		assert.Equal(t, abi.NewStoragePower(0), st.TotalStakePower)

		for epoch := 5; epoch <= 15; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st = getState(rt)
		assert.Equal(t, abi.NewStoragePower(100_000_000), st.TotalStakePower)

		actor.onEpochTickEnd(rt, abi.ChainEpoch(16))
		st = getState(rt)
		assert.Equal(t, abi.NewStoragePower(300_000_000), st.TotalStakePower)

		for epoch := 17; epoch <= 22; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st = getState(rt)
		assert.Equal(t, abi.NewTokenAmount(0), st.LastRoundReward)

		actor.onEpochTickEnd(rt, abi.ChainEpoch(23))
		st = getState(rt)
		assert.Equal(t, abi.NewTokenAmount(3_000_000), st.LastRoundReward)

		availablePrincipalMap, err := adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		var ap1 abi.TokenAmount
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker1), &ap1)
		assert.Nil(t, err)
		assert.False(t, found)

		for epoch := 24; epoch <= 35; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st = getState(rt)
		assert.Equal(t, abi.NewStoragePower(300_000_000), st.TotalStakePower)
		availablePrincipalMap, err = adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker1), &ap1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), ap1)
		var ap2 abi.TokenAmount
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker2), &ap2)
		assert.Nil(t, err)
		assert.False(t, found)

		assert.Equal(t, abi.NewStakePower(300_000_000), st.TotalStakePower)
		vestingRewardMap, err := adt.AsMap(rt.AdtStore(), st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		vfsr1, found, err := st.LoadVestingFunds(rt.AdtStore(), vestingRewardMap, staker1)
		assert.Nil(t, err)
		assert.True(t, found)
		total1 := abi.NewTokenAmount(0)
		for _, vf := range vfsr1.Funds {
			total1 = big.Add(total1, vf.Amount)
		}
		assert.Equal(t, abi.NewTokenAmount(1_000_000), total1)
		assert.Equal(t, 180, len(vfsr1.Funds))

		actor.onEpochTickEnd(rt, abi.ChainEpoch(36))
		st = getState(rt)
		availablePrincipalMap, err = adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker2), &ap2)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(200_000_000), ap2)

		actor.withdrawPrincipal(rt, abi.ChainEpoch(36), staker1, abi.NewTokenAmount(30_000_000))
		st = getState(rt)
		availablePrincipalMap, err = adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker1), &ap1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(70_000_000), ap1)

		rt.ExpectAbortContainsMessage(exitcode.ErrIllegalState, "failed to update available principals: available principal cannot be negative -100000000", func() {
			actor.withdrawPrincipal(rt, abi.ChainEpoch(36), staker2, abi.NewTokenAmount(300_000_000))
		})
	})

	t.Run("vesting", func(t *testing.T) {
		rt := mock.NewBuilder(context.Background(), builtin.StakeActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			WithEpoch(abi.ChainEpoch(0)).
			Build(t)
		params := stake.ConstructorParams{
			RootKey:               admin,
			MaturePeriod:          abi.ChainEpoch(10),
			RoundPeriod:           abi.ChainEpoch(2),
			PrincipalLockDuration: abi.ChainEpoch(30),
			FirstRoundEpoch:       abi.ChainEpoch(3),
			MinDepositAmount:      abi.NewTokenAmount(100_000_000),
			MaxRewardPerRound:     abi.NewTokenAmount(100_000_000_000),
			InflationFactor:       big.NewInt(100),
		}
		actor.constructAndVerify(rt, &params)
		actor.onEpochTickEnd(rt, abi.ChainEpoch(1))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(2))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(3))

		actor.deposit(rt, abi.ChainEpoch(4), staker1, abi.NewTokenAmount(100_000_000))
		for epoch := 4; epoch <= 9999; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st := getState(rt)
		vestingRewardMap, err := adt.AsMap(rt.AdtStore(), st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		vfsr1, found, err := st.LoadVestingFunds(rt.AdtStore(), vestingRewardMap, staker1)
		assert.Nil(t, err)
		assert.True(t, found)
		total1 := abi.NewTokenAmount(0)
		for _, vf := range vfsr1.Funds {
			total1 = big.Add(total1, vf.Amount)
		}
		assert.Equal(t, 361, len(vfsr1.Funds))
		assert.Equal(t, abi.NewTokenAmount(4965076367), total1)
	})

	t.Run("re-deposit", func(t *testing.T) {
		rt := mock.NewBuilder(context.Background(), builtin.StakeActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			WithEpoch(abi.ChainEpoch(0)).
			Build(t)
		params := stake.ConstructorParams{
			RootKey:               admin,
			MaturePeriod:          abi.ChainEpoch(10),
			RoundPeriod:           abi.ChainEpoch(20),
			PrincipalLockDuration: abi.ChainEpoch(30),
			FirstRoundEpoch:       abi.ChainEpoch(3),
			MinDepositAmount:      abi.NewTokenAmount(100_000_000),
			MaxRewardPerRound:     abi.NewTokenAmount(100_000_000_000),
			InflationFactor:       big.NewInt(100),
		}
		actor.constructAndVerify(rt, &params)
		actor.onEpochTickEnd(rt, abi.ChainEpoch(1))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(2))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(3))

		actor.deposit(rt, abi.ChainEpoch(4), staker1, abi.NewTokenAmount(100_000_000))
		for epoch := 4; epoch <= 35; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st := getState(rt)
		vestingRewardMap, err := adt.AsMap(rt.AdtStore(), st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		vfsr1, found, err := st.LoadVestingFunds(rt.AdtStore(), vestingRewardMap, staker1)
		assert.Nil(t, err)
		assert.True(t, found)
		total1 := abi.NewTokenAmount(0)
		for _, vf := range vfsr1.Funds {
			total1 = big.Add(total1, vf.Amount)
		}
		assert.Equal(t, abi.NewTokenAmount(1_000_000), total1)
		availablePrincipalMap, err := adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		var ap1 abi.TokenAmount
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker1), &ap1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), ap1)
		var sp1 abi.StakePower
		stakePowerMap, err := adt.AsMap(rt.AdtStore(), st.StakePowerMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = stakePowerMap.Get(abi.AddrKey(staker1), &sp1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), sp1)

		actor.withdrawPrincipal(rt, abi.ChainEpoch(36), staker1, abi.NewTokenAmount(100_000_000))
		actor.onEpochTickEnd(rt, abi.ChainEpoch(36))
		st = getState(rt)
		availablePrincipalMap, err = adt.AsMap(rt.AdtStore(), st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availablePrincipalMap.Get(abi.AddrKey(staker1), &ap1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(0), ap1)

		stakePowerMap, err = adt.AsMap(rt.AdtStore(), st.StakePowerMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = stakePowerMap.Get(abi.AddrKey(staker1), &sp1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(0), sp1)

		var ar1 abi.TokenAmount
		availableRewardMap, err := adt.AsMap(rt.AdtStore(), st.AvailableRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availableRewardMap.Get(abi.AddrKey(staker1), &ar1)
		assert.Nil(t, err)
		assert.False(t, found)

		// 180 days later
		for epoch := 37; epoch <= 522723; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st = getState(rt)
		availableRewardMap, err = adt.AsMap(rt.AdtStore(), st.AvailableRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = availableRewardMap.Get(abi.AddrKey(staker1), &ar1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(1_000_000), ar1)

		vestingRewardMap, err = adt.AsMap(rt.AdtStore(), st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		vfsr1, found, err = st.LoadVestingFunds(rt.AdtStore(), vestingRewardMap, staker1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, len(vfsr1.Funds))

		actor.deposit(rt, abi.ChainEpoch(522724), staker1, abi.NewTokenAmount(100_000_000))
		for epoch := 522724; epoch <= 522724+30; epoch += 1 {
			actor.onEpochTickEnd(rt, abi.ChainEpoch(epoch))
		}
		st = getState(rt)
		vestingRewardMap, err = adt.AsMap(rt.AdtStore(), st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		vfsr1, found, err = st.LoadVestingFunds(rt.AdtStore(), vestingRewardMap, staker1)
		assert.Nil(t, err)
		assert.True(t, found)
		total1 = abi.NewTokenAmount(0)
		for _, vf := range vfsr1.Funds {
			total1 = big.Add(total1, vf.Amount)
		}
		assert.Equal(t, abi.NewTokenAmount(1_000_000), total1)

		stakePowerMap, err = adt.AsMap(rt.AdtStore(), st.StakePowerMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		found, err = stakePowerMap.Get(abi.AddrKey(staker1), &sp1)
		assert.Nil(t, err)
		assert.True(t, found)
		assert.Equal(t, abi.NewTokenAmount(100_000_000), sp1)
	})

}

type stakeHarness struct {
	stake.Actor
	t testing.TB
}

func (h *stakeHarness) constructAndVerify(rt *mock.Runtime, params *stake.ConstructorParams) {
	rt.Reset()
	rt.ExpectValidateCallerAddr(builtin.SystemActorAddr)
	ret := rt.Call(h.Constructor, params)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *stakeHarness) onEpochTickEnd(rt *mock.Runtime, currEpoch abi.ChainEpoch) {
	rt.Reset()
	rt.SetCaller(builtin.CronActorAddr, builtin.CronActorCodeID)
	rt.ExpectValidateCallerAddr(builtin.CronActorAddr)
	rt.SetEpoch(currEpoch)
	rt.SetReceived(abi.NewTokenAmount(0))
	rt.SetBalance(abi.NewTokenAmount(0))
	rt.Call(h.Actor.OnEpochTickEnd, nil)
	rt.Verify()
}

func (h *stakeHarness) deposit(rt *mock.Runtime, currEpoch abi.ChainEpoch, staker addr.Address, amount abi.TokenAmount) {
	rt.SetCaller(staker, builtin.AccountActorCodeID)
	rt.ExpectValidateCallerAny()
	rt.SetEpoch(currEpoch)
	rt.SetReceived(amount)
	rt.Call(h.Actor.Deposit, nil)
	rt.Verify()
}

func (h *stakeHarness) withdrawPrincipal(rt *mock.Runtime, currEpoch abi.ChainEpoch, staker addr.Address, amount abi.TokenAmount) {
	rt.SetCaller(staker, builtin.AccountActorCodeID)
	rt.ExpectValidateCallerAny()
	rt.ExpectSend(staker, 0, nil, amount, nil, exitcode.Ok)
	rt.SetEpoch(currEpoch)
	rt.SetReceived(abi.NewTokenAmount(0))
	rt.SetBalance(amount)
	rt.Call(h.Actor.WithdrawPrincipal, &stake.WithdrawParams{AmountRequested: amount})
	rt.Verify()
}

func getState(rt *mock.Runtime) *stake.State {
	var st stake.State
	rt.GetState(&st)
	return &st
}
