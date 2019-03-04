package reward

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/data"
)

// Rewarder procceses rewards of the target height
type Rewarder interface {
	ProcessReward(Formulator common.Address, ctx *data.Context) error
}
