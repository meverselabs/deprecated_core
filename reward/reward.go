package reward

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/data"
)

// Rewarder procceses rewards of the target height
type Rewarder interface {
	ProcessReward(FormulationAddress common.Address, ctx *data.Context) error
}
