package consensus

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fletaio/common/util"
	"github.com/fletaio/core/amount"
)

// CommunityPolicy defines a policy of community formulator
type CommunityPolicy struct {
	CommissionRatio1000 uint32
	MinimumStaking      *amount.Amount
	MaximumStaking      *amount.Amount
}

// Clone returns the clonend value of it
func (pc *CommunityPolicy) Clone() *CommunityPolicy {
	return &CommunityPolicy{
		CommissionRatio1000: pc.CommissionRatio1000,
		MinimumStaking:      pc.MinimumStaking.Clone(),
		MaximumStaking:      pc.MaximumStaking.Clone(),
	}
}

// WriteTo is a serialization function
func (pc *CommunityPolicy) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint32(w, pc.CommissionRatio1000); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := pc.MinimumStaking.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := pc.MaximumStaking.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (pc *CommunityPolicy) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		pc.CommissionRatio1000 = v
	}
	if n, err := pc.MinimumStaking.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := pc.MaximumStaking.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (pc *CommunityPolicy) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"commission_ratio_1000":`)
	if bs, err := json.Marshal(pc.CommissionRatio1000); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"minimum_staking":`)
	if bs, err := json.Marshal(pc.MinimumStaking); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"maximum_staking":`)
	if bs, err := json.Marshal(pc.MaximumStaking); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}

// StakingPolicy defines a staking policy user
type StakingPolicy struct {
	AutoStaking bool
}
