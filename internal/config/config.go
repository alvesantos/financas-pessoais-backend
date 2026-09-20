package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config guarda todas as configurações da aplicação, carregadas do ambiente.
type Config struct {
	Port            string
	Env             string
	LogLevel        string
	DatabaseURL     string
	JWTSecret       string
	JWTIssuer       string
	JWTExpiration   time.Duration
	BcryptCost      int
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
}

// Load lê o ambiente e aplica os padrões de desenvolvimento.
// Devolve erro em vez de encerrar: quem chama decide como falhar.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTIssuer:       getEnv("JWT_ISSUER", "financas-api"),
		JWTExpiration:   time.Duration(getEnvInt("JWT_EXPIRATION_HOURS", 24)) * time.Hour,
		BcryptCost:      getEnvInt("BCRYPT_COST", 0),
		AllowedOrigins:  splitAndTrim(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate recusa configurações que só falhariam em produção, e mais tarde.
func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL é obrigatória")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET é obrigatório (gere um com: openssl rand -base64 48)")
	}

	if c.IsProduction() && len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET precisa de ao menos 32 caracteres em produção")
	}

	if c.JWTExpiration <= 0 {
		return fmt.Errorf("JWT_EXPIRATION_HOURS precisa ser maior que zero")
	}

	return nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}
