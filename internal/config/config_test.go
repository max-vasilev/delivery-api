package config

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnvRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("SERVER_WRITE_TIMEOUT", "")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")
	t.Setenv("REQUEST_TIMEOUT", "")

	_, err := loadFromEnv()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is empty")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Errorf("got %q", err.Error())
	}
}

func TestLoadFromEnvCollectsAllErrors(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SERVER_READ_TIMEOUT", "15")
	t.Setenv("SERVER_WRITE_TIMEOUT", "-5s")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")
	t.Setenv("REQUEST_TIMEOUT", "not-a-duration")

	_, err := loadFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
	var loadErr *LoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("want *LoadError, got %T", err)
	}
	joined := err.Error()
	for _, want := range []string{
		"DATABASE_URL is required",
		`SERVER_READ_TIMEOUT: invalid duration "15"`,
		"SERVER_WRITE_TIMEOUT: duration must be positive, got -5s",
		`REQUEST_TIMEOUT: invalid duration "not-a-duration"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	if len(loadErr.Messages) < 4 {
		t.Errorf("got %d messages, want at least 4: %v", len(loadErr.Messages), loadErr.Messages)
	}
}

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://delivery_user:root@localhost:5433/delivery?sslmode=disable")
	t.Setenv("PORT", "")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("SERVER_WRITE_TIMEOUT", "")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")
	t.Setenv("REQUEST_TIMEOUT", "")

	cfg, err := loadFromEnv()
	if err != nil {
		t.Fatalf("loadFromEnv: %v", err)
	}
	if cfg.Server.Addr() != ":8080" {
		t.Errorf("Addr: got %q, want :8080", cfg.Server.Addr())
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout: got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.RequestTimeout != 3*time.Second {
		t.Errorf("RequestTimeout: got %v", cfg.Server.RequestTimeout)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "9090")
	t.Setenv("SERVER_WRITE_TIMEOUT", "15s")
	t.Setenv("REQUEST_TIMEOUT", "5s")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")

	cfg, err := loadFromEnv()
	if err != nil {
		t.Fatalf("loadFromEnv: %v", err)
	}
	if cfg.DB.URL != "postgres://example" {
		t.Errorf("URL: got %q", cfg.DB.URL)
	}
	if cfg.Server.Addr() != ":9090" {
		t.Errorf("Addr: got %q, want :9090", cfg.Server.Addr())
	}
	if cfg.Server.RequestTimeout != 5*time.Second {
		t.Errorf("RequestTimeout: got %v", cfg.Server.RequestTimeout)
	}
}

func TestLoadFromEnvInvalidWriteTimeoutDoesNotFallBack(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SERVER_WRITE_TIMEOUT", "15")
	t.Setenv("REQUEST_TIMEOUT", "3s")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")

	_, err := loadFromEnv()
	if err == nil {
		t.Fatal("invalid SERVER_WRITE_TIMEOUT must not fall back to default")
	}
	if !strings.Contains(err.Error(), `SERVER_WRITE_TIMEOUT: invalid duration "15"`) {
		t.Errorf("got %q", err.Error())
	}
}

func TestLoadFromEnvRequestTimeoutMustBeLessThanWriteTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REQUEST_TIMEOUT", "30s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "15s")
	t.Setenv("SERVER_READ_TIMEOUT", "")
	t.Setenv("SERVER_IDLE_TIMEOUT", "")

	_, err := loadFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "REQUEST_TIMEOUT (30s) must be less than SERVER_WRITE_TIMEOUT (15s)") {
		t.Errorf("got %q", err.Error())
	}
}

func TestAddrAcceptsPrefixedPort(t *testing.T) {
	if (ServerConfig{Port: ":8080"}).Addr() != ":8080" {
		t.Fatal("expected :8080")
	}
}

func TestDurationFromEnvInvalid(t *testing.T) {
	t.Setenv("REQUEST_TIMEOUT", "5")
	_, err := durationFromEnv("REQUEST_TIMEOUT", 3*time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `REQUEST_TIMEOUT: invalid duration "5" (want Go duration, e.g. 15s)`
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestDurationFromEnvNegative(t *testing.T) {
	t.Setenv("SERVER_WRITE_TIMEOUT", "-5s")
	_, err := durationFromEnv("SERVER_WRITE_TIMEOUT", 15*time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "SERVER_WRITE_TIMEOUT: duration must be positive, got -5s"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestDurationFromEnvEmptyUsesFallback(t *testing.T) {
	t.Setenv("REQUEST_TIMEOUT", "")
	d, err := durationFromEnv("REQUEST_TIMEOUT", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if d != 3*time.Second {
		t.Errorf("got %v", d)
	}
}

func TestGetenvFallback(t *testing.T) {
	key := "DELIVERY_API_TEST_UNSET_KEY"
	_ = os.Unsetenv(key)
	if getenv(key, "fallback") != "fallback" {
		t.Fatal("expected fallback")
	}
}
