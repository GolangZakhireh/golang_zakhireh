package config

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	Port          string
	DataDir       string
	UpstreamProxy string
	NoSumDB       string
	AllowList     string
	DenyList      string
}

// Load loads the configuration from environment variables or defaults
func Load() *Config {
	return &Config{
		Port:          getEnv("GOLANGZAKHIREH_PORT", ":8811"),
		DataDir:       getEnv("GOLANGZAKHIREH_DATA_DIR", filepath.Join(".", "data", "modules")),
		UpstreamProxy: getEnv("GOLANGZAKHIREH_UPSTREAM", "https://proxy.golang.org"),
		NoSumDB:       getEnv("GONOSUMDB", ""),
		AllowList:     getEnv("GOLANGZAKHIREH_ALLOW", ""),
		DenyList:      getEnv("GOLANGZAKHIREH_DENY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
