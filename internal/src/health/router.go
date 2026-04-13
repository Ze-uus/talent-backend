package health

import (
	"github.com/danielgtaylor/huma/v2"
)

// Mount registers the health endpoints on the Huma API.
// Pass nil for db if the database is not yet wired.
func Mount(api huma.API, version string, db Pinger) {
	h := new_handler(version, db)
	h.register(api)
}
