package config

import "os"

type Config struct {
	Addr   string
	DBPath string
}

func Load() Config {
	return Config{
		Addr:   getEnv("ADDR", ":8080"),
		DBPath: getEnv("DB_PATH", "data.db"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
