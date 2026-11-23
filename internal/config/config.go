package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	Env                string
	AppURL             string
	DatabaseURL        string
	RedisURL           string
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3UseSSL           bool
	SessionSecret      string
	SessionMaxAgeHours int
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		i, err := strconv.Atoi(val)
		if err == nil {
			return i
		}
	}
	return defaultVal
}

// Load loads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		AppURL:             getEnv("APP_URL", "http://localhost:8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://yoomates:yoomates_secret@localhost:5432/yoomates?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		S3Endpoint:         getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:           getEnv("S3_BUCKET", "yoomates-media"),
		S3UseSSL:           getEnvBool("S3_USE_SSL", false),
		SessionSecret:      getEnv("SESSION_SECRET", "voxion_yoomates_development_secret_change_in_production"),
		SessionMaxAgeHours: getEnvInt("SESSION_MAX_AGE_HOURS", 168),
	}
}
