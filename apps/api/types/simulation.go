package types

type SimulateResponse struct {
	Result SimulationResult `json:"result"`
}

type SimulationResult struct {
	Verdict                string `json:"verdict"`
	MatchedRuleID          string `json:"matchedRuleId,omitempty"`
	MatchedRuleDescription string `json:"matchedRuleDescription,omitempty"`
	PolicyVersionID        string `json:"policyVersionId"`
	Explanation            string `json:"explanation"`
	UsedDefault            bool   `json:"usedDefault"`
	RemediationSuggestion  string `json:"remediationSuggestion,omitempty"`
}
