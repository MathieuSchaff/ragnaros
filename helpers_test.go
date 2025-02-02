package main

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

// fakeNvidiaSmi writes a small shell script that stands in for nvidia-smi.
func fakeNvidiaSmi(t *testing.T, body string) string {
	path := filepath.Join(t.TempDir(), "nvidia-smi")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
