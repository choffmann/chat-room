package upload

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Store struct {
	baseDir string
	logger  *slog.Logger
}

func NewStore(baseDir string, logger *slog.Logger) *Store {
	return &Store{baseDir: baseDir, logger: logger}
}

func (s *Store) BaseDir() string { return s.baseDir }

// Save writes data to <baseDir>/<roomID>/<uuid>.<ext> and returns the relative path.
func (s *Store) Save(roomID uint, data []byte) (string, error) {
	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", roomID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create room upload dir: %w", err)
	}

	ext := extensionFromMIME(http.DetectContentType(data))
	name := uuid.New().String() + ext
	relPath := filepath.Join(fmt.Sprintf("%d", roomID), name)
	absPath := filepath.Join(s.baseDir, relPath)

	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write upload file: %w", err)
	}

	s.logger.Debug("file saved", "path", relPath, "size", len(data))
	return relPath, nil
}

func (s *Store) DeleteRoomDir(roomID uint) error {
	dir := filepath.Join(s.baseDir, fmt.Sprintf("%d", roomID))
	s.logger.Debug("deleting room upload dir", "dir", dir)
	return os.RemoveAll(dir)
}

func (s *Store) DeleteAll() error {
	s.logger.Debug("deleting all uploads", "dir", s.baseDir)
	return os.RemoveAll(s.baseDir)
}

func extensionFromMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/gzip":
		return ".gz"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ".bin"
	}
}
