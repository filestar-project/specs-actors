package stake

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"
	stake2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/stake"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/runtime"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"
)

type Runtime = runtime.Runtime

type Actor struct{}

func (a Actor) Exports() []interface{} {
	return []interface{}{
		builtin.MethodConstructor: a.Constructor,
		2:                         a.Deposit,
		3:                         a.WithdrawPrincipal,
		4:                         a.WithdrawReward,
		5:                         a.ChangeMaturePeriod,
		6:                         a.ChangeRoundPeriod,
		7:                         a.ChangePrincipalLockDuration,
		8:                         a.ChangeMinDepositAmount,
		9:                         a.ChangeMaxRewardsPerRound,
		10:                        a.ChangeInflationFactor,
		11:                        a.ChangeRootKey,
		12:                        a.OnEpochTickEnd,
	}
}

func (a Actor) Code() cid.Cid {
	return builtin.StakeActorCodeID
}

func (a Actor) IsSingleton() bool {
	return true
}

func (a Actor) State() cbor.Er {
	return new(State)
}

var _ runtime.VMActor = Actor{}

type ConstructorParams = stake2.ConstructorParams

func (a Actor) Constructor(rt Runtime, params *ConstructorParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerIs(builtin.SystemActorAddr)
	st, err := ConstructState(adt.AsStore(rt), params)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to construct state")
	rt.StateCreate(st)
	return nil
}

// GasOnStakeDeposit is amount of extra gas charged for Stake Deposit
const GasOnStakeDeposit = 888_888_888

