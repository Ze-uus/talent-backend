package brand

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"github.com/Ze-uus/talent-backend/internal/jsonutil"
	"github.com/Ze-uus/talent-backend/internal/media"
	authpkg "github.com/Ze-uus/talent-backend/internal/src/auth"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const max_contacts_per_brand = 3

var ErrLogoUploadFailed = errors.New("logo_upload_failed")

type BrandService struct {
	st    store.Store
	media media.Uploader
}

func New(s store.Store, up media.Uploader) *BrandService {
	if up == nil {
		up = media.DisabledUploader{}
	}
	return &BrandService{st: s, media: up}
}

// LogoFile is an optional image upload for brand create/patch.
type LogoFile struct {
	Body        io.Reader
	ContentType string
}

func (s *BrandService) Create(ctx context.Context, name, industry, description, website string, logo *LogoFile) (store.Brand, error) {
	shortcode, err := s.generateShortcode(ctx, name)
	if err != nil {
		return store.Brand{}, err
	}
	id := uuid.NewString()
	b := store.Brand{
		ID:          id,
		Name:        name,
		Shortcode:   shortcode,
		Industry:    industry,
		Description: description,
		Website:     website,
		Status:      "active",
	}
	if err := s.st.CreateBrand(ctx, b); err != nil {
		return store.Brand{}, err
	}
	created, err := s.st.GetBrandByShortcode(ctx, shortcode)
	if err != nil {
		return store.Brand{}, err
	}
	if logo != nil && logo.Body != nil {
		if err := s.UpdateLogo(ctx, created.ID, logo.Body, logo.ContentType); err != nil {
			return created, fmt.Errorf("%w: %v", ErrLogoUploadFailed, err)
		}
		return s.st.GetBrandByID(ctx, created.ID)
	}
	return created, nil
}

func (s *BrandService) Patch(ctx context.Context, id string, patch store.BrandPatch, logo *LogoFile) (store.Brand, error) {
	if err := s.st.UpdateBrand(ctx, id, patch); err != nil {
		return store.Brand{}, err
	}
	if logo != nil && logo.Body != nil {
		if err := s.UpdateLogo(ctx, id, logo.Body, logo.ContentType); err != nil {
			b, _ := s.st.GetBrandByID(ctx, id)
			return b, fmt.Errorf("%w: %v", ErrLogoUploadFailed, err)
		}
	}
	return s.st.GetBrandByID(ctx, id)
}

func (s *BrandService) UpdateLogo(ctx context.Context, brandID string, body io.Reader, contentType string) error {
	if _, err := s.st.GetBrandByID(ctx, brandID); err != nil {
		return err
	}
	if err := media.ValidateContentType(contentType); err != nil {
		return err
	}
	fileName, err := media.NewFileName(uuid.NewString(), contentType)
	if err != nil {
		return err
	}
	res, err := s.media.Upload(ctx, media.UploadInput{
		Folder:      media.FolderBrand(brandID),
		FileName:    fileName,
		ContentType: contentType,
		Body:        body,
	})
	if err != nil {
		return err
	}
	url := res.URL
	return s.st.UpdateBrand(ctx, brandID, store.BrandPatch{Logo_url: &url})
}

func (s *BrandService) AddContact(ctx context.Context, brand_id, first_name, last_name, role, email, whatsapp string) (store.BrandContact, string, error) {
	existing, err := s.st.ListBrandContacts(ctx, brand_id)
	if err != nil {
		return store.BrandContact{}, "", err
	}
	active_count := 0
	for _, c := range existing {
		if c.Token_active {
			active_count++
		}
	}
	if active_count >= max_contacts_per_brand {
		return store.BrandContact{}, "", errors.New("max_contacts_reached")
	}

	viewer_token, err := generateViewerToken()
	if err != nil {
		return store.BrandContact{}, "", err
	}

	plain, hash, err := authpkg.GeneratePlainPassword()
	if err != nil {
		return store.BrandContact{}, "", err
	}

	contact := store.BrandContact{
		Brand_id:             brand_id,
		First_name:           first_name,
		Last_name:            last_name,
		Role:                 role,
		Email:                email,
		Whatsapp_number:      whatsapp,
		Viewer_token:         viewer_token,
		Access_password_hash: hash,
		Token_active:         true,
	}
	if err := s.st.CreateBrandContact(ctx, contact); err != nil {
		return store.BrandContact{}, "", err
	}

	created, err := s.st.GetBrandContactByViewerToken(ctx, viewer_token)
	if err != nil {
		return store.BrandContact{}, "", err
	}
	return created, plain, nil
}

func (s *BrandService) RegeneratePassword(ctx context.Context, contact_id string) (string, error) {
	plain, hash, err := authpkg.GeneratePlainPassword()
	if err != nil {
		return "", err
	}
	if err := s.st.RegenerateBrandContactPassword(ctx, contact_id, hash); err != nil {
		return "", err
	}
	return plain, nil
}

func (s *BrandService) RemoveContact(ctx context.Context, contact_id, actor_id string) error {
	contact, err := s.st.GetBrandContactByID(ctx, contact_id)
	if err != nil {
		return err
	}
	if err := s.st.DeactivateBrandContact(ctx, contact_id); err != nil {
		return err
	}
	_ = s.st.WriteAuditLog(ctx, store.AuditLog{
		Actor_id:     actor_id,
		Action_type:  "brand_contact_removed",
		Entity_type:  "brand_contact",
		Entity_id:    contact_id,
		Before_state: jsonutil.Marshal(contact),
	})
	return nil
}

// GetDashboard returns the brand and its active/recent campaigns for the contact dashboard.
func (s *BrandService) GetDashboard(ctx context.Context, viewer_token string) (store.Brand, []store.Campaign, error) {
	contact, err := s.st.GetBrandContactByViewerToken(ctx, viewer_token)
	if err != nil {
		return store.Brand{}, nil, err
	}
	brand, err := s.st.GetBrandByID(ctx, contact.Brand_id)
	if err != nil {
		return store.Brand{}, nil, err
	}
	campaigns, err := s.st.ListCampaigns(ctx, store.CampaignFilter{Brand_id: contact.Brand_id})
	if err != nil {
		return store.Brand{}, nil, err
	}
	return brand, campaigns, nil
}

// GetLiveMetrics returns conversion totals per cycle for a campaign.
func (s *BrandService) GetLiveMetrics(ctx context.Context, campaign_id string) (map[string]any, error) {
	cycles, err := s.st.ListCyclesByCampaign(ctx, campaign_id)
	if err != nil {
		return nil, err
	}
	metrics := make(map[string]any, len(cycles))
	for _, c := range cycles {
		total, err := s.st.GetTotalConversions(ctx, c.ID)
		if err != nil {
			continue
		}
		metrics[c.ID] = map[string]any{
			"cycle_id":    c.ID,
			"status":      c.Status,
			"conversions": total,
		}
	}
	return metrics, nil
}

func (s *BrandService) generateShortcode(ctx context.Context, name string) (string, error) {
	base := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
	if len(base) < 3 {
		base = base + strings.Repeat("X", 3-len(base))
	}
	candidate := base[:3]
	for i := 0; i < 99; i++ {
		exists, err := s.st.ShortcodeExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		suffix := i + 2
		candidate = base[:2] + string(rune('0'+suffix%10))
	}
	return "", errors.New("shortcode_generation_failed")
}

func generateViewerToken() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:24], nil
}
