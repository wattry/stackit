package output

import (
	"os"
	"path/filepath"
)

// GetLogFilePath returns the path to the log file.
// If STACKIT_LOG_FILE is set, uses that path.
// Otherwise, uses ~/.stackit/logs/stackit.log
func GetLogFilePath() string {
	if customPath := os.Getenv("STACKIT_LOG_FILE"); customPath != "" {
		return customPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home dir
		return "stackit.log"
	}

	return filepath.Join(homeDir, ".stackit", "logs", "stackit.log")
}

// GetPanicLogPath returns the path to the panic log file.
// This is a dedicated file for panic/crash information that is
// separate from regular logs to ensure panics are always findable.
// Uses ~/.stackit/logs/panic.log
func GetPanicLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home dir
		return "panic.log"
	}

	return filepath.Join(homeDir, ".stackit", "logs", "panic.log")
}
