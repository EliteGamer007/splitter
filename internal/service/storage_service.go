package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	Save(file *multipart.FileHeader) (string, string, error) // Returns (relativeURL, mediaType, error)
	Delete(path string) error
}

// LocalStorage implements FileStorage for local disk storage
type LocalStorage struct {
	BaseDir string
	BaseURL string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{
		BaseDir: baseDir,
		BaseURL: baseURL,
	}
}

// Save saves a file to local storage with security checks
func (s *LocalStorage) Save(fileHeader *multipart.FileHeader) (string, string, error) {
	// 1. Validate File Size (double check, though usually handled by middleware/handler limits)
	if fileHeader.Size > 5*1024*1024 { // 5MB
		return "", "", fmt.Errorf("file size exceeds 5MB limit")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// 2. Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return "", "", fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset file pointer
	if _, err := src.Seek(0, 0); err != nil {
		return "", "", fmt.Errorf("failed to seek file: %w", err)
	}

	// 3. Validate MIME type
	contentType := http.DetectContentType(buffer)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}

	if !allowedTypes[contentType] {
		return "", "", fmt.Errorf("invalid file type: %s. Only jpeg, png, and gif are allowed", contentType)
	}

	// 4. Generate Safe Filename
	// Use UUID to prevent collisions and directory traversal attacks
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		// Fallback extension based on content type
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		}
	}

	newFilename := uuid.New().String() + ext

	// Ensure upload directory exists
	uploadPath := filepath.Join(s.BaseDir, "uploads")
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 5. Create Destination File
	destPath := filepath.Join(uploadPath, newFilename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 6. Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("failed to save file content: %w", err)
	}

	// Return relative URL and detected request content type
	// Note: We return the specific type like "image/jpeg"
	relativeURL := fmt.Sprintf("/uploads/%s", newFilename)
	return relativeURL, contentType, nil
}

// Delete removes a file from local storage
func (s *LocalStorage) Delete(path string) error {
	// Sanitize path to prevent traversal
	// In this simple implementation, we assume path is just the filename or relative path
	// For better security, we should validate it strictly

	filename := filepath.Base(path)
	fullPath := filepath.Join(s.BaseDir, "uploads", filename)

	return os.Remove(fullPath)
}
