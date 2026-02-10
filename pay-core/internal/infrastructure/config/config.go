package config

import "os"

type Config struct {
	DatabaseURL string
	GRPCAddr    string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://qrpay:qrpay_secret@localhost:5432/qrpay?sslmode=disable"),
		GRPCAddr:    getEnv("GRPC_ADDR", ":50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
