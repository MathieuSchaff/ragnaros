package system

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFiles creates fake /sys files: path to content.
func writeFiles(t *testing.T, files map[string]string) {
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
