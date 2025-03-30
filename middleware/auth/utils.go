package auth

import (
	"context"
	"net/http"
)

type AuthContext struct {
	UserID   string
	UserType string // "DOCTOR" or "PATIENT"
}

type contextKey string

const authContextKey = contextKey("auth")

func SetAuthContext(r *http.Request, auth AuthContext) *http.Request {
	ctx := context.WithValue(r.Context(), authContextKey, auth)
	return r.WithContext(ctx)
}

func GetAuthContext(r *http.Request) AuthContext {
	if ctx, ok := r.Context().Value(authContextKey).(AuthContext); ok {
		return ctx
	}
	return AuthContext{}
}
