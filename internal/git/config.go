// Package git provides low-level Git operations, including repository access,
// branch operations, commit information, PR operations, and metadata management.
package git

import (
	"time"
)

// GetCurrentDate returns the current date and time in yyyyMMddHHmmss format in UTC
func GetCurrentDate() string {
	now := time.Now().UTC()
	return now.Format("20060102150405")
}
