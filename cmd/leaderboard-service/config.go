package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type config struct {
	AppPort                       int
	KafkaBrokers                  string
	KafkaTopic                    string
	KafkaGroupID                  string
	LeaderboardRefreshIntervalSec int

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
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

	return config{
		AppPort:                       envInt("LEADERBOARD_SERVICE_PORT", 8081),
		KafkaBrokers:                  envString("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:                    envString("KAFKA_TOPIC", "roll.events"),
		KafkaGroupID:                  envString("KAFKA_GROUP_ID", "leaderboard-service"),
		LeaderboardRefreshIntervalSec: envInt("LEADERBOARD_REFRESH_INTERVAL_SECONDS", 30),
		DBHost:                        envString("DB_HOST", "localhost"),
		DBPort:                        envInt("DB_PORT", 5432),
		DBUser:                        envString("DB_USER", "numbers"),
		DBPassword:                    envString("DB_PASSWORD", "numbers"),
		DBName:                        envString("DB_NAME", "numbers"),
		DBSSLMode:                     envString("DB_SSLMODE", "disable"),
	}
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
