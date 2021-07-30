package nv9

import (
	"context"

	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	cid "github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	smoothing3 "github.com/filecoin-project/specs-actors/v3/actors/util/smoothing"
)

type powerMigrator struct{}

func (m powerMigrator) migrateState(ctx context.Context, store cbor.IpldStore, in actorMigrationInput) (*actorMigrationResult, error) {
	var inState power2.State
	if err := store.Get(ctx, in.head, &inState); err != nil {
		return nil, err
	}

	var proofValidationBatchOut *cid.Cid
	if inState.ProofValidationBatch != nil {
		proofValidationBatchOutCID, err := migrateHAMTAMTRaw(ctx, store, *inState.ProofValidationBatch, builtin3.DefaultHamtBitwidth, power3.ProofValidationBatchAmtBitwidth)
		if err != nil {
			return nil, err
		}
		proofValidationBatchOut = &proofValidationBatchOutCID
	}

	claimsOut, err := migrateHAMTRaw(ctx, store, inState.Claims, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	cronEventQueueOut, err := migrateHAMTAMTRaw(ctx, store, inState.CronEventQueue, power3.CronQueueHamtBitwidth, power3.CronQueueAmtBitwidth)
	if err != nil {
		return nil, err
	}

	outState := power3.State{
		TotalRawBytePower:         inState.TotalRawBytePower,
		TotalBytesCommitted:       inState.TotalBytesCommitted,
		TotalQualityAdjPower:      inState.TotalQualityAdjPower,
		TotalQABytesCommitted:     inState.TotalQABytesCommitted,
		TotalPledgeCollateral:     inState.TotalPledgeCollateral,
		ThisEpochRawBytePower:     inState.ThisEpochRawBytePower,
		ThisEpochQualityAdjPower:  inState.ThisEpochQualityAdjPower,
		ThisEpochPledgeCollateral: inState.ThisEpochPledgeCollateral,
		ThisEpochQAPowerSmoothed:  smoothing3.FilterEstimate(inState.ThisEpochQAPowerSmoothed),
		MinerCount:                inState.MinerCount,
		MinerAboveMinPowerCount:   inState.MinerAboveMinPowerCount,
		CronEventQueue:            cronEventQueueOut,
		FirstCronEpoch:            inState.FirstCronEpoch,
		Claims:                    claimsOut,
		ProofValidationBatch:      proofValidationBatchOut,
	}
	newHead, err := store.Put(ctx, &outState)
	return &actorMigrationResult{
		newCodeCID: m.migratedCodeCID(),
		newHead:    newHead,
	}, err
}

func (m powerMigrator) migratedCodeCID() cid.Cid {
	return builtin3.StoragePowerActorCodeID
}
