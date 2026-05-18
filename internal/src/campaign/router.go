package campaign

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *CampaignService) {
	h := newHandler(svc)
	h.register(api)
}
