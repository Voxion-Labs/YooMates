package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"yoomates/internal/config"
)

var (
	ErrUserExists         = errors.New("user with this email or username already exists")
	ErrInvalidCredentials = errors.New("invalid email/username or password")
	ErrSessionNotFound    = errors.New("active session not found or expired")
)

type Manager struct {
	db  *sql.DB
	cfg *config.Config
}

func NewManager(db *sql.DB, cfg *config.Config) *Manager {
	return &Manager{
		db:  db,
		cfg: cfg,
	}
}

// Register creates a new user, hashes the password with Argon2id, and persists to MySQL
func (m *Manager) Register(ctx context.Context, username, email, password, name string) (*UserContext, string, time.Time, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if username == "" || email == "" || password == "" {
		return nil, "", time.Time{}, errors.New("username, email and password are required")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate clean unique user ID
	randomBytes, err := GenerateSessionToken()
	if err != nil {
		return nil, "", time.Time{}, err
	}
	userID := "usr_" + randomBytes[:24]

	query := `INSERT INTO users (id, username, email, password_hash, name) VALUES (?, ?, ?, ?, ?)`
	_, err = m.db.ExecContext(ctx, query, userID, username, email, hash, name)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062") {
			return nil, "", time.Time{}, ErrUserExists
		}
		return nil, "", time.Time{}, fmt.Errorf("failed to insert user into mysql: %w", err)
	}

	user := &UserContext{
		ID:       userID,
		Username: username,
		Email:    email,
		Name:     name,
	}

	token, expires, err := m.createSession(ctx, userID)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	return user, token, expires, nil
}

// Login verifies user credentials against the stored Argon2id hash and creates a session
func (m *Manager) Login(ctx context.Context, identifier, password string) (*UserContext, string, time.Time, error) {
	identifier = strings.ToLower(strings.TrimSpace(identifier))

	var user UserContext
	var storedHash string

	query := `
		SELECT id, username, email, COALESCE(name, ''), password_hash, COALESCE(image, '')
		FROM users
		WHERE email = ? OR username = ?
		LIMIT 1
	`
	err := m.db.QueryRowContext(ctx, query, identifier, identifier).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Name,
		&storedHash,
		&user.Image,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", time.Time{}, ErrInvalidCredentials
		}
		return nil, "", time.Time{}, fmt.Errorf("database query error: %w", err)
	}

	valid, err := VerifyPassword(password, storedHash)
	if err != nil || !valid {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	token, expires, err := m.createSession(ctx, user.ID)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	return &user, token, expires, nil
}

// ValidateSession verifies if the session token is active in MySQL
func (m *Manager) ValidateSession(ctx context.Context, sessionToken string) (*UserContext, error) {
	if sessionToken == "" {
		return nil, ErrSessionNotFound
	}

	var user UserContext
	var expires time.Time

	query := `
		SELECT u.id, u.username, u.email, COALESCE(u.name, ''), COALESCE(u.image, ''), s.expires
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.session_token = ?
		LIMIT 1
	`
	err := m.db.QueryRowContext(ctx, query, sessionToken).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Name,
		&user.Image,
		&expires,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	if time.Now().After(expires) {
		_ = m.Logout(ctx, sessionToken)
		return nil, ErrSessionNotFound
	}

	return &user, nil
}

// Logout removes the session token from MySQL
func (m *Manager) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	_, err := m.db.ExecContext(ctx, `DELETE FROM sessions WHERE session_token = ?`, sessionToken)
	return err
}

func (m *Manager) createSession(ctx context.Context, userID string) (string, time.Time, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", time.Time{}, err
	}

	sessionID := "sess_" + token[:24]
	duration := time.Duration(m.cfg.SessionMaxAgeHours) * time.Hour
	expires := time.Now().Add(duration)

	query := `INSERT INTO sessions (id, session_token, user_id, expires) VALUES (?, ?, ?, ?)`
	_, err = m.db.ExecContext(ctx, query, sessionID, token, userID, expires)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to persist session in mysql: %w", err)
	}

	return token, expires, nil
}
