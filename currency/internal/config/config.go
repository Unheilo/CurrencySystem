package config

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type ServiceConfig struct {
	ServerPort  int    `yaml:"server_port"`
	MetricsPort int    `yaml:"metrics_port"`
	Env         string `yaml:"env"`
}

type APIConfig struct {
	BaseURL        string `yaml:"base_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	SkipVerify     bool   `yaml:"skip_verify"`
}

type DatabaseConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
	Name          string `yaml:"name"`
	MigrationPath string `yaml:"migrations_path"`
}

type WorkerConfig struct {
	Schedule       string `yaml:"schedule"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MetricsPort    int    `yaml:"metrics_port"`
	CurrencyPair   struct {
		BaseCurrency   string `yaml:"base_currency"`
		TargetCurrency string `yaml:"target_currency"`
	} `yaml:"currency_pair"`
}

type AppConfig struct {
	Service  ServiceConfig  `yaml:"service"`
	API      APIConfig      `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
	Worker   WorkerConfig   `yaml:"worker"`
}

func (dc DatabaseConfig) ToDSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dc.Host, dc.Port, dc.User, dc.Password, dc.Name,
	)
	return dsn
}

func MustLoad() *AppConfig {

	cfg, err := Load()

	if err != nil {
		panic(err)
	}

	return cfg
}

func Load() (*AppConfig, error) {
	path := fetchConfigPath()
	if path == "" {
		return nil, errors.New("config path is empty (pass --config or set CONFIG_PATH)")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	var cfg AppConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	// --config="path/to/config.yaml"
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res

}

func (c *AppConfig) Validate() error {
	if c.Service.ServerPort <= 0 || c.Service.ServerPort > 65535 {
		return fmt.Errorf("service.server_port out of range: %d", c.Service.ServerPort)
	}

	if c.Service.MetricsPort <= 0 || c.Service.MetricsPort > 65535 {
		return fmt.Errorf("service.metrics_port out of range: %d", c.Service.MetricsPort)
	}

	if c.Worker.MetricsPort <= 0 || c.Worker.MetricsPort > 65535 {
		return fmt.Errorf("worker.metrics_port out of range: %d", c.Worker.MetricsPort)
	}

	if c.Worker.MetricsPort == c.Service.MetricsPort {
		return fmt.Errorf("worker.metrics_port and service.metrics_port must differ: %d", c.Worker.MetricsPort)
	}

	if c.API.TimeoutSeconds <= 0 {
		return fmt.Errorf("api.timeout_seconds must be positive: %d", c.API.TimeoutSeconds)
	}

	if c.Database.Host == "" {
		return errors.New("database.host is empty")
	}

	return nil
}
