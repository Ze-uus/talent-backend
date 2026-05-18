package tracking

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *TrackingService) {
	h := newHandler(svc)
	h.register(api)
}
