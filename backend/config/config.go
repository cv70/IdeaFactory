package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   *ServerConfig   `yaml:"server"`
	Database *DatabaseConfig `yaml:"database"`
	Model    *ModelConfig    `yaml:"model"`
}

type ServerConfig struct {
	Env string `yaml:"env"`
}

type DatabaseConfig struct {
	DB string `json:"db"`
}

type ModelConfig struct {
	BaseURL  string `yaml:"base_url"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
	Provider string `yaml:"provider"`
}

type JWTConfig struct {
	SecretKey string `yaml:"secret_key"`
}

func LoadConfig() (*Config, error) {
	var once sync.Once
	var cfg *Config
	var err error

	once.Do(func() {
		configPath := os.Getenv("IDEA_FACTORY_CONFIG_PATH")
		if configPath == "" {
			configPath = "config/idea_factory_config.yaml"
		}

		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			err = fmt.Errorf("failed to read config file: %w", readErr)
			return
		}

		unmarshalErr := yaml.Unmarshal(data, &cfg)
		if unmarshalErr != nil {
			err = fmt.Errorf("failed to unmarshal config: %w", unmarshalErr)
			return
		}
	})

	return cfg, err
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// GetConfig returns the global config instance
func GetConfig() *Config {
	configOnce.Do(func() {
		cfg, err := LoadConfig()
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
		globalConfig = cfg
	})
	return globalConfig
}
