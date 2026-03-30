package types

import "github.com/cloudfirewall/cloudfirewall/internal/policy/validate"

type ValidationResponse struct {
	Result validate.Result `json:"result"`
}
