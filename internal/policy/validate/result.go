package validate

type Message struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type Result struct {
	Valid    bool      `json:"valid"`
	Errors   []Message `json:"errors"`
	Warnings []Message `json:"warnings"`
}

func NewResult() Result {
	return Result{Valid: true}
}

func Merge(parts ...Result) Result {
	out := NewResult()
	for _, p := range parts {
		out.Errors = append(out.Errors, p.Errors...)
		out.Warnings = append(out.Warnings, p.Warnings...)
	}
	out.Valid = len(out.Errors) == 0
	return out
}
