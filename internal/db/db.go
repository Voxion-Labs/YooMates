package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// NewPool initializes an enterprise-grade MySQL connection pool using database/sql
func NewPool(ctx context.Context, connString string) (*sql.DB, error) {
	// If parseTime is not in DSN, append it so MySQL DATETIME maps cleanly to time.Time
	if !strings.Contains(connString, "parseTime=true") {
		if strings.Contains(connString, "?") {
			connString += "&parseTime=true&multiStatements=true"
		} else {
			connString += "?parseTime=true&multiStatements=true"
		}
	} else if !strings.Contains(connString, "multiStatements=true") {
		connString += "&multiStatements=true"
	}

	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// High-throughput connection pooling
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("failed to ping mysql database: %w", err)
	}

	return db, nil
}

// RunMigrations executes all SQL migration files found in the specified directory
func RunMigrations(ctx context.Context, db *sql.DB, migrationDir string) error {
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

		log.Printf("[mysql-migration] Executing migration: %s\n", entry.Name())
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	log.Println("[mysql-migration] All MySQL migrations applied successfully.")
	return nil
}
