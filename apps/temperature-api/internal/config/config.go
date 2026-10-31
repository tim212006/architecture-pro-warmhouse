package config

import "os"

const defaultHTTPPort = "8081"

// Config contains runtime settings for the service.
type Config struct {
	HTTPPort string
}

// Load reads configuration from environment variables.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	return Config{HTTPPort: port}
}
