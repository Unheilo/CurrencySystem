package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type ServiceConfig struct {
	ServerPort int    `mapstructure:"server_port"`
	Env        string `mapstructure:"env"`
}

type APIConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type DatabaseConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	Name          string `mapstructure:"name"`
	MigrationPath string `mapstructure:"migrations_path"`
}

type WorkerConfig struct {
	Schedule     string `mapstructure:"schedule"`
	CurrencyPair struct {
		BaseCurrency   string `mapstructure:"base_currency"`
		TargetCurrency string `mapstructure:"target_currency"`
	} `mapstructure:"currency_pair"`
}

type AppConfig struct {
	Service  ServiceConfig  `mapstructure:"service"`
	API      APIConfig      `mapstructure:"api"`
	Database DatabaseConfig `mapstructure:"database"`
	Worker   WorkerConfig   `mapstructure:"worker"`
}

func (dc DatabaseConfig) ToDSN() string {
	// TODO: что-то с мигратором
	return ""
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
