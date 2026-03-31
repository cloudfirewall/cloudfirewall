package policybuilder

import (
	"errors"
	"fmt"

	compilepkg "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/compile"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/normalize"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/resolve"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/validate"
)

type PeerType string

const (
	PeerTypePublicInternet PeerType = "PUBLIC_INTERNET"
	PeerTypeOfficeIPs      PeerType = "OFFICE_IPS"
	PeerTypeThisNode       PeerType = "THIS_NODE"
	PeerTypeCIDR           PeerType = "CIDR"
)

type RuleDraft struct {
	ID          string   `json:"id"`
	Direction   string   `json:"direction"`
	Action      string   `json:"action"`
	PeerType    PeerType `json:"peerType"`
	PeerValue   string   `json:"peerValue,omitempty"`
	Protocol    string   `json:"protocol"`
	Ports       []int    `json:"ports"`
	LogEnabled  bool     `json:"logEnabled"`
	Enabled     bool     `json:"enabled"`
	OrderIndex  int      `json:"orderIndex"`
	Description string   `json:"description,omitempty"`
}

type PolicyDraft struct {
	PolicyID              string      `json:"policyId,omitempty"`
	VersionNumber         int         `json:"versionNumber,omitempty"`
	EnvironmentID         string      `json:"environmentId,omitempty"`
	Name                  string      `json:"name"`
	Description           string      `json:"description,omitempty"`
	DefaultInboundAction  string      `json:"defaultInboundAction"`
	DefaultOutboundAction string      `json:"defaultOutboundAction"`
	AllowLoopback         bool        `json:"allowLoopback"`
	AllowEstablished      bool        `json:"allowEstablishedRelated"`
	Rules                 []RuleDraft `json:"rules"`
}

type CompiledPolicy struct {
	Policy   model.AuthoringPolicyModel `json:"policy"`
	Content  string                     `json:"content"`
	Version  string                     `json:"version"`
	Warnings []string                   `json:"warnings"`
}

func CompileDraft(draft PolicyDraft) (CompiledPolicy, error) {
	authoring := toAuthoringPolicy(draft)
	norm, err := normalize.New().Normalize(authoring)
	if err != nil {
		return CompiledPolicy{}, err
	}

	resolved := resolve.New().Resolve(norm, resolve.ResolutionContext{
		EnvironmentID: norm.EnvironmentID,
		VisibleNetworks: []model.ResolvedNetworkObject{
			{
				ID:   "obj-office",
				Name: "office-ips",
				Kind: model.NetworkObjectList,
				Values: []model.NormalizedNetworkValue{
					{Family: model.IPFamilyV4, Value: "203.0.113.0/24"},
				},
			},
		},
	})
	if len(resolved.Errors) > 0 {
		return CompiledPolicy{}, errors.New(resolved.Errors[0])
	}

	structural := validate.NewStructural().Validate(resolved.Policy)
	semantic := validate.NewSemantic().Validate(resolved.Policy)
	merged := validate.Merge(structural, semantic)
	if !merged.Valid {
		return CompiledPolicy{}, errors.New(merged.Errors[0].Message)
	}

	compiled, err := compilepkg.New().Compile(resolved.Policy)
	if err != nil {
		return CompiledPolicy{}, err
	}

	warnings := make([]string, 0, len(merged.Warnings))
	for _, warning := range merged.Warnings {
		warnings = append(warnings, warning.Message)
	}

	return CompiledPolicy{
		Policy:   authoring,
		Content:  compiled.Content,
		Version:  compiled.ContentHash,
		Warnings: warnings,
	}, nil
}

func toAuthoringPolicy(draft PolicyDraft) model.AuthoringPolicyModel {
	policyID := draft.PolicyID
	if policyID == "" {
		policyID = "policy-" + draft.Name
	}
	environmentID := draft.EnvironmentID
	if environmentID == "" {
		environmentID = "env-default"
	}

	rules := make([]model.AuthoringRule, 0, len(draft.Rules))
	for index, rule := range draft.Rules {
		orderIndex := rule.OrderIndex
		if orderIndex == 0 {
			orderIndex = (index + 1) * 10
		}
		rules = append(rules, model.AuthoringRule{
			ID:          defaultString(rule.ID, fmt.Sprintf("rule-%d", index+1)),
			Direction:   model.Direction(rule.Direction),
			Action:      model.Verdict(rule.Action),
			Source:      sourceRef(rule),
			Destination: destinationRef(rule),
			Service: model.AuthoringRuleRef{
				Type:     model.RefTypeLiteral,
				Protocol: model.Protocol(rule.Protocol),
				Ports:    rule.Ports,
			},
			LogEnabled:  rule.LogEnabled,
			Enabled:     rule.Enabled,
			OrderIndex:  orderIndex,
			Description: rule.Description,
		})
	}

	return model.AuthoringPolicyModel{
		PolicyID:              policyID,
		VersionNumber:         max(1, draft.VersionNumber),
		EnvironmentID:         environmentID,
		Name:                  draft.Name,
		Description:           draft.Description,
		DefaultInboundAction:  model.Verdict(draft.DefaultInboundAction),
		DefaultOutboundAction: model.Verdict(draft.DefaultOutboundAction),
		SystemRules: model.SystemRuleOptions{
			AllowLoopback:           draft.AllowLoopback,
			AllowEstablishedRelated: draft.AllowEstablished,
		},
		Rules: rules,
	}
}

func sourceRef(rule RuleDraft) model.AuthoringRuleRef {
	if model.Direction(rule.Direction) == model.DirectionOutbound {
		return model.AuthoringRuleRef{Type: model.RefTypePseudo, PseudoType: model.PseudoThisNode}
	}
	return peerRef(rule)
}

func destinationRef(rule RuleDraft) model.AuthoringRuleRef {
	if model.Direction(rule.Direction) == model.DirectionOutbound {
		return peerRef(rule)
	}
	return model.AuthoringRuleRef{Type: model.RefTypePseudo, PseudoType: model.PseudoThisNode}
}

func peerRef(rule RuleDraft) model.AuthoringRuleRef {
	switch rule.PeerType {
	case PeerTypePublicInternet:
		return model.AuthoringRuleRef{Type: model.RefTypePseudo, PseudoType: model.PseudoPublicInternet}
	case PeerTypeThisNode:
		return model.AuthoringRuleRef{Type: model.RefTypePseudo, PseudoType: model.PseudoThisNode}
	case PeerTypeOfficeIPs:
		return model.AuthoringRuleRef{Type: model.RefTypeObject, ObjectName: "office-ips"}
	case PeerTypeCIDR:
		return model.AuthoringRuleRef{Type: model.RefTypeLiteral, Literal: rule.PeerValue}
	default:
		return model.AuthoringRuleRef{Type: model.RefTypePseudo, PseudoType: model.PseudoPublicInternet}
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
