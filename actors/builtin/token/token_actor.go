package token

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	"github.com/ipfs/go-cid"
)

type Runtime = runtime.Runtime

type Actor struct {}

func (a Actor) Exports() []interface{} {
	return []interface{}{
		builtin.MethodConstructor:   	a.Constructor,
		2:								a.Create,
		3:								a.MintBatch,
		4:								a.BalanceOf,
		5:								a.BalanceOfBatch,
		6:								a.GetURI,
		7:								a.ChangeURI,
		8:								a.SafeTransferFrom,
		9:								a.SafeBatchTransferFrom,
		10:								a.SetApproveForAll,
		11:								a.IsApproveForAll,
	}
}

func (a Actor) Code() cid.Cid {
	return builtin.TokenActorCodeID
}

func (a Actor) IsSingleton() bool {
	return true
}

func (a Actor) State() cbor.Er {
	return new(State)
}

var _ runtime.VMActor = Actor{}

func (a Actor) Constructor(rt Runtime, _ *abi.EmptyValue) *abi.EmptyValue {
	rt.ValidateImmediateCallerIs(builtin.SystemActorAddr)

	emptyArray, err := adt.MakeEmptyArray(adt.AsStore(rt)).Root()
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")

	emptyMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")

	st := ConstructState(emptyArray, emptyMap)
	rt.StateCreate(st)

	return nil
}

type CreateTokenParams struct {
	ValueInit abi.TokenAmount
	TokenURI  string
}

// GasOnTokenCreate is amount of extra gas charged for Token Create
const GasOnTokenCreate = 888_888_888

func (a Actor) Create(rt Runtime, params *CreateTokenParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	tokenOperator := rt.Caller()

	if params.ValueInit.LessThan(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Illegal token amount : %v", params.ValueInit)
	}

	store := adt.AsStore(rt)

	var st State
	rt.StateReadonly(&st)

	rt.StateTransaction(&st, func() {
		creatorsArray, err := adt.AsArray(store, st.Creators)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load creatorsArray")

		st.Nonce = big.Add(st.Nonce, big.NewInt(1))
		err = st.setCreatorAddress(creatorsArray, tokenOperator, st.Nonce)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create a new token type")
		cta, err := creatorsArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush creatorsArray")
		rt.ChargeGas("OnTokenCreate", GasOnTokenCreate, 0)
		st.Creators = cta

		urisArray, err := adt.AsArray(store, st.URIs)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load urisArray")
		err = st.setTokenURI(urisArray, params.TokenURI, st.Nonce)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to set a new token uri")
		ura, err := urisArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush urisArray")
		st.URIs = ura

		balanceMap := adt.MakeEmptyMap(adt.AsStore(rt))
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")
		err = st.putAddrTokenAmount(balanceMap, tokenOperator, params.ValueInit)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceMap")
		blm, err := balanceMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush balanceMap")

		balanceArray, err := adt.AsArray(store, st.Balances)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")
		var addrTokenAmountMap AddrTokenAmountMap
		addrTokenAmountMap.AddrTokenAmountMap = blm
		err = st.putAddrTokenAmountMap(store, balanceArray, st.Nonce, &addrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceArray")
		bla, err := balanceArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush balanceArray")
		st.Balances = bla

		apMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")

		isAllApproveMap, err := adt.AsMap(store, st.Approves)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")
		var addrApproveMap AddrApproveMap
		addrApproveMap.AddrApproveMap = apMap
		err = st.putAddrApproveMap(store, isAllApproveMap, tokenOperator, &addrApproveMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")
		iam, err := isAllApproveMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush isAllApproveMap")
		st.Approves = iam
	})

	return nil
}

type MintBatchTokenParams struct {
	TokenID 	big.Int
	AddrTos 	[]addr.Address
	Values 		[]abi.TokenAmount
}

func (a Actor) MintBatch(rt Runtime, params *MintBatchTokenParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	if len(params.AddrTos) != len(params.Values) {
		rt.Abortf(exitcode.ErrIllegalArgument, "The length of the address array does not match the transfer value array")
	}

	for i, _ := range params.AddrTos {
		if params.AddrTos[i].Empty() {
			rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v", params.AddrTos[i])
		}
		if params.Values[i].LessThan(big.Zero()) {
			rt.Abortf(exitcode.ErrIllegalArgument, "Illegal token amount : %v", params.Values[i])
		}
	}

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	if params.TokenID.GreaterThan(st.Nonce) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenID, st.Nonce)
	}

	tokenCreatorsArray, err := adt.AsArray(store, st.Creators)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load creatorsArray")
	creatorAddress, found, err := st.GetCreatorAddress(tokenCreatorsArray, params.TokenID)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to get creators by tokenID : %v", params.TokenID)
	if !found || creatorAddress != rt.Caller() {
		rt.Abortf(exitcode.ErrIllegalArgument, "The caller %v is not the creator for token with tokenID : %v", rt.Caller(), params.TokenID)
	}

	rt.StateTransaction(&st, func() {
		balanceArray, err := adt.AsArray(store, st.Balances)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")
		isAllApproveMap, err := adt.AsMap(store, st.Approves)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")

		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenID)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenID)
		if !found {
			addrTokenAmountMap = nil
		}
		tokenAmountMap, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")
		for idx, _ := range params.AddrTos {
			tokenAmount, found, err := st.LoadAddrTokenAmount(tokenAmountMap, params.AddrTos[idx])
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmount for %v - %v", params.TokenID, params.AddrTos[idx])
			tokenAmount = big.Add(tokenAmount, params.Values[idx])
			err = st.putAddrTokenAmount(tokenAmountMap, params.AddrTos[idx], tokenAmount)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put addrTokenAmount for %v - %v", params.TokenID, params.AddrTos[idx])

			_, found, err = st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrTos[idx])
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrTos[idx])
			if !found {
				apMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
				builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")
				var addrApproveMap AddrApproveMap
				addrApproveMap.AddrApproveMap = apMap
				err = st.putAddrApproveMap(store, isAllApproveMap, params.AddrTos[idx], &addrApproveMap)
				builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")
			}
		}
		tam, err := tokenAmountMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush tokenAmountMap")
		addrTokenAmountMap.AddrTokenAmountMap = tam

		err = st.putAddrTokenAmountMap(store, balanceArray, params.TokenID, addrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceArray")
		bla, err := balanceArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush balanceArray")
		st.Balances = bla

		iam, err := isAllApproveMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush isAllApproveMap")
		st.Approves = iam

	})

	return nil
}

