package types

import "github.com/cloudfirewall/cloudfirewall/internal/policy/model"

type ValidatePolicyResponse struct {
	Policy model.PolicyVersionIR `json:"policy"`
	Valid  bool                  `json:"valid"`
}
