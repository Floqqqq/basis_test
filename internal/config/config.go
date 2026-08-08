package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppPort              string `yaml:"app_port"`
	PostgresDSN          string `yaml:"postgres_dsn"`
	RedisAddr            string `yaml:"redis_addr"`
	JWTSecret            string `yaml:"jwt_secret"`
	OTelEnabled          bool   `yaml:"otel_enabled"`
	OTelServiceName      string `yaml:"otel_service_name"`
	OTelExporterEndpoint string `yaml:"otel_exporter_otlp_endpoint"`
	OTelExporterInsecure bool   `yaml:"otel_exporter_otlp_insecure"`
}

func Load() Config {
	cfg := Config{
		AppPort:              "8080",
		PostgresDSN:          "postgres://postgres:postgres@localhost:5432/task_manager?sslmode=disable",
		RedisAddr:            "localhost:6379",
		JWTSecret:            "secret",
		OTelServiceName:      "task-manager-api",
		OTelExporterEndpoint: "localhost:4317",
		OTelExporterInsecure: true,
	}

	configPath := getEnv("CONFIG_PATH", "config.yaml")

	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			panic(fmt.Errorf("parse config file %s: %w", configPath, err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("read config file %s: %w", configPath, err))
	}

	cfg.AppPort = getEnv("APP_PORT", cfg.AppPort)
	cfg.PostgresDSN = getEnv("POSTGRES_DSN", cfg.PostgresDSN)
	cfg.RedisAddr = getEnv("REDIS_ADDR", cfg.RedisAddr)
	cfg.JWTSecret = getEnv("JWT_SECRET", cfg.JWTSecret)
	cfg.OTelEnabled = getEnvBool("OTEL_ENABLED", cfg.OTelEnabled)
	cfg.OTelServiceName = getEnv("OTEL_SERVICE_NAME", cfg.OTelServiceName)
	cfg.OTelExporterEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.OTelExporterEndpoint)
	cfg.OTelExporterInsecure = getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", cfg.OTelExporterInsecure)

	return cfg
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Errorf("parse %s: %w", key, err))
	}
	return parsed
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
