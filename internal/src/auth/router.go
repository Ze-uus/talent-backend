package auth

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *AuthService) {
	h := newHandler(svc)
	h.register(api)
}
