package config

import "os"

type Config struct {
	ServiceName string
	GRPCAddr    string
	HTTPAddr    string
	PostgresDSN string
}

func Load() Config {
	return Config{
		ServiceName: getenv("SERVICE_NAME", "ticketing-template"),
		GRPCAddr:    getenv("GRPC_ADDR", ":9090"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
	}
}

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
