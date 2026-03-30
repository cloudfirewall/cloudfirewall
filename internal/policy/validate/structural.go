package validate

import (
	"fmt"
	"net"

	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

type StructuralValidator interface {
	Validate(policy model.PolicyVersionIR) Result
}

type DefaultStructuralValidator struct{}

func NewStructural() StructuralValidator {
	return DefaultStructuralValidator{}
}

func (v DefaultStructuralValidator) Validate(policy model.PolicyVersionIR) Result {
	res := NewResult()
	if policy.Defaults.Inbound == "" || policy.Defaults.Outbound == "" {
		res.Errors = append(res.Errors, Message{Code: CodeMissingDefaults, Message: "policy defaults are required"})
	}

	seenIDs := map[string]struct{}{}
	seenOrder := map[model.Direction]map[int]struct{}{
		model.DirectionInbound:  {},
		model.DirectionOutbound: {},
	}

	check := func(rules []model.RuleIR, dir model.Direction) {
		for _, rule := range rules {
			if _, ok := seenIDs[rule.ID]; ok {
				res.Errors = append(res.Errors, Message{Code: CodeDuplicateRuleID, Field: rule.ID, Message: fmt.Sprintf("duplicate rule id: %s", rule.ID)})
			} else {
				seenIDs[rule.ID] = struct{}{}
			}
			if _, ok := seenOrder[dir][rule.OrderIndex]; ok {
				res.Errors = append(res.Errors, Message{Code: CodeDuplicateOrderIndex, Field: rule.ID, Message: fmt.Sprintf("duplicate order index %d", rule.OrderIndex)})
			} else {
				seenOrder[dir][rule.OrderIndex] = struct{}{}
			}
			for _, sel := range []model.NetworkSelector{rule.Source, rule.Destination} {
				for _, lit := range sel.LiteralValues {
					if ip := net.ParseIP(lit.Value); ip == nil {
						if _, _, err := net.ParseCIDR(lit.Value); err != nil {
							res.Errors = append(res.Errors, Message{Code: CodeInvalidCIDR, Field: rule.ID, Message: fmt.Sprintf("invalid network literal: %s", lit.Value)})
						}
					}
				}
			}
			for _, svc := range rule.Service.Entries {
				if svc.Port < 1 || svc.Port > 65535 {
					res.Errors = append(res.Errors, Message{Code: CodeInvalidPort, Field: rule.ID, Message: fmt.Sprintf("invalid port: %d", svc.Port)})
				}
			}
		}
	}

	check(policy.InboundRules, model.DirectionInbound)
	check(policy.OutboundRules, model.DirectionOutbound)
	res.Valid = len(res.Errors) == 0
	return res
}
