package x402

import (
	"fmt"
	"time"
)

// ValidateWindow returns an error if the authorization window is too large
// (> 3600s), inverted, or already expired (validBefore <= now).
func ValidateWindow(validAfter, validBefore int64) error {
	window := validBefore - validAfter
	if window > 3600 {
		return fmt.Errorf("payment window too large: %d seconds (max 3600)", window)
	}
	if window <= 0 {
		return fmt.Errorf("payment window is invalid: validBefore (%d) must be > validAfter (%d)", validBefore, validAfter)
	}
	now := time.Now().Unix()
	if validBefore <= now {
		return fmt.Errorf("authorization expired: validBefore (%d) is not in the future (now: %d)", validBefore, now)
	}
	return nil
}
