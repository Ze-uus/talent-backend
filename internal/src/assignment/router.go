package assignment

import "github.com/danielgtaylor/huma/v2"

func Mount(api huma.API, svc *AssignmentService) {
	h := newHandler(svc)
	h.register(api)
}
