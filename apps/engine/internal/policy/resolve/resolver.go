package resolve

import (
	"fmt"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/normalize"
)

type ResolutionContext struct {
	WorkspaceID     string
	EnvironmentID   string
	VisibleNetworks []model.ResolvedNetworkObject
	VisibleServices []model.ResolvedServiceObject
	NodeContext     *NodeContext
}

type NodeContext struct {
	NodeID    string
	Addresses []model.NormalizedNetworkValue
}

type Result struct {
	Policy   model.PolicyVersionIR
	Errors   []string
	Warnings []string
}

type Resolver interface {
	Resolve(in model.AuthoringPolicyModel, ctx ResolutionContext) Result
}

type DefaultResolver struct{}

func New() Resolver {
	return DefaultResolver{}
}

func (r DefaultResolver) Resolve(in model.AuthoringPolicyModel, ctx ResolutionContext) Result {
	res := Result{
		Policy: model.PolicyVersionIR{
			PolicyID:      defaultString(in.PolicyID, "policy-0001"),
			VersionNumber: max(1, in.VersionNumber),
			EnvironmentID: in.EnvironmentID,
			Metadata: model.PolicyMetadata{
				Name:        in.Name,
				Description: in.Description,
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			},
			Mode: model.PolicyMode{
				Type:   model.PolicyModeHostFiltering,
				Family: model.FamilyINET,
			},
			Defaults: model.PolicyDefaults{
				Inbound:  in.DefaultInboundAction,
				Outbound: in.DefaultOutboundAction,
			},
			SystemRules: in.SystemRules,
			Objects: model.ResolvedObjectCatalog{
				Networks: map[string]model.ResolvedNetworkObject{},
				Services: map[string]model.ResolvedServiceObject{},
			},
		},
	}

	networksByName := map[string]model.ResolvedNetworkObject{}
	for _, n := range ctx.VisibleNetworks {
		networksByName[n.Name] = n
		res.Policy.Objects.Networks[n.Name] = n
	}
	servicesByName := map[string]model.ResolvedServiceObject{}
	for _, s := range ctx.VisibleServices {
		servicesByName[s.Name] = s
		res.Policy.Objects.Services[s.Name] = s
	}

	for _, rule := range in.Rules {
		ruleIR := model.RuleIR{
			ID:          rule.ID,
			Direction:   rule.Direction,
			Action:      rule.Action,
			LogEnabled:  rule.LogEnabled,
			Enabled:     rule.Enabled,
			OrderIndex:  rule.OrderIndex,
			Description: rule.Description,
		}
		ruleIR.Source = resolveNetwork(rule.Source, networksByName, &res)
		ruleIR.Destination = resolveNetwork(rule.Destination, networksByName, &res)
		ruleIR.Service = resolveService(rule.Service, servicesByName, &res)

		if ruleIR.Direction == model.DirectionInbound {
			res.Policy.InboundRules = append(res.Policy.InboundRules, ruleIR)
		} else {
			res.Policy.OutboundRules = append(res.Policy.OutboundRules, ruleIR)
		}
	}

	return res
}

func resolveNetwork(ref model.AuthoringRuleRef, visible map[string]model.ResolvedNetworkObject, res *Result) model.NetworkSelector {
	sel := model.NetworkSelector{
		RefType:    ref.Type,
		ObjectID:   ref.ObjectID,
		ObjectName: ref.ObjectName,
		PseudoType: ref.PseudoType,
	}
	switch ref.Type {
	case model.RefTypeObject:
		obj, ok := visible[ref.ObjectName]
		if !ok {
			res.Errors = append(res.Errors, fmt.Sprintf("unknown network object: %s", ref.ObjectName))
			return sel
		}
		sel.ObjectID = obj.ID
		sel.ObjectName = obj.Name
	case model.RefTypeLiteral:
		for _, raw := range append(oneOrZero(ref.Literal), ref.Literals...) {
			if v, ok := normalize.NormalizeNetworkLiteral(raw); ok {
				sel.LiteralValues = append(sel.LiteralValues, v)
			} else {
				res.Errors = append(res.Errors, fmt.Sprintf("invalid network literal: %s", raw))
			}
		}
	case model.RefTypePseudo:
	default:
		res.Errors = append(res.Errors, fmt.Sprintf("unsupported network ref type: %s", ref.Type))
	}
	return sel
}

func resolveService(ref model.AuthoringRuleRef, visible map[string]model.ResolvedServiceObject, res *Result) model.ServiceSelector {
	sel := model.ServiceSelector{
		RefType:    ref.Type,
		ObjectID:   ref.ObjectID,
		ObjectName: ref.ObjectName,
	}
	switch ref.Type {
	case model.RefTypeObject:
		obj, ok := visible[ref.ObjectName]
		if !ok {
			res.Errors = append(res.Errors, fmt.Sprintf("unknown service object: %s", ref.ObjectName))
			return sel
		}
		sel.ObjectID = obj.ID
		sel.ObjectName = obj.Name
	case model.RefTypeLiteral:
		for _, port := range ref.Ports {
			sel.Entries = append(sel.Entries, model.ServiceEntry{Protocol: ref.Protocol, Port: port})
		}
		normalize.SortServiceEntries(sel.Entries)
	default:
		res.Errors = append(res.Errors, fmt.Sprintf("unsupported service ref type: %s", ref.Type))
	}
	return sel
}

func oneOrZero(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func defaultString(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
