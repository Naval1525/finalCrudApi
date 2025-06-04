package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseUrl string
	Port        string
	JWTSecret   string
}

func LoadConfig() *Config {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		DatabaseUrl: getEnv("DATABASE_URL", ""),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
	}

	// Debug: Print what we're actually using
	log.Printf("Config loaded - Port: %s, DatabaseUrl starts with: %.50s...",
		config.Port, config.DatabaseUrl)

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		log.Printf("Using environment variable %s", key)
		return value
	}
	log.Printf("Using default value for %s", key)
	return defaultValue
}
