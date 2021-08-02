package stake

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	stake2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/stake"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	"sort"
)

// Rounds e to the nearest exact multiple of the quantization unit offset by
// offsetSeed % unit, rounding up.
// This function is equivalent to `unit * ceil(e - (offsetSeed % unit) / unit) + (offsetSeed % unit)`
// with the variables/operations are over real numbers instead of ints.
// Precondition: unit >= 0 else behaviour is undefined
func quantizeUp(e abi.ChainEpoch, unit abi.ChainEpoch, offsetSeed abi.ChainEpoch) abi.ChainEpoch {
	offset := offsetSeed % unit

	remainder := (e - offset) % unit
	quotient := (e - offset) / unit
	// Don't round if epoch falls on a quantization epoch
	if remainder == 0 {
		return unit*quotient + offset
	}
	// Negative truncating division rounds up
	if e-offset < 0 {
		return unit*quotient + offset
	}
	return unit*(quotient+1) + offset
}

// The vesting schedule for total rewards (block reward + gas reward) earned by a block producer.
var RewardVestingSpec = miner.VestSpec{ // PARAM_SPEC
	InitialDelay: abi.ChainEpoch(0),
	VestPeriod:   abi.ChainEpoch(180 * builtin.EpochsInDay),
	StepDuration: abi.ChainEpoch(1 * builtin.EpochsInDay),
	Quantization: 12 * builtin.EpochsInHour,
}

type VestingFund = stake2.VestingFund

// VestingFunds represents the vesting table state for the miner.
// It is a slice of (VestingEpoch, VestingAmount).
// The slice will always be sorted by the VestingEpoch.
type VestingFunds struct {
	Funds []VestingFund
}

func (v *VestingFunds) unlockVestedFunds(currEpoch abi.ChainEpoch) abi.TokenAmount {
	amountUnlocked := abi.NewTokenAmount(0)

	lastIndexToRemove := -1
	for i, vf := range v.Funds {
		if vf.Epoch >= currEpoch {
			break
		}

		amountUnlocked = big.Add(amountUnlocked, vf.Amount)
		lastIndexToRemove = i
	}

	// remove all entries upto and including lastIndexToRemove
	if lastIndexToRemove != -1 {
		v.Funds = v.Funds[lastIndexToRemove+1:]
	}

	return amountUnlocked
}

func (v *VestingFunds) addLockedFunds(currEpoch abi.ChainEpoch, vestingSum abi.TokenAmount,
	stakePeriodStart abi.ChainEpoch, spec *miner.VestSpec) {
	// maps the epochs in VestingFunds to their indices in the slice
	epochToIndex := make(map[abi.ChainEpoch]int, len(v.Funds))
	for i, vf := range v.Funds {
		epochToIndex[vf.Epoch] = i
	}

	// Quantization is aligned with when regular cron will be invoked, in the last epoch of deadlines.
	vestBegin := currEpoch + spec.InitialDelay // Nothing unlocks here, this is just the start of the clock.
	vestPeriod := big.NewInt(int64(spec.VestPeriod))
	vestedSoFar := big.Zero()
	for e := vestBegin + spec.StepDuration; vestedSoFar.LessThan(vestingSum); e += spec.StepDuration {
		vestEpoch := quantizeUp(e, spec.Quantization, stakePeriodStart)
		elapsed := vestEpoch - vestBegin

		targetVest := big.Zero() //nolint:ineffassign
		if elapsed < spec.VestPeriod {
			// Linear vesting
			targetVest = big.Div(big.Mul(vestingSum, big.NewInt(int64(elapsed))), vestPeriod)
		} else {
			targetVest = vestingSum
		}

		vestThisTime := big.Sub(targetVest, vestedSoFar)
		vestedSoFar = targetVest

		// epoch already exists. Load existing entry
		// and update amount.
		if index, ok := epochToIndex[vestEpoch]; ok {
			currentAmt := v.Funds[index].Amount
			v.Funds[index].Amount = big.Add(currentAmt, vestThisTime)
		} else {
			// append a new entry -> slice will be sorted by epoch later.
			entry := VestingFund{Epoch: vestEpoch, Amount: vestThisTime}
			v.Funds = append(v.Funds, entry)
			epochToIndex[vestEpoch] = len(v.Funds) - 1
		}
	}

	// sort slice by epoch
	sort.Slice(v.Funds, func(first, second int) bool {
		return v.Funds[first].Epoch < v.Funds[second].Epoch
	})
}

// ConstructVestingFunds constructs empty VestingFunds state.
func ConstructVestingFunds() *VestingFunds {
	v := new(VestingFunds)
	v.Funds = nil
	return v
}
