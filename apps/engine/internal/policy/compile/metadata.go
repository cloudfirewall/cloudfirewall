package compile

import (
	"fmt"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

func renderHeader(policy model.PolicyVersionIR) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# cloudfirewall generated artifact\n"))
	b.WriteString(fmt.Sprintf("# policy_id=%s\n", policy.PolicyID))
	b.WriteString(fmt.Sprintf("# version=%d\n", policy.VersionNumber))
	b.WriteString(fmt.Sprintf("# name=%s\n\n", policy.Metadata.Name))
	return b.String()
}
