package hash

import (
	"crypto/sha256"
	"fmt"
)

func SHA256String(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("sha256:%x", sum[:])
}
