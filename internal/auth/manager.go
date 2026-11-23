package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"yoomates/internal/config"
)

var (
	ErrUserExists         = errors.New("user with this email or username already exists")
	ErrInvalidCredentials = errors.New("invalid email/username or password")
	ErrSessionNotFound    = errors.New("active session not found or expired")
)

type Manager struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewManager(pool *pgxpool.Pool, cfg *config.Config) *Manager {
	return &Manager{
		pool: pool,
		cfg:  cfg,
	}
}

// Register creates a new user, hashes the password with Argon2id, and creates a session
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

	var userID string
	err = m.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, username, email, hash, name).Scan(&userID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, "", time.Time{}, ErrUserExists
		}
		return nil, "", time.Time{}, fmt.Errorf("failed to insert user: %w", err)
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

	err := m.pool.QueryRow(ctx, `
		SELECT id, username, email, COALESCE(name, ''), password_hash, COALESCE(image, '')
		FROM users
		WHERE email = $1 OR username = $1
	`, identifier).Scan(&user.ID, &user.Username, &user.Email, &user.Name, &storedHash, &user.Image)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

// ValidateSession verifies if the session token is active and returns the user
func (m *Manager) ValidateSession(ctx context.Context, sessionToken string) (*UserContext, error) {
	if sessionToken == "" {
		return nil, ErrSessionNotFound
	}

	var user UserContext
	var expires time.Time

	err := m.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, COALESCE(u.name, ''), COALESCE(u.image, ''), s.expires
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.session_token = $1
	`, sessionToken).Scan(&user.ID, &user.Username, &user.Email, &user.Name, &user.Image, &expires)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

// Logout removes the session token from the database
func (m *Manager) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	_, err := m.pool.Exec(ctx, `DELETE FROM sessions WHERE session_token = $1`, sessionToken)
	return err
}

func (m *Manager) createSession(ctx context.Context, userID string) (string, time.Time, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", time.Time{}, err
	}

	duration := time.Duration(m.cfg.SessionMaxAgeHours) * time.Hour
	expires := time.Now().Add(duration)

	_, err = m.pool.Exec(ctx, `
		INSERT INTO sessions (session_token, user_id, expires)
		VALUES ($1, $2, $3)
	`, token, userID, expires)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to persist session: %w", err)
	}

	return token, expires, nil
}
