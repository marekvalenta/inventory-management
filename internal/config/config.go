package config

import (
	"os"
)

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type Config struct {
	Port     string
	DataDir  string
	LogLevel LogLevel
}

func Load() *Config {
	cfg := &Config{
		Port:     envOrDefault("APP_PORT", "8080"),

		DataDir:  envOrDefault("DATA_DIR", "./data"),
		LogLevel: LogLevel(envOrDefault("LOG_LEVEL", "info")),
	}

	switch cfg.LogLevel {
	case LogDebug, LogInfo, LogWarn, LogError:
	default:
		panic("invalid LOG_LEVEL: " + string(cfg.LogLevel))
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
