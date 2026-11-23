package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}

	if cfg.Env != "development" {
		t.Errorf("expected default Env development, got %s", cfg.Env)
	}

	if cfg.S3Bucket != "yoomates-media" {
		t.Errorf("expected default S3Bucket yoomates-media, got %s", cfg.S3Bucket)
	}

	if cfg.SessionMaxAgeHours != 168 {
		t.Errorf("expected default SessionMaxAgeHours 168, got %d", cfg.SessionMaxAgeHours)
	}
}

func TestConfigCustomEnv(t *testing.T) {
	os.Setenv("PORT", "9999")
	os.Setenv("S3_USE_SSL", "true")
	os.Setenv("SESSION_MAX_AGE_HOURS", "48")
	defer os.Clearenv()

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("expected Port 9999, got %s", cfg.Port)
	}

	if !cfg.S3UseSSL {
		t.Errorf("expected S3UseSSL to be true")
	}

	if cfg.SessionMaxAgeHours != 48 {
		t.Errorf("expected SessionMaxAgeHours 48, got %d", cfg.SessionMaxAgeHours)
	}
}
