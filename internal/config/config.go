package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
}

type DBConfig struct {
	URL string
}

type LoadError struct {
	Messages []string
}

func (e *LoadError) Error() string {
	return strings.Join(e.Messages, "; ")
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}
	return loadFromEnv()
}

func loadFromEnv() (Config, error) {
	var messages []string

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		messages = append(messages, "DATABASE_URL is required")
	}

	readTimeout, err := durationFromEnv("SERVER_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		messages = append(messages, err.Error())
	}
	writeTimeout, err := durationFromEnv("SERVER_WRITE_TIMEOUT", 15*time.Second)
	if err != nil {
		messages = append(messages, err.Error())
	}
	idleTimeout, err := durationFromEnv("SERVER_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		messages = append(messages, err.Error())
	}
	requestTimeout, err := durationFromEnv("REQUEST_TIMEOUT", 3*time.Second)
	if err != nil {
		messages = append(messages, err.Error())
	}

	if writeTimeout > 0 && requestTimeout >= writeTimeout {
		messages = append(messages, fmt.Sprintf(
			"REQUEST_TIMEOUT (%s) must be less than SERVER_WRITE_TIMEOUT (%s)",
			requestTimeout, writeTimeout,
		))
	}

	if len(messages) > 0 {
		return Config{}, &LoadError{Messages: messages}
	}

	return Config{
		Server: ServerConfig{
			Port:           getenv("PORT", "8080"),
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			RequestTimeout: requestTimeout,
		},
		DB: DBConfig{URL: url},
	}, nil
}

func (c ServerConfig) Addr() string {
	if len(c.Port) > 0 && c.Port[0] == ':' {
		return c.Port
	}
	return ":" + c.Port
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q (want Go duration, e.g. 15s)", key, v)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s: duration must be positive, got %s", key, d)
	}
	return d, nil
}
