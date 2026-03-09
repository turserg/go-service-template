package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName   string
	GRPCAddr      string
	HTTPAddr      string
	PostgresDSN   string
	MigrationsDir string
	OTLPEndpoint  string
	OTLPInsecure  bool
}

func Load() Config {
	return Config{
		ServiceName:   getenv("SERVICE_NAME", "ticketing-template"),
		GRPCAddr:      getenv("GRPC_ADDR", ":9090"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:   os.Getenv("POSTGRES_DSN"),
		MigrationsDir: getenv("MIGRATIONS_DIR", "migrations"),
		OTLPEndpoint:  os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTLPInsecure:  getenvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
	}
}

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getenvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
