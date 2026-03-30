package compile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

func renderPolicy(policy model.PolicyVersionIR) string {
	var b strings.Builder
	b.WriteString(renderHeader(policy))
	b.WriteString("table inet cloudfirewall {\n")
	b.WriteString(renderInputChain(policy))
	b.WriteString(renderOutputChain(policy))
	b.WriteString("}\n")
	return b.String()
}

func renderRule(rule model.RuleIR, policy model.PolicyVersionIR, inbound bool) string {
	parts := []string{}
	if inbound {
		parts = append(parts, renderNetworkMatch("ip saddr", rule.Source, policy))
	} else {
		parts = append(parts, renderNetworkMatch("ip daddr", rule.Destination, policy))
	}
	parts = append(parts, renderServiceMatch(rule.Service, policy))
	parts = append(parts, strings.ToLower(string(rule.Action)))
	parts = append(parts, fmt.Sprintf("comment \"cfw:rule=%s;policy=%s;version=%d\"", rule.ID, policy.PolicyID, policy.VersionNumber))
	return "    " + strings.Join(filterEmpty(parts), " ") + "\n"
}

func renderNetworkMatch(prefix string, sel model.NetworkSelector, policy model.PolicyVersionIR) string {
	switch sel.RefType {
	case model.RefTypePseudo:
		if sel.PseudoType == model.PseudoPublicInternet || sel.PseudoType == model.PseudoThisNode {
			return ""
		}
	case model.RefTypeLiteral:
		vals := make([]string, 0, len(sel.LiteralValues))
		for _, v := range sel.LiteralValues {
			vals = append(vals, v.Value)
		}
		sort.Strings(vals)
		if len(vals) == 1 {
			return prefix + " " + vals[0]
		}
		return prefix + " { " + strings.Join(vals, ", ") + " }"
	case model.RefTypeObject:
		obj, ok := policy.Objects.Networks[sel.ObjectName]
		if !ok {
			return ""
		}
		vals := make([]string, 0, len(obj.Values))
		for _, v := range obj.Values {
			vals = append(vals, v.Value)
		}
		sort.Strings(vals)
		if len(vals) == 1 {
			return prefix + " " + vals[0]
		}
		return prefix + " { " + strings.Join(vals, ", ") + " }"
	}
	return ""
}

func renderServiceMatch(sel model.ServiceSelector, policy model.PolicyVersionIR) string {
	entries := []model.ServiceEntry{}
	switch sel.RefType {
	case model.RefTypeLiteral:
		entries = sel.Entries
	case model.RefTypeObject:
		if obj, ok := policy.Objects.Services[sel.ObjectName]; ok {
			entries = obj.Entries
		}
	}
	if len(entries) == 0 {
		return ""
	}
	ports := make([]string, 0, len(entries))
	proto := strings.ToLower(string(entries[0].Protocol))
	for _, e := range entries {
		ports = append(ports, fmt.Sprintf("%d", e.Port))
	}
	if len(ports) == 1 {
		return fmt.Sprintf("%s dport %s", proto, ports[0])
	}
	return fmt.Sprintf("%s dport { %s }", proto, strings.Join(ports, ", "))
}

func filterEmpty(parts []string) []string {
	out := []string{}
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}
