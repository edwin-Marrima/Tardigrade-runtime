package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data to path with executable permissions,
// creating parent directories as needed.
func WriteFile(data []byte, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(path, data, 0755)
}
