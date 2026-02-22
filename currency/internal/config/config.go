package config

import (
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type ServiceConfig struct {
	ServerPort int    `yaml:"server_port"`
	Env        string `yaml:"env"`
}

type APIConfig struct {
	BaseURL        string `yaml:"base_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
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
	Schedule     string `yaml:"schedule"`
	CurrencyPair struct {
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
	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg AppConfig

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config:" + err.Error())
	}

	return &cfg
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
