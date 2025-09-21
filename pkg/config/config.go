package config

import (
	"context"
	"flag"
	"log/slog"

	"github.com/caarlos0/env/v6"

	"github.com/talx-hub/gophkeeper/internal/model"
)

type Config struct {
	RunAddr     string `env:"RUN_ADDRESS"    envDefault:"localhost:50051"`
	DatabaseURI string `env:"DATABASE_URI"   envDefault:""`
	SecretKey   string `env:"SECRET_KEY"     envDefault:""`
	LogLevel    string `env:"LOG_LEVEL"      envDefault:"info"`
}

type Builder struct {
	cfg *Config
	log *slog.Logger
}

func NewBuilder(log *slog.Logger) *Builder {
	return &Builder{
		cfg: &Config{
			RunAddr:     "",
			DatabaseURI: "",
			SecretKey:   "",
			LogLevel:    "",
		},
		log: log,
	}
}

func (b *Builder) FromEnv() *Builder {
	if err := env.Parse(b.cfg); err != nil {
		b.log.ErrorContext(context.Background(),
			"failed to load config from env",
			slog.Any(model.KeyLoggerError, err),
		)
	}
	return b
}

func (b *Builder) FromFlags() *Builder {
	flag.StringVar(&b.cfg.RunAddr, "a", b.cfg.RunAddr, "Run address")
	flag.StringVar(&b.cfg.DatabaseURI, "d", b.cfg.DatabaseURI, "Database URI")
	flag.StringVar(&b.cfg.SecretKey, "k", b.cfg.SecretKey, "Secret key")
	flag.StringVar(&b.cfg.LogLevel, "l", b.cfg.LogLevel, "Log level")

	flag.Parse()
	return b
}

func (b *Builder) GetConfig() *Config {
	return b.cfg
}
