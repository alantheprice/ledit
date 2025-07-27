package utils

import (
	"crypto/sha1"
	"crypto/sha256" // New import for SHA256
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath" // Added for filepath.Dir
	"strings"
)

// GenerateRequestHash generates a SHA1 hash of the given instructions.
func GenerateRequestHash(instructions string) string {
	hasher := sha1.New()
	hasher.Write([]byte(instructions))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GenerateFileRevisionHash generates a SHA1 hash of the filename and code.
func GenerateFileRevisionHash(filename, code string) string {
	hasher := sha1.New()
	hasher.Write([]byte(filename + code))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GenerateFileHash creates a SHA256 hash of the file content.
func GenerateFileHash(content string) string {
	hasher := sha256.New()
	hasher.Write([]byte(content))
	return hex.EncodeToString(hasher.Sum(nil))
}

// SaveFile saves the given content to a file. If content is empty, it removes the file.
func SaveFile(filename, content string) error {
	if content == "" {
		if _, err := os.Stat(filename); err == nil {
			return os.Remove(filename)
		}
		return nil // File doesn't exist, nothing to do
	}

	dir := filepath.Dir(filename)
	if dir != "" {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("could not create directory %s: %w", dir, err)
		}
	}

	return os.WriteFile(filename, []byte(content), 0644)
}

func ReadFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// TrimSpaceAndNewlines trims leading/trailing whitespace and newlines from a string.
func TrimSpaceAndNewlines(s string) string {
	return strings.TrimSpace(s)
}
