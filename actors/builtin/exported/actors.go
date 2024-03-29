package exported

import (
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/account"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/cron"
	init_ "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/paych"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/stake"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/system"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/verifreg"
	"github.com/filecoin-project/specs-actors/v3/actors/runtime"
)

func BuiltinActors() []runtime.VMActor {
	return []runtime.VMActor{
		account.Actor{},
		cron.Actor{},
		init_.Actor{},
		market.Actor{},
		miner.Actor{},
		multisig.Actor{},
		paych.Actor{},
		power.Actor{},
		reward.Actor{},
		system.Actor{},
		verifreg.Actor{},
		stake.Actor{},
		token.Actor{},
	}
}
