package config

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env              string `yaml:"env"`
	HTTPPort         int    `yaml:"http_port"`
	CurrencyGRPCAddr string `yaml:"currency_grpc_addr" env:"CURRENCY_GRPC_ADDR"`
	AuthHTTPAddr     string `yaml:"auth_http_addr"     env:"AUTH_HTTP_ADDR"`
	JWTSecret        string `yaml:"-"                  env:"JWT_SECRET"`
}

func Load() (*Config, error) {
	var path string
	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()
	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}
	if path == "" {
		return nil, errors.New("config path is empty")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found: %s", path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if cfg.HTTPPort <= 0 {
		return nil, errors.New("currency_grpc_addr empty")
	}
	return &cfg, nil
}