func (a Actor) Deposit(rt Runtime, _ *abi.EmptyValue) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	depositAmount := rt.ValueReceived()
	staker := rt.Caller()
	currEpoch := rt.CurrEpoch()

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)
	builtin.RequireParam(rt, depositAmount.GreaterThanEqual(st.MinDepositAmount), "amount to deposit must be greater than or equal to %s", st.MinDepositAmount)

	rt.StateTransaction(&st, func() {
		lockedPrincipalMap, err := adt.AsMap(store, st.LockedPrincipalMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load locked principalsMap")
		lockedPrincipals, found, err := st.LoadLockedPrincipals(store, lockedPrincipalMap, staker)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load locked principals for %v", staker)
		if !found {
			lockedPrincipals = ConstructLockedPrincipals()
		}
		newlyUnlocked := lockedPrincipals.unlockLockedPrincipals(st.PrincipalLockDuration, currEpoch)

		availablePrincipalMap, err := adt.AsMap(store, st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load available principals")
		_, err = st.updateAvailablePrincipal(availablePrincipalMap, staker, newlyUnlocked)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update available principals")
		ap, err := availablePrincipalMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush available principals")
		st.AvailablePrincipalMap = ap

		lockedPrincipals.addLockedPrincipal(depositAmount, currEpoch)
		err = st.putLockedPrincipals(store, lockedPrincipalMap, staker, lockedPrincipals)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put locked principals")
		lpm, err := lockedPrincipalMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush locked principalMap")

		rt.ChargeGas("OnStakeDeposit", GasOnStakeDeposit, 0)
		st.LockedPrincipalMap = lpm
	})
	return nil
}

type WithdrawParams = stake2.WithdrawParams

func (a Actor) WithdrawPrincipal(rt Runtime, params *WithdrawParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()
	if params.AmountRequested.LessThanEqual(abi.NewTokenAmount(0)) {
		rt.Abortf(exitcode.ErrIllegalArgument, "negative or zero fund requested for withdrawal: %s", params.AmountRequested)
	}
	stakerAddr := rt.Caller()

	store := adt.AsStore(rt)
	var st State
	rt.StateTransaction(&st, func() {
		availablePrincipalMap, err := adt.AsMap(store, st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load available principals")
		_, err = st.updateAvailablePrincipal(availablePrincipalMap, stakerAddr, params.AmountRequested.Neg())
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update available principals")
		ap, err := availablePrincipalMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush available principals")
		st.AvailablePrincipalMap = ap

		stakePowerMap, err := adt.AsMap(store, st.StakePowerMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load stake powers")
		err = st.updateStakePower(stakePowerMap, stakerAddr, params.AmountRequested.Neg())
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update stake power")
		sp, err := stakePowerMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush stake power")
		st.StakePowerMap = sp
	})

	code := rt.Send(stakerAddr, builtin.MethodSend, nil, params.AmountRequested, &builtin.Discard{})
	builtin.RequireSuccess(rt, code, "failed to withdraw principal")
	return nil
}

func (a Actor) WithdrawReward(rt Runtime, params *WithdrawParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()
	store := adt.AsStore(rt)
	var st State

	if params.AmountRequested.LessThanEqual(abi.NewTokenAmount(0)) {
		rt.Abortf(exitcode.ErrIllegalArgument, "negative or zero fund requested for withdrawal: %s", params.AmountRequested)
	}
	stakerAddr := rt.Caller()

	rt.StateTransaction(&st, func() {
		availableRewardMap, err := adt.AsMap(store, st.AvailableRewardMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load available rewards")

		err = st.updateAvailableReward(availableRewardMap, stakerAddr, params.AmountRequested.Neg())
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update available rewards")

		ar, err := availableRewardMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush available rewards")
		st.AvailableRewardMap = ar
	})

	code := rt.Send(stakerAddr, builtin.MethodSend, nil, params.AmountRequested, &builtin.Discard{})
	builtin.RequireSuccess(rt, code, "failed to withdraw reward")
	return nil
}

type ChangeMaturePeriodParams = stake2.ChangeMaturePeriodParams

func (a Actor) ChangeMaturePeriod(rt Runtime, params *ChangeMaturePeriodParams) *abi.EmptyValue {
	if params.MaturePeriod <= 0 {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid mature period: %d", params.MaturePeriod)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	if params.MaturePeriod > st.PrincipalLockDuration {
		rt.Abortf(exitcode.ErrIllegalArgument, "mature period %d cannot be greater than principal lock duration %d", params.MaturePeriod, st.PrincipalLockDuration)
	}

	rt.StateTransaction(&st, func() {
		st.MaturePeriod = params.MaturePeriod
	})
	return nil
}

type ChangeRoundPeriodParams = stake2.ChangeRoundPeriodParams

func (a Actor) ChangeRoundPeriod(rt Runtime, params *ChangeRoundPeriodParams) *abi.EmptyValue {
	if params.RoundPeriod <= 0 {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid round period: %d", params.RoundPeriod)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	rt.StateTransaction(&st, func() {
		st.RoundPeriod = params.RoundPeriod
	})
	return nil
}

type ChangePrincipalLockDurationParams = stake2.ChangePrincipalLockDurationParams

func (a Actor) ChangePrincipalLockDuration(rt Runtime, params *ChangePrincipalLockDurationParams) *abi.EmptyValue {
	if params.PrincipalLockDuration <= 0 {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid principal lock duration: %d", params.PrincipalLockDuration)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	if params.PrincipalLockDuration < st.MaturePeriod {
		rt.Abortf(exitcode.ErrIllegalArgument, "principal lock duration %d cannot be less than mature period %d", params.PrincipalLockDuration, st.MaturePeriod)
	}

	rt.StateTransaction(&st, func() {
		st.PrincipalLockDuration = params.PrincipalLockDuration
	})
	return nil
}

type ChangeMinDepositAmountParams = stake2.ChangeMinDepositAmountParams

func (a Actor) ChangeMinDepositAmount(rt Runtime, params *ChangeMinDepositAmountParams) *abi.EmptyValue {
	if params.MinDepositAmount.LessThanEqual(abi.NewTokenAmount(0)) {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid min deposit amount: %s", params.MinDepositAmount)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	rt.StateTransaction(&st, func() {
		st.MinDepositAmount = params.MinDepositAmount
	})
	return nil
}

type ChangeMaxRewardsPerRoundParams = stake2.ChangeMaxRewardsPerRoundParams

func (a Actor) ChangeMaxRewardsPerRound(rt Runtime, params *ChangeMaxRewardsPerRoundParams) *abi.EmptyValue {
	if params.MaxRewardPerRound.LessThan(abi.NewTokenAmount(0)) {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid max reward per round: %s", params.MaxRewardPerRound)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	rt.StateTransaction(&st, func() {
		st.MaxRewardPerRound = params.MaxRewardPerRound
	})
	return nil
}

type ChangeInflationFactorParams = stake2.ChangeInflationFactorParams

func (a Actor) ChangeInflationFactor(rt Runtime, params *ChangeInflationFactorParams) *abi.EmptyValue {
	if params.InflationFactor.LessThan(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "invalid inflation factor: %v", params.InflationFactor)
	}

	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	rt.StateTransaction(&st, func() {
		st.InflationFactor = params.InflationFactor
	})
	return nil
}

func (a Actor) ChangeRootKey(rt Runtime, newRootKey *addr.Address) *abi.EmptyValue {
	var st State
	rt.StateReadonly(&st)
	rt.ValidateImmediateCallerIs(st.RootKey)

	rt.StateTransaction(&st, func() {
		st.RootKey = *newRootKey
	})
	return nil
}

