package artifact

import platformhash "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/platform/hash"

func HashContent(content string) string {
	return platformhash.SHA256String(content)
}
