package settings

import (
	"context"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/response"
	authsvc "github.com/Ze-uus/talent-backend/internal/src/auth"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type SettingsService struct {
	st    store.Store
	auth  *authsvc.AuthService
	media media.Uploader
}

func New(s store.Store, a *authsvc.AuthService, up media.Uploader) *SettingsService {
	if up == nil {
		up = media.DisabledUploader{}
	}
	return &SettingsService{st: s, auth: a, media: up}
}

func (s *SettingsService) GetProfile(ctx context.Context, user_id string) (store.User, error) {
	return s.st.GetUserByID(ctx, user_id)
}

func (s *SettingsService) PatchProfile(ctx context.Context, user_id string, patch store.UserPatch) error {
	return s.st.UpdateUser(ctx, user_id, patch)
}

func (s *SettingsService) UploadAvatar(ctx context.Context, userID string, body io.Reader, contentType string) (store.User, error) {
	if err := media.ValidateContentType(contentType); err != nil {
		return store.User{}, err
	}
	fileName, err := media.NewFileName(uuid.NewString(), contentType)
	if err != nil {
		return store.User{}, err
	}
	res, err := s.media.Upload(ctx, media.UploadInput{
		Folder:      media.FolderAvatar(userID),
		FileName:    fileName,
		ContentType: contentType,
		Body:        body,
	})
	if err != nil {
		return store.User{}, err
	}
	url := res.URL
	if err := s.st.UpdateUser(ctx, userID, store.UserPatch{Avatar_url: &url}); err != nil {
		return store.User{}, err
	}
	return s.st.GetUserByID(ctx, userID)
}

func (s *SettingsService) ChangePassword(ctx context.Context, user_id, old_pw, new_pw string) error {
	u, err := s.st.GetUserByID(ctx, user_id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password_hash), []byte(old_pw)); err != nil {
		return response.Unauthorized("invalid_password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(new_pw), 12)
	if err != nil {
		return err
	}
	h := string(hash)
	return s.st.UpdateUser(ctx, user_id, store.UserPatch{Password_hash: &h})
}

func (s *SettingsService) ListSessions(ctx context.Context, user_id string) ([]store.Session, error) {
	return s.st.ListSessionsByUser(ctx, user_id)
}

func (s *SettingsService) RevokeSession(ctx context.Context, session_id, user_id string) error {
	sess, err := s.st.GetSession(ctx, session_id)
	if err != nil {
		return err
	}
	if sess.User_id != user_id {
		return response.Forbidden("forbidden")
	}
	return s.st.InvalidateSession(ctx, session_id)
}

func (s *SettingsService) RevokeAllSessions(ctx context.Context, user_id string) error {
	return s.st.InvalidateAllUserSessions(ctx, user_id)
}

func (s *SettingsService) EnrollTOTP(ctx context.Context, user_id string) (secret, qr_uri string, err error) {
	return s.auth.EnrollTOTP(ctx, user_id)
}

func (s *SettingsService) VerifyTOTP(ctx context.Context, user_id, code string) error {
	return s.auth.VerifyAndEnableTOTP(ctx, user_id, code)
}

func (s *SettingsService) DisableTOTP(ctx context.Context, user_id string) error {
	f := false
	empty := ""
	return s.st.UpdateUser(ctx, user_id, store.UserPatch{
		Totp_enabled:  &f,
		Totp_verified: &f,
		Totp_secret:   &empty,
	})
}
