package auth

import (
	"encoding/json"
	"net/http"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type AuthResponse struct {
	User *UserContext `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// RegisterHandler handles user registration requests
func (m *Manager) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, expires, err := m.Register(r.Context(), req.Username, req.Email, req.Password, req.Name)
	if err != nil {
		if err == ErrUserExists {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	isSecure := m.cfg.Env == "production"
	SetSessionCookie(w, token, expires, isSecure)
	writeJSON(w, http.StatusCreated, AuthResponse{User: user})
}

// LoginHandler handles user authentication requests
func (m *Manager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, expires, err := m.Login(r.Context(), req.Identifier, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	isSecure := m.cfg.Env == "production"
	SetSessionCookie(w, token, expires, isSecure)
	writeJSON(w, http.StatusOK, AuthResponse{User: user})
}

// LogoutHandler terminates the user session and clears the cookie
func (m *Manager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	token := GetSessionToken(r)
	if token != "" {
		_ = m.Logout(r.Context(), token)
	}

	ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// MeHandler returns the authenticated user context
func (m *Manager) MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{User: user})
}

// SessionMiddleware extracts session cookie and injects UserContext into request context
func (m *Manager) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetSessionToken(r)
		if token != "" {
			user, err := m.ValidateSession(r.Context(), token)
			if err == nil && user != nil {
				r = r.WithContext(WithUserContext(r.Context(), user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth enforces authenticated access on protected routes
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUserFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
