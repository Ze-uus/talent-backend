package payout

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *PayoutService) {
	h := newHandler(svc)
	h.register(api)
}