type BalanceOfParams struct {
	AddrOwner 	addr.Address
	TokenID 	big.Int
}

type BalanceOfResults struct {
	Balance 	abi.TokenAmount
}

func (a Actor) BalanceOf(rt Runtime, params *BalanceOfParams) *BalanceOfResults {
	rt.ValidateImmediateCallerAcceptAny()

	if params.AddrOwner.Empty() {
		rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v", params.AddrOwner)
	}

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	if params.TokenID.GreaterThan(st.Nonce) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenID, st.Nonce)
	}

	balanceArray, err := adt.AsArray(store, st.Balances)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")
	addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenID)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenID)
	if !found {
		return &BalanceOfResults{Balance: big.Zero()}
	}

	tokenAmountMap, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")

	addrTokenAmount, found, err := st.LoadAddrTokenAmount(tokenAmountMap, params.AddrOwner)
	if !found {
		return &BalanceOfResults{Balance: big.Zero()}
	}

	return &BalanceOfResults{Balance: addrTokenAmount}
}

type BalanceOfBatchParams struct {
	AddrOwners 	[]addr.Address
	TokenIDs 	[]big.Int
}

type BalanceOfBatchResults struct {
	Balances 	[]abi.TokenAmount
}

func (a Actor) BalanceOfBatch(rt Runtime, params *BalanceOfBatchParams) *BalanceOfBatchResults {
	rt.ValidateImmediateCallerAcceptAny()

	if len(params.AddrOwners) != len(params.TokenIDs) {
		rt.Abortf(exitcode.ErrIllegalArgument, "The length of the address array does not match the tokenID array")
	}

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	for i, _ := range params.AddrOwners {
		if params.AddrOwners[i].Empty() {
			rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v", params.AddrOwners[i])
		}
		if params.TokenIDs[i].LessThan(st.Nonce) {
			rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenIDs[i], st.Nonce)
		}
	}

	balanceArray, err := adt.AsArray(store, st.Balances)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")

	var tokenAmounts []abi.TokenAmount

	for idx, _ := range params.AddrOwners {
		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenIDs[idx])
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenIDs[idx])
		if !found {
			tokenAmounts = append(tokenAmounts, big.Zero())
			continue
		}
		tokenAmountMap, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")

		addrTokenAmount, found, err := st.LoadAddrTokenAmount(tokenAmountMap, params.AddrOwners[idx])
		if !found {
			tokenAmounts = append(tokenAmounts, big.Zero())
			continue
		}
		tokenAmounts = append(tokenAmounts, addrTokenAmount)
	}

	return &BalanceOfBatchResults{Balances: tokenAmounts}
}

type GetURIParams struct {
	TokenID		big.Int
}

