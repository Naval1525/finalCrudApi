package main

import "os"

type Config struct {
	DatabaseUrl string
	Port        string
	JWTSecret   string
}

func LoadConfig() *Config {
	config := &Config{
		DatabaseUrl: getEnv("DATABASE_URL", "postgres://username:password@localhost/dbname?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
	}
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
