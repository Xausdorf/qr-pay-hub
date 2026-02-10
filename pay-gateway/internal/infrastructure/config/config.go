package config

import "os"

type Config struct {
	CoreGRPCAddr string
	HTTPAddr     string
}

func Load() *Config {
	return &Config{
		CoreGRPCAddr: getEnv("CORE_GRPC_ADDR", "localhost:50051"),
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
