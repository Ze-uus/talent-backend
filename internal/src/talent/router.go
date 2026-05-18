package talent

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *TalentService) {
	h := newHandler(svc)
	h.register(api)
}
