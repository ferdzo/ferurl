package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Database string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Name     string
}

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}
	return nil

}

func GetEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func DatabaseUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", GetEnv("DB_USERNAME", "postgres"),
		GetEnv("DB_PASSWORD", "postgres"), GetEnv("DB_HOST", "localhost"), GetEnv("DB_PORT", "5432"), GetEnv("DB_NAME", "postgres"))
}

func RedisUrl() string {
	return fmt.Sprintf("%s:%s", GetEnv("REDIS_HOST", "localhost"), GetEnv("REDIS_PORT", "6379"))
}

func LoadRedisConfig() (RedisConfig, error) {
	err := LoadEnv()
	if err != nil {
		return RedisConfig{}, err
	}

	return RedisConfig{
		Host:     GetEnv("REDIS_HOST", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Database: GetEnv("REDIS_DB", "0"),
	}, nil
}

func LoadDbConfig() (DatabaseConfig, error) {
	err := LoadEnv()
	if err != nil {
		return DatabaseConfig{}, err
	}

	return DatabaseConfig{
		Host:     GetEnv("POSTGRES_HOST", "localhost"),
		Port:     GetEnv("POSTGRES_PORT", "5432"),
		Username: GetEnv("POSTGRES_USER", "postgres"),
		Password: GetEnv("POSTGRES_PASSWORD", "password"),
		Name:     GetEnv("POSTGRES_DB", "ferurl"),
	}, nil
}
