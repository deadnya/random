package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type config struct {
	AppPort           int
	AppEnv            string
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxConns        int32
	DBMinConns        int32
	RollMaxTokens     int
	RollRefillSeconds int
	RarityScoreScale  float64

	KafkaBrokers    string
	KafkaTopic      string

	LeaderboardServiceURL         string
	LeaderboardServicePort        int
	LeaderboardRefreshIntervalSec int
}

func (c config) databaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func loadConfig() config {
	_ = godotenv.Load()

	cfg := config{
		AppPort:           envInt("APP_PORT", 8080),
		AppEnv:            envString("APP_ENV", "development"),
		DBHost:            envString("DB_HOST", "localhost"),
		DBPort:            envInt("DB_PORT", 5432),
		DBUser:            envString("DB_USER", "numbers"),
		DBPassword:        envString("DB_PASSWORD", "numbers"),
		DBName:            envString("DB_NAME", "numbers"),
		DBSSLMode:         envString("DB_SSLMODE", "disable"),
		DBMaxConns:        int32(envInt("DB_MAX_CONNS", 10)),
		DBMinConns:        int32(envInt("DB_MIN_CONNS", 2)),
		RollMaxTokens:     envInt("ROLL_MAX_TOKENS", 10),
		RollRefillSeconds: envInt("ROLL_REFILL_SECONDS", 60),
		RarityScoreScale:              envFloat("RARITY_SCORE_SCALE", 150),
		KafkaBrokers:                  envString("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:                    envString("KAFKA_TOPIC", "roll.events"),
		LeaderboardServiceURL:         envString("LEADERBOARD_SERVICE_URL", "http://localhost:8081"),
		LeaderboardServicePort:        envInt("LEADERBOARD_SERVICE_PORT", 8081),
		LeaderboardRefreshIntervalSec: envInt("LEADERBOARD_REFRESH_INTERVAL_SECONDS", 30),
	}

	if cfg.DBMinConns > cfg.DBMaxConns {
		cfg.DBMinConns = cfg.DBMaxConns
	}

	if cfg.RollMaxTokens < 1 {
		cfg.RollMaxTokens = 1
	}
	if cfg.RollMaxTokens > 10 {
		cfg.RollMaxTokens = 10
	}
	if cfg.RollRefillSeconds < 1 {
		cfg.RollRefillSeconds = 60
	}

	return cfg
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
