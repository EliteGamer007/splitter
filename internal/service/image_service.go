package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// ReadAndValidateImage validates and reads an uploaded image into memory.
func ReadAndValidateImage(fileHeader *multipart.FileHeader, maxSize int64) ([]byte, string, error) {
	if fileHeader == nil {
		return nil, "", fmt.Errorf("missing file")
	}

	if fileHeader.Size > maxSize {
		return nil, "", fmt.Errorf("file size exceeds %d bytes limit", maxSize)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read uploaded file: %w", err)
	}

	if int64(len(fileBytes)) > maxSize {
		return nil, "", fmt.Errorf("file size exceeds %d bytes limit", maxSize)
	}

	contentType := http.DetectContentType(fileBytes)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}

	if !allowedTypes[contentType] {
		return nil, "", fmt.Errorf("invalid file type: %s. Only jpeg, png, and gif are allowed", contentType)
	}

	return fileBytes, contentType, nil
}