// Called by Cron.
func (a Actor) OnEpochTickEnd(rt Runtime, _ *abi.EmptyValue) *abi.EmptyValue {
	rt.ValidateImmediateCallerIs(builtin.CronActorAddr)

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)
	currEpoch := rt.CurrEpoch()

	rt.StateTransaction(&st, func() {
		// 1. unlocked locked principals、 update available principals、update stake powers
		lockedPrincipalMap, err := adt.AsMap(store, st.LockedPrincipalMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load locked principal map")

		availablePrincipalMap, err := adt.AsMap(store, st.AvailablePrincipalMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load available principals")

		stakePowerMap, err := adt.AsMap(store, st.StakePowerMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load stake powers")

		lpKeys, err := lockedPrincipalMap.CollectKeys()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to collect locked principal keys")

		totalStakePower := big.Zero()
		for _, key := range lpKeys {
			staker, err := addr.NewFromBytes([]byte(key))
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to convert locked principal keys")

			lockedPrincipals, found, err := st.LoadLockedPrincipals(store, lockedPrincipalMap, staker)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load locked principal for: %v", staker)
			if !found {
				// should never happen
				continue
			}

			amountUnlocked := lockedPrincipals.unlockLockedPrincipals(st.PrincipalLockDuration, currEpoch)
			if !amountUnlocked.IsZero() {
				err = st.putLockedPrincipals(store, lockedPrincipalMap, staker, lockedPrincipals)
				builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put locked principals for %v", staker)
			}

			newAvailablePrincipal, err := st.updateAvailablePrincipal(availablePrincipalMap, staker, amountUnlocked)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update available principals")

			powerInLockedPrincipals := lockedPrincipals.stakePower(st.MaturePeriod, currEpoch)
			stakePower := big.Add(powerInLockedPrincipals, newAvailablePrincipal)
			err = stakePowerMap.Put(abi.AddrKey(staker), &stakePower)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update stake power for %v", staker)
			totalStakePower = big.Add(totalStakePower, stakePower)
		}
		st.TotalStakePower = totalStakePower

		lpm, err := lockedPrincipalMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush locked principalMap")
		st.LockedPrincipalMap = lpm

		ap, err := availablePrincipalMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush available principals")
		st.AvailablePrincipalMap = ap

		sp, err := stakePowerMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush stake powers")
		st.StakePowerMap = sp

		// 2. unlock vesting and update available reward
		availableRewardMap, err := adt.AsMap(store, st.AvailableRewardMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load available rewards")

		vestingRewardMap, err := adt.AsMap(store, st.VestingRewardMap, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load vesting rewards")
		vrKeys, err := vestingRewardMap.CollectKeys()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to collect vesting rewards keys")
		newVestingRewards := make(map[addr.Address]*VestingFunds)

		for _, key := range vrKeys {
			staker, err := addr.NewFromBytes([]byte(key))
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to convert vesting rewards keys")

			vestingFunds, found, err := st.LoadVestingFunds(store, vestingRewardMap, staker)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load vesting funds for: %v", staker)
			if !found {
				// should never happen
				continue
			}
			amountUnlocked := vestingFunds.unlockVestedFunds(currEpoch)
			if amountUnlocked.IsZero() {
				continue
			}
			newVestingRewards[staker] = vestingFunds

			err = st.updateAvailableReward(availableRewardMap, staker, amountUnlocked)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to update available reward for %v", staker)
		}
		ar, err := availableRewardMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush available rewards")
		st.AvailableRewardMap = ar

		// 3. distribute vesting reward
		if currEpoch >= st.NextRoundEpoch {
			totalReward := abi.NewTokenAmount(0)
			if st.TotalStakePower.GreaterThan(big.Zero()) {
				totalReward = big.Mul(st.TotalStakePower, st.InflationFactor)
				totalReward = big.Div(totalReward, InflationDenominator)
				totalReward = big.Min(totalReward, st.MaxRewardPerRound)
				if totalReward.GreaterThan(big.Zero()) {
					var power abi.StakePower
					err = stakePowerMap.ForEach(&power, func(key string) error {
						reward := big.Mul(power, totalReward)
						reward = big.Div(reward, st.TotalStakePower)
						if reward.GreaterThan(big.Zero()) {
							staker, err := addr.NewFromBytes([]byte(key))
							if err != nil {
								return err
							}
							vestingFunds, ok := newVestingRewards[staker]
							if !ok {
								var found bool
								vestingFunds, found, err = st.LoadVestingFunds(store, vestingRewardMap, staker)
								builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load vesting funds for: %v", staker)
								if !found {
									vestingFunds = ConstructVestingFunds()
								}
							}
							vestingFunds.addLockedFunds(currEpoch, reward, st.StakePeriodStart, &RewardVestingSpec)
							newVestingRewards[staker] = vestingFunds
						}
						return nil
					})
					builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to iter stake powers")
				}
			}
			st.LastRoundReward = totalReward
			st.NextRoundEpoch += st.RoundPeriod
		}

		for staker, vestingFunds := range newVestingRewards {
			err = st.putVestingFunds(store, vestingRewardMap, staker, vestingFunds)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put vesting funds for %v", staker)
		}
		vr, err := vestingRewardMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush vesting rewards")
		st.VestingRewardMap = vr
	})
	return nil
}
