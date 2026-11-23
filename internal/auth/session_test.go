package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionTokenGeneration(t *testing.T) {
	token1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	token2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("failed to generate token 2: %v", err)
	}

	if len(token1) != 64 { // 32 bytes hex encoded = 64 characters
		t.Errorf("expected 64 character hex token, got %d", len(token1))
	}
	if token1 == token2 {
		t.Error("expected randomly generated session tokens to be distinct")
	}
}

func TestSessionCookies(t *testing.T) {
	rr := httptest.NewRecorder()
	token := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	expires := time.Now().Add(24 * time.Hour)

	SetSessionCookie(rr, token, expires, false)

	res := rr.Result()
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Errorf("expected cookie value %s, got %s", token, cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}

	// Test Cookie Reading from Request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	extracted := GetSessionToken(req)
	if extracted != token {
		t.Errorf("expected extracted token %s, got %s", token, extracted)
	}

	// Test Clearing Cookie
	rrClear := httptest.NewRecorder()
	ClearSessionCookie(rrClear)
	clearCookies := rrClear.Result().Cookies()
	if len(clearCookies) == 0 {
		t.Fatal("expected cleared cookie header")
	}
	if clearCookies[0].MaxAge != -1 {
		t.Errorf("expected MaxAge -1 on cleared cookie, got %d", clearCookies[0].MaxAge)
	}
}

func TestUserContext(t *testing.T) {
	ctx := context.Background()
	dummyUser := &UserContext{
		ID:       "usr_123",
		Username: "voxiondev",
		Email:    "dev@voxionlabs.io",
		Name:     "Voxion Developer",
	}

	ctx = WithUserContext(ctx, dummyUser)

	retrieved, ok := GetUserFromContext(ctx)
	if !ok || retrieved == nil {
		t.Fatal("expected to retrieve user from context")
	}

	if retrieved.ID != "usr_123" {
		t.Errorf("expected user ID usr_123, got %s", retrieved.ID)
	}
	if retrieved.Username != "voxiondev" {
		t.Errorf("expected username voxiondev, got %s", retrieved.Username)
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Unauthorized call
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rr.Code)
	}

	// Authorized call with user in context
	user := &UserContext{ID: "usr_1", Username: "tester"}
	reqAuth := httptest.NewRequest(http.MethodGet, "/protected", nil)
	reqAuth = reqAuth.WithContext(WithUserContext(reqAuth.Context(), user))
	rrAuth := httptest.NewRecorder()
	handler.ServeHTTP(rrAuth, reqAuth)

	if rrAuth.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rrAuth.Code)
	}
}
