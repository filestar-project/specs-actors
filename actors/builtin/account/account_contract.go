package account

import (
	"encoding/hex"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filestar-project/evm-adapter/evm"

	"github.com/filecoin-project/specs-actors/v2/actors/runtime"
)

type ContractCreateParams struct {
	Code []byte
}

var _ = &evm.EVM{}

// Create new EVM contract
func (a Actor) CreateContract(rt runtime.Runtime, params *ContractCreateParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()
	fmt.Printf("accountActor.CreateContract, code = %s", hex.EncodeToString(params.Code))
	return nil
}

type ContractCallParams struct {
	Code []byte
}

// Call EVM contract
func (a Actor) CallContract(rt runtime.Runtime, params *ContractCallParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()
	fmt.Printf("accountActor.CallContract, code = %s", hex.EncodeToString(params.Code))
	return nil
}
