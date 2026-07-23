package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/imagekit-developer/imagekit-go/v2"
	"github.com/imagekit-developer/imagekit-go/v2/option"
)

// Config holds ImageKit credentials.
type Config struct {
	PrivateKey  string
	PublicKey   string
	URLEndpoint string
	Log         *slog.Logger
}

// New returns an ImageKit uploader when PrivateKey is set, otherwise DisabledUploader.
func New(cfg Config) Uploader {
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		log.Info("media: IMAGEKIT_PRIVATE_KEY empty — uploads disabled")
		return DisabledUploader{}
	}
	client := imagekit.NewClient(option.WithPrivateKey(cfg.PrivateKey))
	return &ImageKitUploader{client: client, log: log}
}

// ImageKitUploader uploads via the ImageKit REST API.
type ImageKitUploader struct {
	client imagekit.Client
	log    *slog.Logger
}

func (u *ImageKitUploader) Upload(ctx context.Context, in UploadInput) (UploadResult, error) {
	if err := ValidateContentType(in.ContentType); err != nil {
		return UploadResult{}, err
	}
	if strings.TrimSpace(in.FileName) == "" {
		return UploadResult{}, fmt.Errorf("empty file name")
	}
	if in.Body == nil {
		return UploadResult{}, fmt.Errorf("empty file body")
	}

	payload, err := io.ReadAll(io.LimitReader(in.Body, MaxImageBytes+1))
	if err != nil {
		return UploadResult{}, err
	}
	if len(payload) > MaxImageBytes {
		return UploadResult{}, ErrTooLarge
	}
	if len(payload) == 0 {
		return UploadResult{}, fmt.Errorf("empty file body")
	}

	file := imagekit.NewFile(bytes.NewReader(payload), in.FileName, in.ContentType)
	resp, err := u.client.Files.Upload(ctx, imagekit.FileUploadParams{
		File:     file,
		FileName: in.FileName,
		Folder:   imagekit.String(in.Folder),
	})
	if err != nil {
		return UploadResult{}, err
	}
	return UploadResult{URL: resp.URL, FileID: resp.FileID}, nil
}
