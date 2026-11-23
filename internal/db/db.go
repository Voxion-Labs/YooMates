package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool initializes a high-performance pgx connection pool
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Optimize pool settings for real-time collaboration
	config.MaxConns = 50
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// RunMigrations executes all SQL files found in the specified migrations directory
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationDir string) error {
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("unable to read migration directory %s: %w", migrationDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(migrationDir, entry.Name())
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
		}

		log.Printf("[db-migration] Executing migration: %s\n", entry.Name())
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	log.Println("[db-migration] All migrations applied successfully.")
	return nil
}
