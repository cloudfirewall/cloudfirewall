package compile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

func renderInputChain(policy model.PolicyVersionIR) string {
	var b strings.Builder
	b.WriteString("  chain input {\n")
	b.WriteString("    type filter hook input priority 0; policy accept;\n")
	if policy.SystemRules.AllowLoopback {
		b.WriteString("    iifname \"lo\" accept comment \"cfw:system=loopback\"\n")
	}
	if policy.SystemRules.AllowEstablishedRelated {
		b.WriteString("    ct state established,related accept comment \"cfw:system=established-related\"\n")
	}
	rules := append([]model.RuleIR(nil), policy.InboundRules...)
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].OrderIndex == rules[j].OrderIndex {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].OrderIndex < rules[j].OrderIndex
	})
	for _, rule := range rules {
		if rule.Enabled {
			b.WriteString(renderRule(rule, policy, true))
		}
	}
	b.WriteString(fmt.Sprintf("    %s comment \"cfw:default=inbound\"\n", strings.ToLower(string(policy.Defaults.Inbound))))
	b.WriteString("  }\n")
	return b.String()
}

func renderOutputChain(policy model.PolicyVersionIR) string {
	var b strings.Builder
	b.WriteString("  chain output {\n")
	b.WriteString("    type filter hook output priority 0; policy accept;\n")
	if policy.SystemRules.AllowLoopback {
		b.WriteString("    oifname \"lo\" accept comment \"cfw:system=loopback\"\n")
	}
	if policy.SystemRules.AllowEstablishedRelated {
		b.WriteString("    ct state established,related accept comment \"cfw:system=established-related\"\n")
	}
	rules := append([]model.RuleIR(nil), policy.OutboundRules...)
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].OrderIndex == rules[j].OrderIndex {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].OrderIndex < rules[j].OrderIndex
	})
	for _, rule := range rules {
		if rule.Enabled {
			b.WriteString(renderRule(rule, policy, false))
		}
	}
	b.WriteString(fmt.Sprintf("    %s comment \"cfw:default=outbound\"\n", strings.ToLower(string(policy.Defaults.Outbound))))
	b.WriteString("  }\n")
	return b.String()
}
