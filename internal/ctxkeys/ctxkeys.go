package ctxkeys

import (
	"context"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type key string

const (
	user_key       key = "user"
	contact_key    key = "brand_contact"
	request_id_key key = "request_id"
	ip_key         key = "ip_address"
	ua_key         key = "user_agent"
)

// Exported for packages that need typed context keys.
var (
	RequestIDKey = request_id_key
	IPKey        = ip_key
	UserAgentKey = ua_key
)

func WithUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, user_key, u)
}

func WithBrandContact(ctx context.Context, c store.BrandContact) context.Context {
	return context.WithValue(ctx, contact_key, c)
}

func WithRequestMeta(ctx context.Context, requestID, ip, ua string) context.Context {
	ctx = context.WithValue(ctx, request_id_key, requestID)
	ctx = context.WithValue(ctx, ip_key, ip)
	ctx = context.WithValue(ctx, ua_key, ua)
	return ctx
}

func UserFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(user_key).(store.User)
	return u, ok
}

func BrandContactFromContext(ctx context.Context) (store.BrandContact, bool) {
	c, ok := ctx.Value(contact_key).(store.BrandContact)
	return c, ok
}
