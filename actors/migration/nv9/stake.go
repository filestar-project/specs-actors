package nv9

import (
	"context"
	stake2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/stake"
	cid "github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	stake3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/stake"
)

type stakeMigrator struct{}

func (m stakeMigrator) migrateState(ctx context.Context, store cbor.IpldStore, in actorMigrationInput) (*actorMigrationResult, error) {
	var inState stake2.State
	if err := store.Get(ctx, in.head, &inState); err != nil {
		return nil, err
	}

	lockedPrincipalMap, err := migrateHAMTRaw(ctx, store, inState.LockedPrincipalMap, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	availablePrincipalMap, err := migrateHAMTRaw(ctx, store, inState.AvailablePrincipalMap, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	stakePowerMap, err := migrateHAMTRaw(ctx, store, inState.StakePowerMap, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	vestingRewardMap, err := migrateHAMTRaw(ctx, store, inState.VestingRewardMap, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	availableRewardMap, err := migrateHAMTRaw(ctx, store, inState.AvailableRewardMap, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	outState := stake3.State{
		RootKey:         inState.RootKey,
		TotalStakePower: inState.TotalStakePower,

		MaturePeriod:          inState.MaturePeriod,
		RoundPeriod:           inState.RoundPeriod,
		PrincipalLockDuration: inState.PrincipalLockDuration,
		MinDepositAmount:      inState.MinDepositAmount,
		MaxRewardPerRound:     inState.MaxRewardPerRound,
		InflationFactor:       inState.InflationFactor,
		LastRoundReward:       inState.LastRoundReward,
		StakePeriodStart:      inState.StakePeriodStart,
		NextRoundEpoch:        inState.NextRoundEpoch,

		LockedPrincipalMap:    lockedPrincipalMap,
		AvailablePrincipalMap: availablePrincipalMap,
		StakePowerMap:         stakePowerMap,
		VestingRewardMap:      vestingRewardMap,
		AvailableRewardMap:    availableRewardMap,
	}
	newHead, err := store.Put(ctx, &outState)
	return &actorMigrationResult{
		newCodeCID: m.migratedCodeCID(),
		newHead:    newHead,
	}, err
}
func (m stakeMigrator) migratedCodeCID() cid.Cid {
	return builtin3.StakeActorCodeID
}