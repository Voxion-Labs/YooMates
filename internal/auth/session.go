package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

const (
	SessionCookieName = "yoomates_session"
)

type contextKey string

const userContextKey contextKey = "yoomates_user"

type UserContext struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Image    string `json:"image,omitempty"`
}

// GenerateSessionToken creates a cryptographically secure 256-bit random session token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes for session: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SetSessionCookie sets a secure, HTTP-only cookie with the session token
func SetSessionCookie(w http.ResponseWriter, token string, expires time.Time, isSecure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie invalidates the cookie in the client browser
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetSessionToken extracts the session token from the request cookie
func GetSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

// WithUserContext stores UserContext into context.Context
func WithUserContext(ctx context.Context, user *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext retrieves UserContext from context.Context
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value(userContextKey).(*UserContext)
	return user, ok && user != nil
}
