package reward

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/data"
)

// Rewarder procceses rewards of the target height
type Rewarder interface {
	ProcessReward(Formulator common.Address, ctx *data.Context) error
}
