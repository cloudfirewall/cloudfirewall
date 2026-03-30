package resolve

import "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"

func IsObjectVisible(scope model.ObjectScope, objectEnvironmentID, requestEnvironmentID string) bool {
	if scope == model.ObjectScopeWorkspace {
		return true
	}
	return objectEnvironmentID == requestEnvironmentID
}
