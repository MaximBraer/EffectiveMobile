package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `yaml:"env" env-default:"development"`
	HTTPServer `yaml:"http_server"`
	Storage    `yaml:"storage"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"5s"`
}

type Storage struct {
	Address                         string        `yaml:"address" env-default:"localhost:5432"`
	User                            string        `yaml:"user"`
	Password                        string        `yaml:"password"`
	Database                        string        `yaml:"database" env-default:"default"`
	MinConns                        int           `yaml:"minConns" env-default:"1"`
	MaxConns                        int           `yaml:"maxConns" env-default:"1"`
	MaxConnLifetime                 time.Duration `yaml:"maxConnLifetime" env-default:"3600s"`
	HealthCheckPeriod               time.Duration `yaml:"healthCheckPeriod" env-default:"30s"`
	MaxConnIdleTime                 time.Duration `yaml:"maxConnIdleTime" env-default:"60s"`
	StatementTimeout                string        `yaml:"statementTimeout" env-default:"5000"`
	IdleInTransactionSessionTimeout string        `yaml:"idleInTransactionSessionTimeout" env-default:"5000"`
}

func MustLoad() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return nil, fmt.Errorf("CONFIG_PATH environment variable is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	return &cfg, nil
}
