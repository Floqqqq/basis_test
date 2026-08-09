package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("OTEL_ENABLED", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("WORKER_METRICS_PORT", "")

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
	if cfg.OTelEnabled {
		t.Fatal("OTelEnabled = true, want false")
	}
	if cfg.OTelServiceName != "task-manager-api" {
		t.Fatalf("OTelServiceName = %q, want task-manager-api", cfg.OTelServiceName)
	}
	if cfg.OTelExporterEndpoint != "localhost:4317" {
		t.Fatalf("OTelExporterEndpoint = %q, want localhost:4317", cfg.OTelExporterEndpoint)
	}
	if !cfg.OTelExporterInsecure {
		t.Fatal("OTelExporterInsecure = false, want true")
	}
	if cfg.SMTPHost != "localhost" || cfg.SMTPPort != 1025 || cfg.SMTPFrom != "no-reply@task-manager.local" {
		t.Fatalf("SMTP defaults = %s:%d from %q", cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
	}
	if cfg.WorkerMetricsPort != "9091" {
		t.Fatalf("WorkerMetricsPort = %q, want 9091", cfg.WorkerMetricsPort)
	}
}

func TestLoadEnvironmentValues(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@postgres:5432/db?sslmode=disable")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "test-api")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "sender")
	t.Setenv("SMTP_PASSWORD", "secret-password")
	t.Setenv("SMTP_FROM", "sender@example.com")
	t.Setenv("WORKER_METRICS_PORT", "9191")

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
	if !cfg.OTelEnabled {
		t.Fatal("OTelEnabled = false, want true")
	}
	if cfg.OTelServiceName != "test-api" {
		t.Fatalf("OTelServiceName = %q, want test-api", cfg.OTelServiceName)
	}
	if cfg.OTelExporterEndpoint != "collector:4317" {
		t.Fatalf("OTelExporterEndpoint = %q, want collector:4317", cfg.OTelExporterEndpoint)
	}
	if cfg.OTelExporterInsecure {
		t.Fatal("OTelExporterInsecure = true, want false")
	}
	if cfg.SMTPHost != "smtp.example.com" || cfg.SMTPPort != 587 || cfg.SMTPUsername != "sender" || cfg.SMTPPassword != "secret-password" || cfg.SMTPFrom != "sender@example.com" {
		t.Fatal("SMTP environment was not loaded correctly")
	}
	if cfg.WorkerMetricsPort != "9191" {
		t.Fatalf("WorkerMetricsPort = %q, want 9191", cfg.WorkerMetricsPort)
	}
}

func TestGetEnvBoolRejectsInvalidValue(t *testing.T) {
	t.Setenv("TEST_BOOL", "not-a-bool")

	defer func() {
		if recover() == nil {
			t.Fatal("getEnvBool() did not panic")
		}
	}()

	getEnvBool("TEST_BOOL", false)
}
