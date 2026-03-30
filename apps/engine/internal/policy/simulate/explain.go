package simulate

import (
	"fmt"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

func explainMatch(rule model.RuleIR, req Request) string {
	label := rule.Description
	if label == "" {
		label = rule.ID
	}
	return fmt.Sprintf("%s by %s rule %q (%s)", strings.Title(strings.ToLower(string(rule.Action))), strings.ToLower(string(rule.Direction)), label, rule.ID)
}

func explainDefault(req Request, verdict model.Verdict) string {
	return fmt.Sprintf("%s by default %s policy because no rule matched source %s to %s/%d", strings.Title(strings.ToLower(string(verdict))), strings.ToLower(string(req.Direction)), req.SrcIP, req.Protocol, req.DstPort)
}
