package auth

import "crypto/subtle"

// ValidAPIKey compares a presented key against configured non-empty keys.
func ValidAPIKey(input string, keys []string) bool {
	if input == "" {
		return false
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(input), []byte(key)) == 1 {
			return true
		}
	}
	return false
}