func (a Actor) GetURI(rt Runtime, params *GetURIParams) *TokenURI {
	rt.ValidateImmediateCallerAcceptAny()

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	if params.TokenID.GreaterThan(st.Nonce) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenID, st.Nonce)
	}

	tokenURIArray, err := adt.AsArray(store, st.URIs)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load urisArray")

	tokenURI, found, err := st.LoadTokenURI(tokenURIArray, params.TokenID)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load urisArray")

	if !found {
		return &TokenURI{TokenURI: ""}
	}

	return tokenURI
}

type ChangeURIParams struct {
	NewURI 		string
	TokenID 	big.Int
}

func (a Actor) ChangeURI(rt Runtime, params *ChangeURIParams) *abi.EmptyValue {

	rt.ValidateImmediateCallerAcceptAny()

	tokenOperator := rt.Caller()
	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	if params.TokenID.GreaterThan(st.Nonce) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenID, st.Nonce)
	}

	balanceArray, err := adt.AsArray(store, st.Balances)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")
	addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenID)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenID)
	if !found {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) , its amount info is not exist", params.TokenID)
	}

	tokenAmountMap, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")

	addrTokenAmount, found, err := st.LoadAddrTokenAmount(tokenAmountMap, tokenOperator)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmount")
	if !found || addrTokenAmount.LessThan(big.Zero()){
		rt.Abortf(exitcode.ErrIllegalArgument, "caller has no permission to token ID (%v) , which amount is zero", params.TokenID)
	}

	rt.StateTransaction(&st, func() {
		urisArray, err := adt.AsArray(store, st.URIs)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load urisArray")
		err = st.setTokenURI(urisArray, params.NewURI, params.TokenID)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to set a new token uri")
		ura, err := urisArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush urisArray")
		st.URIs = ura
	})

	return nil
}

type SafeTransferFromParams struct {
	AddrFrom 	addr.Address
	AddrTo 		addr.Address
	TokenID 	big.Int
	Value 		abi.TokenAmount
}

func (a Actor) SafeTransferFrom(rt Runtime, params *SafeTransferFromParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	if params.AddrFrom.Empty() || params.AddrTo.Empty() {
		rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v , %v", params.AddrFrom, params.AddrTo)
	}

	if params.AddrFrom == params.AddrTo {
		rt.Abortf(exitcode.ErrIllegalArgument, "cant not be the same address : %v , %v", params.AddrFrom, params.AddrTo)
	}

	if params.Value.LessThan(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Illegal token amount : %v", params.Value)
	}

	tokenOperator := rt.Caller()
	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	if params.TokenID.GreaterThan(st.Nonce) {
		rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenID, st.Nonce)
	}

	isAllApproveMap, err := adt.AsMap(store, st.Approves)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")

	if params.AddrFrom != tokenOperator {
		addrApproveMap, found, err := st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrFrom)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrFrom)
		if !found {
			rt.Abortf(exitcode.ErrIllegalArgument, "The caller does not have permission")
		}
		approveMap, err := adt.AsMap(store, addrApproveMap.AddrApproveMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")
		res, found, err := st.LoadAddrApprove(approveMap, tokenOperator)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")
		if !found || !res {
			rt.Abortf(exitcode.ErrIllegalArgument, "The caller does not have permission")
		}
	}

	balanceArray, err := adt.AsArray(store, st.Balances)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")
	addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenID)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenID)
	if !found {
		addrTokenAmountMap = nil
	}

	addrTokenAmount, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")
	tokenAmountFrom, found, err := st.LoadAddrTokenAmount(addrTokenAmount, params.AddrFrom)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")
	if ((addrTokenAmountMap == nil || !found) && params.Value.GreaterThan(big.Zero())) || params.Value.GreaterThan(tokenAmountFrom) {
		rt.Abortf(exitcode.ErrIllegalArgument, "The balance is not enough for transfer")
	}

	rt.StateTransaction(&st, func() {
		tokenAmountFrom = big.Sub(tokenAmountFrom, params.Value)
		err = st.putAddrTokenAmount(addrTokenAmount, params.AddrFrom, tokenAmountFrom)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceMap")

		tokenAmountTo, found, err := st.LoadAddrTokenAmount(addrTokenAmount, params.AddrTo)
		tokenAmountTo = big.Add(tokenAmountTo, params.Value)
		err = st.putAddrTokenAmount(addrTokenAmount, params.AddrTo, tokenAmountTo)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceMap")

		_, found, err = st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrTo)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrTo)
		if !found {
			apMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")
			var addrApproveMapTmp AddrApproveMap
			addrApproveMapTmp.AddrApproveMap = apMap
			err = st.putAddrApproveMap(store, isAllApproveMap, params.AddrTo, &addrApproveMapTmp)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")
		}

		ata, err := addrTokenAmount.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush tokenAmountMap")
		addrTokenAmountMap.AddrTokenAmountMap = ata

		err = st.putAddrTokenAmountMap(store, balanceArray, params.TokenID, addrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceArray")
		bla, err := balanceArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush balanceArray")
		st.Balances = bla

		iam, err := isAllApproveMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush isAllApproveMap")
		st.Approves = iam
	})

	return nil
}

