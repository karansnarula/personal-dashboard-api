package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	minJWTSecretLength = 32
)

// Config is the full runtime configuration. Every field comes from the
// environment; there are no per-environment config files.
type Config struct {
	AppEnv          string        `env:"APP_ENV" envDefault:"development"`
	HTTPPort        int           `env:"HTTP_PORT" envDefault:"8080"`
	DatabaseURL     string        `env:"DATABASE_URL,required,notEmpty"`
	JWTSecret       string        `env:"JWT_SECRET,required,notEmpty"`
	JWTTTL          time.Duration `env:"JWT_TTL" envDefault:"24h"`
	ExternalTimeout time.Duration `env:"EXTERNAL_TIMEOUT" envDefault:"5s"`

	// Browser origins allowed to call the API (comma-separated in env).
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:5173,http://localhost:3000"`

	OpenWeatherMapKey string `env:"OPENWEATHERMAP_API_KEY"`
	NewsAPIKey        string `env:"NEWSAPI_API_KEY"`
	FinnhubKey        string `env:"FINNHUB_API_KEY"`
}

// Load reads an optional .env file (real environment variables take
// precedence), parses the environment into Config, and validates it.
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	var errs []error
	if c.AppEnv != EnvDevelopment && c.AppEnv != EnvProduction {
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, c.AppEnv))
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", c.HTTPPort))
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength))
	}
	if c.JWTTTL <= 0 {
		errs = append(errs, errors.New("JWT_TTL must be positive"))
	}
	if c.ExternalTimeout <= 0 {
		errs = append(errs, errors.New("EXTERNAL_TIMEOUT must be positive"))
	}
	return errors.Join(errs...)
}

func (c Config) IsDevelopment() bool { return c.AppEnv == EnvDevelopment }
