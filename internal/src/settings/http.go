package settings

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/response"
)

const maxMultipartMemory = media.MaxImageBytes + (1 << 20)

func (h *handler) registerHTTP(r chi.Router) {
	r.Post("/settings/avatar", h.uploadAvatarHTTP)
}

func (h *handler) uploadAvatarHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := ctxkeys.UserFromContext(r.Context())
	if !ok {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail("unauthenticated"))
		return
	}
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	file, hdr, err := r.FormFile("avatar")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			file, hdr, err = r.FormFile("file")
		}
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail("avatar_required"))
			return
		}
	}
	defer func() { _ = file.Close() }()

	ct := hdr.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		ct = sniffImageContentType(hdr.Filename)
	}

	profile, err := h.svc.UploadAvatar(r.Context(), u.ID, file, ct)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, media.ErrInvalidType) || errors.Is(err, media.ErrTooLarge) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, media.ErrNotConfigured) {
			status = http.StatusBadGateway
		}
		response.WriteJSON(w, status, response.Fail(err.Error()))
		return
	}
	response.WriteJSON(w, http.StatusOK, response.Ok(profile, "avatar_updated"))
}

func sniffImageContentType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	default:
		return ""
	}
}
