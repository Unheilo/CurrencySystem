package config

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type User struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

type Config struct {
	Env             string `yaml:"env"`
	HTTPPort        int    `yaml:"http_port"`
	TokenTTLMinutes int    `yaml:"token_ttl_minutes"`
	Issuer          string `yaml:"issuer"`
	Users           []User `yaml:"users"`
	JWTSecret       string `yaml:"-" env:"JWT_SECRET"`
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
		return nil, errors.New("http_port invalid")
	}
	if cfg.TokenTTLMinutes <= 0 {
		return nil, errors.New("token_ttl_minutes must be > 0")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes (HS256 requirement)")
	}
	if len(cfg.Users) == 0 {
		return nil, errors.New("users list must not be empty")
	}
	for i, u := range cfg.Users {
		if u.Login == "" || u.Password == "" {
			return nil, fmt.Errorf("user[%d]: login and password are required", i)
		}
	}
	return &cfg, nil
}
