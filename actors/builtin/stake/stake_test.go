package stake_test

import (
	//"bytes"
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/stretchr/testify/assert"

	//"encoding/binary"
	//"fmt"
	//"strings"
	"testing"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/stake"
	//"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	//"github.com/filecoin-project/specs-actors/v2/actors/util/smoothing"
	"github.com/filecoin-project/specs-actors/v2/support/mock"
	tutil "github.com/filecoin-project/specs-actors/v2/support/testing"
)

func init() {

}

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, stake.Actor{})
}

func TestConstruction(t *testing.T) {
	actor := stakeHarness{stake.Actor{}, t}
	admin := tutil.NewIDAddr(t, 100)

	t.Run("construct", func(t *testing.T) {
		rt := mock.NewBuilder(context.Background(), builtin.StakeActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)
		params := stake.ConstructorParams{
			RootKey:         admin,
			FirstRoundEpoch: 123,
		}
		actor.constructAndVerify(rt, &params)
		st := getState(rt)

		assert.Equal(t, admin, st.RootKey)
		assert.Equal(t, big.Zero(), st.TotalStakePower)
		//assert.Equal(t, stake.DefaultMaturePeriod, st.MaturePeriod)
		//assert.Equal(t, stake.DefaultRoundPeriod, st.RoundPeriod)
		//assert.Equal(t, stake.DefaultPrincipalLockDuration, st.PrincipalLockDuration)
		//assert.Equal(t, stake.DefaultMinDepositAmount, st.MinDepositAmount)
		//assert.Equal(t, stake.DefaultMaxRewardsPerRound, st.MaxRewardPerRound)
		//assert.Equal(t, stake.DefaultInflationFactor, st.InflationFactor)
		assert.Equal(t, big.Zero(), st.LastRoundReward)
		assert.Equal(t, abi.ChainEpoch(123), st.NextRoundEpoch)
	})
}

func TestDeposit(t *testing.T) {
	//actor := stakeHarness{stake.Actor{}, t}
	//admin := tutil.NewIDAddr(t, 100)

}

type stakeHarness struct {
	stake.Actor
	t testing.TB
}

func (h *stakeHarness) constructAndVerify(rt *mock.Runtime, params *stake.ConstructorParams) {
	rt.ExpectValidateCallerAddr(builtin.SystemActorAddr)
	ret := rt.Call(h.Constructor, params)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func getState(rt *mock.Runtime) *stake.State {
	var st stake.State
	rt.GetState(&st)
	return &st
}
