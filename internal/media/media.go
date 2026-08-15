package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const MaxImageBytes = 5 << 20 // 5 MiB

var (
	ErrNotConfigured = errors.New("imagekit_not_configured")
	ErrInvalidType   = errors.New("invalid_image_type")
	ErrTooLarge      = errors.New("image_too_large")
)

// UploadInput is a single file upload request.
type UploadInput struct {
	Folder      string
	FileName    string
	ContentType string
	Body        io.Reader
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	URL    string
	FileID string
}

// Uploader stores image bytes and returns a public CDN URL.
type Uploader interface {
	Upload(ctx context.Context, in UploadInput) (UploadResult, error)
}

func FolderBrand(brandID string) string {
	return path.Join("/scaloo/brands", brandID)
}

func FolderAvatar(userID string) string {
	return path.Join("/scaloo/avatars", userID)
}

func FolderCampaignContent(campaignID string) string {
	return path.Join("/scaloo/campaigns", campaignID, "content")
}

// ExtForContentType maps allowed image MIME types to file extensions.
func ExtForContentType(ct string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/jpeg", "image/jpg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", ErrInvalidType
	}
}

// ValidateContentType ensures ct is an allowed image type.
func ValidateContentType(ct string) error {
	_, err := ExtForContentType(ct)
	return err
}

// NewFileName builds a unique filename with the correct extension for ct.
func NewFileName(id, ct string) (string, error) {
	ext, err := ExtForContentType(ct)
	if err != nil {
		return "", err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("empty file id")
	}
	return id + ext, nil
}
