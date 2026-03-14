package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile(t *testing.T) {
	t.Run("creates file with correct content", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "testfile")
		content := []byte("hello world")

		if err := WriteFile(content, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("got %q, want %q", got, content)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "a", "b", "c", "testfile")

		if err := WriteFile([]byte("nested"), path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file does not exist: %v", err)
		}
	})

	t.Run("file has executable permissions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "binary")

		if err := WriteFile([]byte("data"), path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("expected executable permissions, got %v", info.Mode())
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")

		if err := WriteFile([]byte("original"), path); err != nil {
			t.Fatalf("unexpected error on first write: %v", err)
		}
		if err := WriteFile([]byte("updated"), path); err != nil {
			t.Fatalf("unexpected error on second write: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(got) != "updated" {
			t.Errorf("got %q, want %q", got, "updated")
		}
	})
}
