package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("JWT_SECRET", "")

	cfg := Load()

	if cfg.AppPort != "8080" {
		t.Fatalf("AppPort = %q, want 8080", cfg.AppPort)
	}
	if cfg.MySQLDSN != "root:root@tcp(localhost:3306)/task_manager?parseTime=true" {
		t.Fatalf("MySQLDSN = %q, want default DSN", cfg.MySQLDSN)
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
	t.Setenv("MYSQL_DSN", "user:pass@tcp(mysql:3306)/db?parseTime=true")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg := Load()

	if cfg.AppPort != "9090" {
		t.Fatalf("AppPort = %q, want 9090", cfg.AppPort)
	}
	if cfg.MySQLDSN != "user:pass@tcp(mysql:3306)/db?parseTime=true" {
		t.Fatalf("MySQLDSN = %q, want env DSN", cfg.MySQLDSN)
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("RedisAddr = %q, want redis:6379", cfg.RedisAddr)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("JWTSecret = %q, want test-secret", cfg.JWTSecret)
	}
}
