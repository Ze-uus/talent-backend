package ctxkeys

import (
	"context"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type key string

const (
	user_key    key = "user"
	contact_key key = "brand_contact"
)

func WithUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, user_key, u)
}

func WithBrandContact(ctx context.Context, c store.BrandContact) context.Context {
	return context.WithValue(ctx, contact_key, c)
}

func UserFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(user_key).(store.User)
	return u, ok
}

func BrandContactFromContext(ctx context.Context) (store.BrandContact, bool) {
	c, ok := ctx.Value(contact_key).(store.BrandContact)
	return c, ok
}
