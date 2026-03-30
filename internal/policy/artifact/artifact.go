package artifact

import (
	"time"

	compilepkg "github.com/cloudfirewall/cloudfirewall/internal/policy/compile"
	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

type CompiledPolicyArtifact struct {
	PolicyID        string   `json:"policyId"`
	PolicyVersionID string   `json:"policyVersionId"`
	GeneratedAt     string   `json:"generatedAt"`
	CompilerVersion string   `json:"compilerVersion"`
	Family          string   `json:"family"`
	Mode            string   `json:"mode"`
	Warnings        []string `json:"warnings"`
	ContentHash     string   `json:"contentHash"`
	Content         string   `json:"content"`
}

func NewFromCompilation(authoring model.AuthoringPolicyModel, ir model.PolicyVersionIR, compiled compilepkg.Result, compilerVersion string) CompiledPolicyArtifact {
	return CompiledPolicyArtifact{
		PolicyID:        defaultString(authoring.PolicyID, ir.PolicyID),
		PolicyVersionID: ir.PolicyID,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		CompilerVersion: compilerVersion,
		Family:          string(ir.Mode.Family),
		Mode:            string(ir.Mode.Type),
		Warnings:        compiled.Warnings,
		ContentHash:     compiled.ContentHash,
		Content:         compiled.Content,
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