type SafeBatchTransferFromParams struct {
	AddrFrom 	addr.Address
	AddrTo 		addr.Address
	TokenIDs 	[]big.Int
	Values 		[]abi.TokenAmount
}

func (a Actor) SafeBatchTransferFrom(rt Runtime, params *SafeBatchTransferFromParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	if params.AddrFrom.Empty() || params.AddrTo.Empty() {
		rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v , %v", params.AddrFrom, params.AddrTo)
	}

	if params.AddrFrom == params.AddrTo {
		rt.Abortf(exitcode.ErrIllegalArgument, "cant not be the same address : %v , %v", params.AddrFrom, params.AddrTo)
	}

	if len(params.TokenIDs) != len(params.Values) {
		rt.Abortf(exitcode.ErrIllegalArgument, "The length of the tokenIDs array does not match the transfer value array")
	}

	tokenOperator := rt.Caller()
	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	isAllApproveMap, err := adt.AsMap(store, st.Approves)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")

	if params.AddrFrom != tokenOperator {
		addrApproveMap, found, err := st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrFrom)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrFrom)
		if !found {
			rt.Abortf(exitcode.ErrIllegalArgument, "The caller does not have permission")
		}
		approveMap, err := adt.AsMap(store, addrApproveMap.AddrApproveMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")
		res, found, err := st.LoadAddrApprove(approveMap, tokenOperator)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")
		if !found || !res {
			rt.Abortf(exitcode.ErrIllegalArgument, "The caller does not have permission")
		}
	}

	var addrTokenAmountMaps []*AddrTokenAmountMap
	var addrTokenAmounts	[]*adt.Map
	var tokenAmountFroms 	[]abi.TokenAmount
	var tokenAmountTos 		[]abi.TokenAmount

	balanceArray, err := adt.AsArray(store, st.Balances)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceArray")

	for idx, _ := range params.TokenIDs {
		if params.TokenIDs[idx].GreaterThan(st.Nonce) {
			rt.Abortf(exitcode.ErrIllegalArgument, "Invalid token ID (%v) greater than actual maxID (%v)", params.TokenIDs[idx], st.Nonce)
		}
		if params.Values[idx].LessThan(big.Zero()) {
			rt.Abortf(exitcode.ErrIllegalArgument, "Illegal token amount : %v", params.Values[idx])
		}

		addrTokenAmountMap, found, err := st.LoadAddrTokenAmountMap(store, balanceArray, params.TokenIDs[idx])
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrTokenAmountMap for %v", params.TokenIDs[idx])
		if !found {
			addrTokenAmountMap = nil
		}

		addrTokenAmount, err := adt.AsMap(store, addrTokenAmountMap.AddrTokenAmountMap)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")
		tokenAmountFrom, found, err := st.LoadAddrTokenAmount(addrTokenAmount, params.AddrFrom)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")
		if ((addrTokenAmountMap == nil || !found) && params.Values[idx].GreaterThan(big.Zero())) || params.Values[idx].GreaterThan(tokenAmountFrom) {
			rt.Abortf(exitcode.ErrIllegalArgument, "The %vth balance is not enough for transfer : %v-%v", idx + 1, tokenAmountFrom, params.Values[idx])
		}

		tokenAmountTo, found, err := st.LoadAddrTokenAmount(addrTokenAmount, params.AddrTo)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load balanceMap")

		addrTokenAmountMaps = append(addrTokenAmountMaps, addrTokenAmountMap)
		addrTokenAmounts = append(addrTokenAmounts, addrTokenAmount)
		tokenAmountFroms = append(tokenAmountFroms, big.Sub(tokenAmountFrom, params.Values[idx]))
		tokenAmountTos = append(tokenAmountTos, big.Add(tokenAmountTo, params.Values[idx]))
		// fmt.Println(addrTokenAmountMap, addrTokenAmount, tokenAmountFrom, tokenAmountTo)
	}

	for idx, _ := range params.TokenIDs {
		rt.StateTransaction(&st, func() {

			err = st.putAddrTokenAmount(addrTokenAmounts[idx], params.AddrFrom, tokenAmountFroms[idx])
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceMap")

			err = st.putAddrTokenAmount(addrTokenAmounts[idx], params.AddrTo, tokenAmountTos[idx])
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceMap")

			ata, err := addrTokenAmounts[idx].Root()
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush tokenAmountMap")
			addrTokenAmountMaps[idx].AddrTokenAmountMap = ata

			err = st.putAddrTokenAmountMap(store, balanceArray, params.TokenIDs[idx], addrTokenAmountMaps[idx])
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put balanceArray")

			// fmt.Println(addrTokenAmountMaps[idx], addrTokenAmounts[idx], tokenAmountFroms[idx], tokenAmountTos[idx])
		})
	}

	rt.StateTransaction(&st, func() {

		bla, err := balanceArray.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush balanceArray")
		st.Balances = bla

		_, found, err := st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrTo)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrTo)
		if !found {
			apMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")
			var addrApproveMapTmp AddrApproveMap
			addrApproveMapTmp.AddrApproveMap = apMap
			err = st.putAddrApproveMap(store, isAllApproveMap, params.AddrTo, &addrApproveMapTmp)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")
		}
		iam, err := isAllApproveMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush isAllApproveMap")
		st.Approves = iam
	})

	return nil
}

