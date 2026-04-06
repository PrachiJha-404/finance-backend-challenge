package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DBURL          string
	JWTSecret      string
	JWTExpiryHours int
}

func Load() (*Config, error) {

	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/finance_db?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-key-change-me")

	expiryStr := getEnv("JWT_EXPIRY_HOURS", "24")
	expiry, _ := strconv.Atoi(expiryStr)

	return &Config{
		Port:           port,
		DBURL:          dbURL,
		JWTSecret:      jwtSecret,
		JWTExpiryHours: expiry,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
