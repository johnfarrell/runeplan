package user

import "context"

type contextKey struct{}

// SetUser stores u in ctx and returns the updated context.
func SetUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// GetUser retrieves the User from ctx. Returns false if not present.
func GetUser(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}
