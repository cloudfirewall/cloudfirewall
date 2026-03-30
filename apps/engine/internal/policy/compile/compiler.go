package compile

import (
	platformhash "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/platform/hash"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

type Result struct {
	Content     string   `json:"content"`
	ContentHash string   `json:"contentHash"`
	Warnings    []string `json:"warnings"`
}

type Compiler interface {
	Compile(policy model.PolicyVersionIR) (Result, error)
}

type NftCompiler struct{}

func New() Compiler {
	return NftCompiler{}
}

func (c NftCompiler) Compile(policy model.PolicyVersionIR) (Result, error) {
	content := renderPolicy(policy)
	return Result{
		Content:     content,
		ContentHash: platformhash.SHA256String(content),
	}, nil
}
