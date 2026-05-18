package brand

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *BrandService) {
	h := newHandler(svc)
	h.register(api)
}
