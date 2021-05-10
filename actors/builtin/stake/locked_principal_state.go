package stake

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
)

type LockedPrincipals struct {
	Data []LockedPrincipal
}

type LockedPrincipal struct {
	Amount abi.TokenAmount
	Epoch  abi.ChainEpoch
}

// ConstructVestingFunds constructs empty VestingFunds state.
func ConstructLockedPrincipals() *LockedPrincipals {
	l := new(LockedPrincipals)
	l.Data = nil
	return l
}

func (lps *LockedPrincipals) addLockedPrincipal(amount abi.TokenAmount, epoch abi.ChainEpoch) {
	lps.Data = append(lps.Data, LockedPrincipal{
		amount,
		epoch,
	})
}

func (lps *LockedPrincipals) unlockLockedPrincipals(lockDuration, currEpoch abi.ChainEpoch) abi.TokenAmount {
	amountUnlocked := abi.NewTokenAmount(0)

	lastIndexToRemove := -1
	for i, lp := range lps.Data {
		if lp.Epoch+lockDuration >= currEpoch {
			break
		}

		amountUnlocked = big.Add(amountUnlocked, lp.Amount)
		lastIndexToRemove = i
	}

	// remove all entries upto and including lastIndexToRemove
	if lastIndexToRemove != -1 {
		lps.Data = lps.Data[lastIndexToRemove+1:]
	}

	return amountUnlocked
}

func (lps *LockedPrincipals) stakePower(matureDuration, currEpoch abi.ChainEpoch) abi.StakePower {
	power := abi.NewStakePower(0)
	for _, lp := range lps.Data {
		if lp.Epoch+matureDuration >= currEpoch {
			break
		}
		power = big.Add(power, lp.Amount)
	}
	return power
}
