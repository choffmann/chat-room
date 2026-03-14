package upload

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, testLogger())

	// PNG header
	data := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0}
	relPath, err := s.Save(42, data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !strings.HasPrefix(relPath, "42/") {
		t.Errorf("expected path starting with '42/', got %q", relPath)
	}
	if !strings.HasSuffix(relPath, ".png") {
		t.Errorf("expected .png extension, got %q", relPath)
	}

	absPath := filepath.Join(dir, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	if len(content) != len(data) {
		t.Errorf("expected %d bytes, got %d", len(data), len(content))
	}
}

func TestSave_UnknownType(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, testLogger())

	relPath, err := s.Save(1, []byte("random binary stuff"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// text/plain maps to .bin fallback
	if !strings.HasSuffix(relPath, ".bin") {
		t.Errorf("expected .bin extension for unknown type, got %q", relPath)
	}
}

func TestDeleteRoomDir(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, testLogger())

	_, err := s.Save(7, []byte("hello"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	roomDir := filepath.Join(dir, "7")
	if _, err := os.Stat(roomDir); os.IsNotExist(err) {
		t.Fatal("room dir should exist after save")
	}

	if err := s.DeleteRoomDir(7); err != nil {
		t.Fatalf("DeleteRoomDir failed: %v", err)
	}

	if _, err := os.Stat(roomDir); !os.IsNotExist(err) {
		t.Error("room dir should be deleted")
	}
}

func TestDeleteAll(t *testing.T) {
	dir := t.TempDir()
	baseDir := filepath.Join(dir, "uploads")
	s := NewStore(baseDir, testLogger())

	_, err := s.Save(1, []byte("a"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := s.DeleteAll(); err != nil {
		t.Fatalf("DeleteAll failed: %v", err)
	}

	if _, err := os.Stat(baseDir); !os.IsNotExist(err) {
		t.Error("base dir should be deleted")
	}
}

func TestExtensionFromMIME(t *testing.T) {
	tests := []struct {
		mime string
		ext  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		{"video/mp4", ".mp4"},
		{"something/unknown", ".bin"},
	}

	for _, tt := range tests {
		got := extensionFromMIME(tt.mime)
		if got != tt.ext {
			t.Errorf("extensionFromMIME(%q) = %q, want %q", tt.mime, got, tt.ext)
		}
	}
}

func TestBaseDir(t *testing.T) {
	s := NewStore("/tmp/test-uploads", testLogger())
	if s.BaseDir() != "/tmp/test-uploads" {
		t.Errorf("expected /tmp/test-uploads, got %s", s.BaseDir())
	}
}
