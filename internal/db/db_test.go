package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMySQLMigrationFilesValid(t *testing.T) {
	migrationDir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	foundSQL := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			foundSQL = true
			content, err := os.ReadFile(filepath.Join(migrationDir, entry.Name()))
			if err != nil {
				t.Fatalf("failed to read %s: %v", entry.Name(), err)
			}
			sqlStr := string(content)

			if !strings.Contains(sqlStr, "ENGINE=InnoDB") {
				t.Errorf("%s missing MySQL ENGINE=InnoDB definition", entry.Name())
			}
			if !strings.Contains(sqlStr, "CREATE TABLE IF NOT EXISTS users") {
				t.Errorf("%s missing users table definition", entry.Name())
			}
			if !strings.Contains(sqlStr, "CREATE TABLE IF NOT EXISTS messages") {
				t.Errorf("%s missing messages table definition", entry.Name())
			}
			if !strings.Contains(sqlStr, "CREATE TABLE IF NOT EXISTS servers") {
				t.Errorf("%s missing servers table definition", entry.Name())
			}
			if !strings.Contains(sqlStr, "CREATE TABLE IF NOT EXISTS posts") {
				t.Errorf("%s missing posts table definition", entry.Name())
			}
		}
	}

	if !foundSQL {
		t.Error("expected at least one .sql migration file in migrations/")
	}
}