type SetApproveForAllParams struct {
	AddrTo 		addr.Address
	Approved 	bool
}

func (a Actor) SetApproveForAll(rt Runtime, params *SetApproveForAllParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	if params.AddrTo.Empty() {
		rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v", params.AddrTo)
	}

	tokenOperator := rt.Caller()

	if tokenOperator == params.AddrTo {
		rt.Abortf(exitcode.ErrIllegalArgument, "target address cant be self: %v", params.AddrTo)
	}

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	rt.StateTransaction(&st, func() {
		isAllApproveMap, err := adt.AsMap(store, st.Approves)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")

		addrApproveMap, found, err := st.LoadAddrApproveMap(store, isAllApproveMap, tokenOperator)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", tokenOperator)
		if !found {
			apMap := adt.MakeEmptyMap(adt.AsStore(rt))
			err = st.putAddrApprove(apMap, params.AddrTo, params.Approved)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put approveMap")

			var addrApproveMapTmp AddrApproveMap
			addrApproveMapTmp.AddrApproveMap, err = apMap.Root()
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap.root")
			err = st.putAddrApproveMap(store, isAllApproveMap, tokenOperator, addrApproveMap)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")

		} else {
			approveMap, err := adt.AsMap(store, addrApproveMap.AddrApproveMap)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")

			err = st.putAddrApprove(approveMap, params.AddrTo, params.Approved)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put approveMap")

			apMap, err := approveMap.Root()
			addrApproveMap.AddrApproveMap = apMap
			err = st.putAddrApproveMap(store, isAllApproveMap, tokenOperator, addrApproveMap)
			builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to put isAllApproveMap")
		}

		iam, err := isAllApproveMap.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to flush isAllApproveMap")
		st.Approves = iam
	})

	return nil
}

type IsApproveForAllParams struct {
	AddrFrom	addr.Address
	AddrTo 		addr.Address
}

type IsApprovedForAllResults struct {
	IsApproved 	bool
}

func (a Actor) IsApproveForAll(rt Runtime, params *IsApproveForAllParams) *IsApprovedForAllResults {
	rt.ValidateImmediateCallerAcceptAny()

	if params.AddrFrom.Empty() || params.AddrTo.Empty() {
		rt.Abortf(exitcode.ErrIllegalArgument, "empty address : %v , %v", params.AddrFrom, params.AddrTo)
	}

	store := adt.AsStore(rt)
	var st State
	rt.StateReadonly(&st)

	isAllApproveMap, err := adt.AsMap(store, st.Approves)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load isAllApproveMap")

	addrApproveMap, found, err := st.LoadAddrApproveMap(store, isAllApproveMap, params.AddrFrom)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load addrApproveMap for %v", params.AddrFrom)
	if !found {
		apMap, err := adt.MakeEmptyMap(adt.AsStore(rt)).Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to create state")
		var addrApproveMapTmp AddrApproveMap
		addrApproveMapTmp.AddrApproveMap = apMap
		return &IsApprovedForAllResults{false}
	}

	approveMap, err := adt.AsMap(store, addrApproveMap.AddrApproveMap)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")

	res, found, err := st.LoadAddrApprove(approveMap, params.AddrTo)
	builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "failed to load approveMap")

	if !found {
		return &IsApprovedForAllResults{false}
	}
	return &IsApprovedForAllResults{res}
}


