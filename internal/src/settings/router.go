package settings

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

func Mount(api huma.API, r chi.Router, svc *SettingsService) {
	h := newHandler(svc)
	h.registerHTTP(r)
	h.register(api)
}
