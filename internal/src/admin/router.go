package admin

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *AdminService) {
	h := newHandler(svc)
	h.register(api)
}
