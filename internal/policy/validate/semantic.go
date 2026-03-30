package validate

import (
	"fmt"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

type SemanticValidator interface {
	Validate(policy model.PolicyVersionIR) Result
}

type DefaultSemanticValidator struct{}

func NewSemantic() SemanticValidator {
	return DefaultSemanticValidator{}
}

func (v DefaultSemanticValidator) Validate(policy model.PolicyVersionIR) Result {
	res := NewResult()
	for _, rule := range policy.InboundRules {
		if !rule.Enabled || rule.Action != model.VerdictAllow {
			continue
		}
		isPublic := rule.Source.RefType == model.RefTypePseudo && rule.Source.PseudoType == model.PseudoPublicInternet
		for _, entry := range serviceEntries(policy, rule.Service) {
			if isPublic && entry.Protocol == model.ProtocolTCP && entry.Port == 22 {
				res.Warnings = append(res.Warnings, Message{Code: CodePublicSSHExposure, Field: rule.ID, Message: "SSH is reachable from public internet"})
			}
			if isPublic && entry.Protocol == model.ProtocolTCP && isDatabasePort(entry.Port) {
				res.Warnings = append(res.Warnings, Message{Code: CodePublicDatabaseExposure, Field: rule.ID, Message: fmt.Sprintf("database port %d is reachable from public internet", entry.Port)})
			}
		}
		if isPublic && isAnyService(policy, rule.Service) {
			res.Warnings = append(res.Warnings, Message{Code: CodeAllowAllInbound, Field: rule.ID, Message: "allow-all inbound rule detected"})
		}
	}
	res.Valid = len(res.Errors) == 0
	return res
}

func serviceEntries(policy model.PolicyVersionIR, sel model.ServiceSelector) []model.ServiceEntry {
	if sel.RefType == model.RefTypeLiteral {
		return sel.Entries
	}
	if sel.RefType == model.RefTypeObject {
		if obj, ok := policy.Objects.Services[sel.ObjectName]; ok {
			return obj.Entries
		}
	}
	return nil
}

func isAnyService(policy model.PolicyVersionIR, sel model.ServiceSelector) bool {
	entries := serviceEntries(policy, sel)
	if len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if strings.ToUpper(string(e.Protocol)) == string(model.ProtocolAny) {
			return true
		}
	}
	return false
}

func isDatabasePort(port int) bool {
	switch port {
	case 3306, 5432, 1433, 1521, 27017:
		return true
	default:
		return false
	}
}
