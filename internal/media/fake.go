package media

import (
	"context"
	"io"
	"sync"
)

// RecordingUploader captures uploads for tests.
type RecordingUploader struct {
	Mu      sync.Mutex
	Calls   []UploadInput
	Result  UploadResult
	Err     error
	ReadAll bool // when true, drains Body into memory (for assertions)
	Bodies  [][]byte
}

func (r *RecordingUploader) Upload(ctx context.Context, in UploadInput) (UploadResult, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	cp := in
	cp.Body = nil
	r.Calls = append(r.Calls, cp)
	if r.ReadAll && in.Body != nil {
		b, err := io.ReadAll(in.Body)
		if err != nil {
			return UploadResult{}, err
		}
		r.Bodies = append(r.Bodies, b)
	}
	if r.Err != nil {
		return UploadResult{}, r.Err
	}
	out := r.Result
	if out.URL == "" {
		out.URL = "https://ik.imagekit.io/test/" + in.FileName
	}
	return out, nil
}

// DisabledUploader errors on every upload (no ImageKit credentials).
type DisabledUploader struct{}

func (DisabledUploader) Upload(context.Context, UploadInput) (UploadResult, error) {
	return UploadResult{}, ErrNotConfigured
}
