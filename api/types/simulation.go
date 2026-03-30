package types

import "github.com/cloudfirewall/cloudfirewall/internal/policy/simulate"

type SimulateResponse struct {
	Result simulate.Result `json:"result"`
}
