package config

import "os"

const (
	defaultHTTPPort    = "8080"
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/telemetry"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	return Config{
		HTTPPort:    port,
		DatabaseURL: databaseURL,
	}
}
