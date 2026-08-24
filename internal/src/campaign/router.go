package campaign

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

func Mount(api huma.API, router chi.Router, svc *CampaignService) {
	h := newHandler(svc)
	h.register(api)
	router.Post("/admin/campaigns/{id}/content/images", h.uploadContentImagesHTTP)
}
