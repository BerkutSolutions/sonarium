package service

import "context"

type contextKey string

const authContextKey contextKey = "soundhub-auth-user"

func WithCurrentUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, authContextKey, user)
}

func CurrentUser(ctx context.Context) *User {
	user, _ := ctx.Value(authContextKey).(*User)
	return user
}

func CurrentUserID(ctx context.Context) string {
	user := CurrentUser(ctx)
	if user == nil || user.ID == "" {
		return ""
	}
	return user.ID
}
