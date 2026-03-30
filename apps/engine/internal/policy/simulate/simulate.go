package simulate

import (
	"fmt"
	"net"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

type Request struct {
	NodeContextID   string          `json:"nodeContextId,omitempty"`
	PolicyVersionID string          `json:"policyVersionId"`
	Direction       model.Direction `json:"direction"`
	SrcIP           string          `json:"srcIp"`
	DstIP           string          `json:"dstIp"`
	Protocol        model.Protocol  `json:"protocol"`
	DstPort         int             `json:"dstPort,omitempty"`
}

type Result struct {
	Verdict                model.Verdict `json:"verdict"`
	MatchedRuleID          string        `json:"matchedRuleId,omitempty"`
	MatchedRuleDescription string        `json:"matchedRuleDescription,omitempty"`
	PolicyVersionID        string        `json:"policyVersionId"`
	Explanation            string        `json:"explanation"`
	UsedDefault            bool          `json:"usedDefault"`
	RemediationSuggestion  string        `json:"remediationSuggestion,omitempty"`
}

type Simulator interface {
	Simulate(policy model.PolicyVersionIR, req Request) (Result, error)
}

type DefaultSimulator struct{}

func New() Simulator {
	return DefaultSimulator{}
}

func (s DefaultSimulator) Simulate(policy model.PolicyVersionIR, req Request) (Result, error) {
	if net.ParseIP(req.SrcIP) == nil || net.ParseIP(req.DstIP) == nil {
		return Result{}, fmt.Errorf("invalid source or destination IP")
	}

	rules := policy.OutboundRules
	defaultVerdict := policy.Defaults.Outbound
	if req.Direction == model.DirectionInbound {
		rules = policy.InboundRules
		defaultVerdict = policy.Defaults.Inbound
	}

	for _, rule := range rules {
		if rule.Enabled && matchRule(policy, rule, req) {
			return Result{
				Verdict:                rule.Action,
				MatchedRuleID:          rule.ID,
				MatchedRuleDescription: rule.Description,
				PolicyVersionID:        policy.PolicyID,
				Explanation:            explainMatch(rule, req),
				UsedDefault:            false,
			}, nil
		}
	}

	return Result{
		Verdict:         defaultVerdict,
		PolicyVersionID: policy.PolicyID,
		Explanation:     explainDefault(req, defaultVerdict),
		UsedDefault:     true,
	}, nil
}
