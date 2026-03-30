package types

type ValidationResponse struct {
	Result ValidationResult `json:"result"`
}

type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationMessage `json:"errors,omitempty"`
	Warnings []ValidationMessage `json:"warnings,omitempty"`
}

type ValidationMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
