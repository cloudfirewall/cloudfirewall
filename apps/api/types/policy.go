package types

type ValidatePolicyResponse struct {
	Policy map[string]any `json:"policy"`
	Valid  bool           `json:"valid"`
}
