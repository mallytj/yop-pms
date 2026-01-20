package config

import (
	"ollerod-pms/internal/env"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func NewConfig() *Config {
	godotenv.Load("./.env")
	return &Config{
		DBHost:     env.GetEnv("DB_HOST", "localhost"),
		DBPort:     env.GetEnv("DB_PORT", "5432"),
		DBUser:     env.GetEnv("DB_USER", "nil"),
		DBPassword: env.GetEnv("DB_PASSWORD", "nil"),
		DBName:     env.GetEnv("DB_NAME", "ollerod_pms"),
	}
}
