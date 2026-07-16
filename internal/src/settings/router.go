package settings

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *SettingsService) {
	h := newHandler(svc)
	h.register(api)
}
