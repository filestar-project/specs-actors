package token_test

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/filecoin-project/specs-actors/v3/support/mock"
	tutil "github.com/filecoin-project/specs-actors/v3/support/testing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {

}

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, token.Actor{})
}

func TestConstructor(t *testing.T) {
	actor := tokenHarness{token.Actor{}, t}

	t.Run("construct", func(t *testing.T) {
		rt := mock.NewBuilder(builtin.TokenActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)
		actor.constructAndVerify(rt, &abi.EmptyValue{})
		st := getState(rt)

		assert.Equal(t, st.Nonce, big.Zero())
	})
}

func TestToken(t *testing.T) {
	actor := tokenHarness{token.Actor{}, t}
	admin := tutil.NewIDAddr(t, 101)
	tokenOpreratorsMintBatch := []addr.Address {
		tutil.NewIDAddr(t, 102),
		tutil.NewIDAddr(t, 103),
		tutil.NewIDAddr(t, 104),
		tutil.NewIDAddr(t, 105),
	}
	valuesMintBatch := []abi.TokenAmount {
		big.NewInt(66),
		big.NewInt(55),
		big.NewInt(44),
		big.NewInt(33),
	}

	transferFrom := admin
	transferTo := tokenOpreratorsMintBatch[3]

	safeBatchTransferFromTokenIDs := []big.Int {
		big.NewInt(1),
		big.NewInt(2),
	}

	safeBatchTransferFromValues := []big.Int {
		big.NewInt(2),
		big.NewInt(6),
	}

	t.Run("Create-MintBatch", func(t *testing.T) {
		rt := mock.NewBuilder(builtin.TokenActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)

		actor.constructAndVerify(rt, &abi.EmptyValue{})

		actor.createAndVerify(rt, admin, big.NewInt(10), "token 1")

		st := getState(rt)
		assert.Equal(t, big.NewInt(1), st.Nonce)
		creatorsArray, err := adt.AsArray(rt.AdtStore(), st.Creators, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		creatorAddress, found, err := st.GetCreatorAddress(creatorsArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, admin, creatorAddress)

		balanceArray, err := adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap, err := adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		addrTokenAmount, found, err := st.LoadAddrTokenAmount(ataMap, admin)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(10), addrTokenAmount)

		isAllApproveMap, err := adt.AsMap(rt.AdtStore(), st.Approves, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		_, found, err = st.LoadAddrApproveMap(rt.AdtStore(), isAllApproveMap, admin)
		assert.True(t, found)
		assert.Nil(t, err)

		rt.ExpectAbortContainsMessage(exitcode.ErrIllegalArgument, "The caller " + tokenOpreratorsMintBatch[0].String() + " is not the creator for token with tokenID : 1", func() {
			actor.mintBatchAndVerify(rt, tokenOpreratorsMintBatch[0], big.NewInt(1), tokenOpreratorsMintBatch, valuesMintBatch)
		})
		actor.mintBatchAndVerify(rt, admin, big.NewInt(1), tokenOpreratorsMintBatch, valuesMintBatch)
		st = getState(rt)

		balanceArray, err = adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err = st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)

		isAllApproveMap, err = adt.AsMap(rt.AdtStore(), st.Approves, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)

		for idx, _ := range tokenOpreratorsMintBatch {
			ataMap, err = adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
			assert.Nil(t, err)
			addrTokenAmount, found, err = st.LoadAddrTokenAmount(ataMap, tokenOpreratorsMintBatch[idx])
			assert.True(t, found)
			assert.Nil(t, err)
			assert.Equal(t, valuesMintBatch[idx], addrTokenAmount)

			_, found, err = st.LoadAddrApproveMap(rt.AdtStore(), isAllApproveMap, tokenOpreratorsMintBatch[idx])
			assert.True(t, found)
			assert.Nil(t, err)
		}

		actor.createAndVerify(rt, admin, big.NewInt(20), "token 2")
		st = getState(rt)

		assert.Equal(t, big.NewInt(2), st.Nonce)
		creatorsArray, err = adt.AsArray(rt.AdtStore(), st.Creators, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		creatorAddress, found, err = st.GetCreatorAddress(creatorsArray, big.NewInt(2))
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, admin, creatorAddress)

		balanceArray, err = adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err = st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(2))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap, err = adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)

		addrTokenAmount, found, err = st.LoadAddrTokenAmount(ataMap, admin)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(20), addrTokenAmount)
	})

	t.Run("URI", func(t *testing.T) {
		rt := mock.NewBuilder(builtin.TokenActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)

		actor.constructAndVerify(rt, &abi.EmptyValue{})
		actor.createAndVerify(rt, admin, big.NewInt(10), "token 1")
		st := getState(rt)

		urisArray, err := adt.AsArray(rt.AdtStore(), st.URIs, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		tokenURI, found, err := st.LoadTokenURI(urisArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, tokenURI.TokenURI, "token 1")

		actor.ChangeURIAndVerify(rt, admin, "Change URI in test unit", big.NewInt(1))
		st = getState(rt)
		urisArray, err = adt.AsArray(rt.AdtStore(), st.URIs, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		tokenURI, found, err = st.LoadTokenURI(urisArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, tokenURI.TokenURI, "Change URI in test unit")

		rt.ExpectAbortContainsMessage(exitcode.ErrIllegalArgument, "caller has no permission to token ID (1) , which amount is zero", func() {
			actor.ChangeURIAndVerify(rt, tokenOpreratorsMintBatch[0], "Change URI in test unit", big.NewInt(1))
		})

		actor.mintBatchAndVerify(rt, admin, big.NewInt(1), tokenOpreratorsMintBatch, valuesMintBatch)
		actor.ChangeURIAndVerify(rt, tokenOpreratorsMintBatch[0], "Change URI in test unit2", big.NewInt(1))
		st = getState(rt)
		urisArray, err = adt.AsArray(rt.AdtStore(), st.URIs, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		tokenURI, found, err = st.LoadTokenURI(urisArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, tokenURI.TokenURI, "Change URI in test unit2")


	})

	t.Run("Safe(Batch)TransferFrom", func(t *testing.T) {
		rt := mock.NewBuilder(builtin.TokenActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)

		actor.constructAndVerify(rt, &abi.EmptyValue{})

		actor.createAndVerify(rt, admin, big.NewInt(10), "token 1")
		actor.mintBatchAndVerify(rt, admin, big.NewInt(1), tokenOpreratorsMintBatch, valuesMintBatch)
		actor.safeTransferFromAndVerify(rt, admin, transferFrom, transferTo, big.NewInt(1), big.NewInt(5))
		actor.createAndVerify(rt, admin, big.NewInt(20), "token 2")
		actor.safeTransferFromAndVerify(rt, admin, transferFrom, transferTo, big.NewInt(2), big.NewInt(6))
		st := getState(rt)

		balanceArray, err := adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap, err := adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountFrom, found, err := st.LoadAddrTokenAmount(ataMap, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(5), addrTokenAmountFrom)

		addrTokenAmountTo, found, err := st.LoadAddrTokenAmount(ataMap, transferTo)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(38), addrTokenAmountTo)

		balanceArray, err = adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err = st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(2))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap, err = adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountFrom, found, err = st.LoadAddrTokenAmount(ataMap, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(14), addrTokenAmountFrom)

		addrTokenAmountTo, found, err = st.LoadAddrTokenAmount(ataMap, transferTo)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(6), addrTokenAmountTo)

		actor.safeBatchTransferFromAndVerify(rt, admin, transferFrom, transferTo, safeBatchTransferFromTokenIDs, safeBatchTransferFromValues)
		st = getState(rt)

		balanceArray, err = adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap1, found, err := st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap1, err := adt.AsMap(rt.AdtStore(), addrTokenAmountMap1.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)

		addrTokenAmountFrom1, found, err := st.LoadAddrTokenAmount(ataMap1, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(3), addrTokenAmountFrom1)

		addrTokenAmountTo1, found, err := st.LoadAddrTokenAmount(ataMap1, transferTo)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(40), addrTokenAmountTo1)

		addrTokenAmountMap2, found, err := st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(2))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap2, err := adt.AsMap(rt.AdtStore(), addrTokenAmountMap2.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)

		addrTokenAmountFrom2, found, err := st.LoadAddrTokenAmount(ataMap2, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(8), addrTokenAmountFrom2)

		addrTokenAmountTo2, found, err := st.LoadAddrTokenAmount(ataMap2, transferTo)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(12), addrTokenAmountTo2)


	})

	t.Run("Approve-Transfer", func(t *testing.T) {
		rt := mock.NewBuilder(builtin.TokenActorAddr).
			WithCaller(builtin.SystemActorAddr, builtin.SystemActorCodeID).
			Build(t)

		actor.constructAndVerify(rt, &abi.EmptyValue{})

		actor.createAndVerify(rt, transferFrom, big.NewInt(10), "token 1")
		actor.setApproveForAllAndVerify(rt, transferFrom, tokenOpreratorsMintBatch[0], true)
		st := getState(rt)

		isAllApproveMap, err := adt.AsMap(rt.AdtStore(), st.Approves, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		addrApproveMap, found, err := st.LoadAddrApproveMap(rt.AdtStore(), isAllApproveMap, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		apMap, err := adt.AsMap(rt.AdtStore(), addrApproveMap.AddrApproveMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		isApproved, found, err := st.LoadAddrApprove(apMap, tokenOpreratorsMintBatch[0])
		assert.True(t, found)
		assert.Nil(t, err)
		assert.True(t, isApproved)

		actor.safeTransferFromAndVerify(rt, tokenOpreratorsMintBatch[0], transferFrom, transferTo, big.NewInt(1), big.NewInt(5))
		st = getState(rt)

		balanceArray, err := adt.AsArray(rt.AdtStore(), st.Balances, token.LaneStatesAmtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(rt.AdtStore(), balanceArray, big.NewInt(1))
		assert.True(t, found)
		assert.Nil(t, err)
		ataMap, err := adt.AsMap(rt.AdtStore(), addrTokenAmountMap.AddrTokenAmountMap, builtin.DefaultHamtBitwidth)
		assert.Nil(t, err)
		addrTokenAmountFrom, found, err := st.LoadAddrTokenAmount(ataMap, transferFrom)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(5), addrTokenAmountFrom)
		addrTokenAmountTo, found, err := st.LoadAddrTokenAmount(ataMap, transferTo)
		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, big.NewInt(5), addrTokenAmountTo)
	})
}


type tokenHarness struct {
	token.Actor
	t testing.TB
}

func (h *tokenHarness) constructAndVerify(rt *mock.Runtime, params *abi.EmptyValue) {
	rt.Reset()
	rt.ExpectValidateCallerAddr(builtin.SystemActorAddr)
	ret := rt.Call(h.Constructor, params)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) createAndVerify(rt *mock.Runtime, tokenOprerator addr.Address, amount abi.TokenAmount, uri string) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(tokenOprerator, builtin.AccountActorCodeID)
	rt.SetReceived(amount)
	ret := rt.Call(h.Actor.Create, &token.CreateTokenParams{ValueInit: amount, TokenURI: uri})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) mintBatchAndVerify(rt *mock.Runtime, addrCall addr.Address, tokenID big.Int, addrTos []addr.Address, values []abi.TokenAmount) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(addrCall, builtin.AccountActorCodeID)
	ret := rt.Call(h.Actor.MintBatch, &token.MintBatchTokenParams{
		TokenID: tokenID,
		AddrTos: addrTos,
		Values: values,
	})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) ChangeURIAndVerify(rt *mock.Runtime, tokenOperator addr.Address, newURI string, tokenID big.Int) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(tokenOperator, builtin.AccountActorCodeID)
	ret := rt.Call(h.Actor.ChangeURI, &token.ChangeURIParams{
		NewURI: newURI,
		TokenID: tokenID,
	})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) safeTransferFromAndVerify(rt *mock.Runtime, addrCall addr.Address, addrFrom addr.Address, addrTo addr.Address, tokenID big.Int, value abi.TokenAmount) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(addrCall, builtin.AccountActorCodeID)
	rt.SetReceived(value)
	ret := rt.Call(h.Actor.SafeTransferFrom, &token.SafeTransferFromParams{
		AddrFrom: addrFrom,
		AddrTo: addrTo,
		TokenID: tokenID,
		Value: value,
	})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) safeBatchTransferFromAndVerify(rt *mock.Runtime, tokenOperator addr.Address, addrFrom addr.Address, addrTo addr.Address, tokenIDs []big.Int, values []abi.TokenAmount) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(tokenOperator, builtin.AccountActorCodeID)
	ret := rt.Call(h.Actor.SafeBatchTransferFrom, &token.SafeBatchTransferFromParams{
		AddrFrom: addrFrom,
		AddrTo: addrTo,
		TokenIDs: tokenIDs,
		Values: values,
	})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tokenHarness) setApproveForAllAndVerify(rt *mock.Runtime, addrCall addr.Address, addrTo addr.Address, approved bool) {
	rt.ExpectValidateCallerAny()
	rt.SetCaller(addrCall, builtin.AccountActorCodeID)
	ret := rt.Call(h.Actor.SetApproveForAll, &token.SetApproveForAllParams{
		AddrTo: addrTo,
		Approved: approved,
	})
	assert.Nil(h.t, ret)
	rt.Verify()
}

func getState(rt *mock.Runtime) *token.State {
	var st token.State
	rt.GetState(&st)
	return &st
}

