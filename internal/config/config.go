package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppPort   string `yaml:"app_port"`
	MySQLDSN  string `yaml:"mysql_dsn"`
	RedisAddr string `yaml:"redis_addr"`
	JWTSecret string `yaml:"jwt_secret"`
}

func Load() Config {
	cfg := Config{
		AppPort:   "8080",
		MySQLDSN:  "root:root@tcp(localhost:3306)/task_manager?parseTime=true",
		RedisAddr: "localhost:6379",
		JWTSecret: "secret",
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
	cfg.MySQLDSN = getEnv("MYSQL_DSN", cfg.MySQLDSN)
	cfg.RedisAddr = getEnv("REDIS_ADDR", cfg.RedisAddr)
	cfg.JWTSecret = getEnv("JWT_SECRET", cfg.JWTSecret)

	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
