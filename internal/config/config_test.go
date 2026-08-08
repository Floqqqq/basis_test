package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("JWT_SECRET", "")

	cfg := Load()

	if cfg.AppPort != "8080" {
		t.Fatalf("AppPort = %q, want 8080", cfg.AppPort)
	}
	if cfg.PostgresDSN != "postgres://postgres:postgres@localhost:5432/task_manager?sslmode=disable" {
		t.Fatalf("PostgresDSN = %q, want default DSN", cfg.PostgresDSN)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("RedisAddr = %q, want localhost:6379", cfg.RedisAddr)
	}
	if cfg.JWTSecret != "secret" {
		t.Fatalf("JWTSecret = %q, want secret", cfg.JWTSecret)
	}
}

func TestLoadEnvironmentValues(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@postgres:5432/db?sslmode=disable")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg := Load()

	if cfg.AppPort != "9090" {
		t.Fatalf("AppPort = %q, want 9090", cfg.AppPort)
	}
	if cfg.PostgresDSN != "postgres://user:pass@postgres:5432/db?sslmode=disable" {
		t.Fatalf("PostgresDSN = %q, want env DSN", cfg.PostgresDSN)
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("RedisAddr = %q, want redis:6379", cfg.RedisAddr)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q, want test-secret", cfg.JWTSecret)
	}
}
