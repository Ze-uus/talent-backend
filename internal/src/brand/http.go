package brand

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const maxMultipartMemory = media.MaxImageBytes + (1 << 20)

func (h *handler) registerHTTP(r chi.Router) {
	r.Post("/admin/brands", h.createBrandHTTP)
	r.Patch("/admin/brands/{id}", h.patchBrandHTTP)
	r.Post("/admin/brands/{id}/logo", h.uploadLogoHTTP)
}

func (h *handler) createBrandHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAdminOrAbove(r.Context()) {
		response.WriteJSON(w, http.StatusForbidden, response.Fail("insufficient_role"))
		return
	}

	name, industry, description, website, logo, err := parseBrandWrite(r)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}
	if strings.TrimSpace(name) == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail("name_required"))
		return
	}

	b, err := h.svc.Create(r.Context(), name, industry, description, website, logo)
	if err != nil {
		if errors.Is(err, ErrLogoUploadFailed) {
			response.WriteJSON(w, http.StatusBadGateway, response.Fail(err.Error()))
			return
		}
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	response.WriteJSON(w, http.StatusOK, response.Ok(b, "brand_created"))
}

func (h *handler) patchBrandHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAdminOrAbove(r.Context()) {
		response.WriteJSON(w, http.StatusForbidden, response.Fail("insufficient_role"))
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail("id_required"))
		return
	}

	patch, logo, err := parseBrandPatch(r)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	b, err := h.svc.Patch(r.Context(), id, patch, logo)
	if err != nil {
		if errors.Is(err, ErrLogoUploadFailed) {
			response.WriteJSON(w, http.StatusBadGateway, response.Fail(err.Error()))
			return
		}
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	response.WriteJSON(w, http.StatusOK, response.Ok(b, "brand_updated"))
}

func (h *handler) uploadLogoHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAdminOrAbove(r.Context()) {
		response.WriteJSON(w, http.StatusForbidden, response.Fail("insufficient_role"))
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail("id_required"))
		return
	}
	logo, err := readLogoFile(r)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}
	if logo == nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail("logo_required"))
		return
	}
	if err := h.svc.UpdateLogo(r.Context(), id, logo.Body, logo.ContentType); err != nil {
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
	b, err := h.svc.st.GetBrandByID(r.Context(), id)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	response.WriteJSON(w, http.StatusOK, response.Ok(b, "logo_updated"))
}

func parseBrandWrite(r *http.Request) (name, industry, description, website string, logo *LogoFile, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			return "", "", "", "", nil, err
		}
		name = r.FormValue("name")
		industry = r.FormValue("industry")
		description = r.FormValue("description")
		website = r.FormValue("website")
		logo, err = optionalLogoFromForm(r)
		return name, industry, description, website, logo, err
	}

	var body struct {
		Name        string `json:"name"`
		Industry    string `json:"industry"`
		Description string `json:"description"`
		Website     string `json:"website"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", "", "", "", nil, err
	}
	return body.Name, body.Industry, body.Description, body.Website, nil, nil
}

func parseBrandPatch(r *http.Request) (store.BrandPatch, *LogoFile, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			return store.BrandPatch{}, nil, err
		}
		patch := store.BrandPatch{}
		if v, ok := r.MultipartForm.Value["name"]; ok && len(v) > 0 {
			patch.Name = &v[0]
		}
		if v, ok := r.MultipartForm.Value["industry"]; ok && len(v) > 0 {
			patch.Industry = &v[0]
		}
		if v, ok := r.MultipartForm.Value["description"]; ok && len(v) > 0 {
			patch.Description = &v[0]
		}
		if v, ok := r.MultipartForm.Value["website"]; ok && len(v) > 0 {
			patch.Website = &v[0]
		}
		logo, err := optionalLogoFromForm(r)
		return patch, logo, err
	}

	var body struct {
		Name        *string `json:"name,omitempty"`
		Industry    *string `json:"industry,omitempty"`
		Description *string `json:"description,omitempty"`
		Website     *string `json:"website,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		return store.BrandPatch{}, nil, err
	}
	return store.BrandPatch{
		Name:        body.Name,
		Industry:    body.Industry,
		Description: body.Description,
		Website:     body.Website,
	}, nil, nil
}

func optionalLogoFromForm(r *http.Request) (*LogoFile, error) {
	file, hdr, err := r.FormFile("logo")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, media.MaxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > media.MaxImageBytes {
		return nil, media.ErrTooLarge
	}

	ct := hdr.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		ct = sniffImageContentType(hdr.Filename)
	}
	return &LogoFile{Body: bytes.NewReader(payload), ContentType: ct}, nil
}

func readLogoFile(r *http.Request) (*LogoFile, error) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		return nil, err
	}
	return optionalLogoFromForm(r)
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
