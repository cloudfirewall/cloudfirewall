package simulate

import (
	"net"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

func matchRule(policy model.PolicyVersionIR, rule model.RuleIR, req Request) bool {
	return rule.Direction == req.Direction &&
		matchNetworkSelector(policy, rule.Source, req.SrcIP) &&
		matchDestinationSelector(policy, rule.Destination, req.DstIP) &&
		matchServiceSelector(policy, rule.Service, req.Protocol, req.DstPort)
}

func matchNetworkSelector(policy model.PolicyVersionIR, sel model.NetworkSelector, ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	switch sel.RefType {
	case model.RefTypePseudo:
		return sel.PseudoType == model.PseudoPublicInternet || sel.PseudoType == model.PseudoThisNode
	case model.RefTypeLiteral:
		return matchValues(sel.LiteralValues, ip)
	case model.RefTypeObject:
		obj, ok := policy.Objects.Networks[sel.ObjectName]
		if !ok {
			return false
		}
		return matchValues(obj.Values, ip)
	default:
		return false
	}
}

func matchDestinationSelector(policy model.PolicyVersionIR, sel model.NetworkSelector, ipStr string) bool {
	return matchNetworkSelector(policy, sel, ipStr)
}

func matchValues(values []model.NormalizedNetworkValue, ip net.IP) bool {
	for _, v := range values {
		if literal := net.ParseIP(v.Value); literal != nil && literal.Equal(ip) {
			return true
		}
		if _, cidr, err := net.ParseCIDR(v.Value); err == nil && cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func matchServiceSelector(policy model.PolicyVersionIR, sel model.ServiceSelector, proto model.Protocol, port int) bool {
	var entries []model.ServiceEntry
	switch sel.RefType {
	case model.RefTypeLiteral:
		entries = sel.Entries
	case model.RefTypeObject:
		obj, ok := policy.Objects.Services[sel.ObjectName]
		if !ok {
			return false
		}
		entries = obj.Entries
	default:
		return false
	}
	for _, e := range entries {
		if e.Protocol == proto && e.Port == port {
			return true
		}
	}
	return false
}
