package headless

import "strings"

// IdentityBlocked reports whether err is an upstream refusal of this serve
// process's identity or egress. Another fingerprint can still succeed. A
// request-shape rejection is not a block.
func IdentityBlocked(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "abusive traffic") ||
		strings.Contains(text, "abuse traffic") ||
		strings.Contains(text, "abuse of traffic") ||
		strings.Contains(text, "freetiererror") ||
		strings.Contains(text, "free usage limit") {
		return true
	}
	return strings.Contains(text, "free models aren't available") &&
		strings.Contains(text, "from your current network")
}
