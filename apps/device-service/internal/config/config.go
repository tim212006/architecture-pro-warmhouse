package config

import "os"

const (
	defaultHTTPPort    = "8080"
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/devices"
	defaultNATSURL     = "nats://localhost:4222"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	NATSURL     string
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

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNATSURL
	}

	return Config{
		HTTPPort:    port,
		DatabaseURL: databaseURL,
		NATSURL:     natsURL,
	}
}
