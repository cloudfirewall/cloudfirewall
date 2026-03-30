package resolve

import "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"

func IsPublicInternet(sel model.NetworkSelector) bool {
	return sel.RefType == model.RefTypePseudo && sel.PseudoType == model.PseudoPublicInternet
}

func IsThisNode(sel model.NetworkSelector) bool {
	return sel.RefType == model.RefTypePseudo && sel.PseudoType == model.PseudoThisNode
}

func IsLoopback(sel model.NetworkSelector) bool {
	return sel.RefType == model.RefTypePseudo && sel.PseudoType == model.PseudoLoopback
}
