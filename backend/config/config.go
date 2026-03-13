package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server        *ServerConfig    `yaml:"server"`
	Database      *DatabaseConfig  `yaml:"database"`
	Redis         *RedisConfig     `yaml:"redis"`
	Qdrant        *QdrantConfig    `yaml:"qdrant"`
	ES            *ESConfig        `yaml:"es"`
	Doris         *DorisConfig     `yaml:"doris"`
	Scylla        *ScyllaConfig    `yaml:"scylla"`
	LLM           *LLMConfig       `yaml:"llm"`
	TextEmbedding *EmbeddingConfig `yaml:"text_embedding"`
	Browser       *BrowserConfig   `yaml:"browser"`
	JWT           *JWTConfig       `yaml:"jwt"`
}

type BrowserConfig struct {
	URL string `yaml:"url"`
}

type ServerConfig struct {
	Env string `yaml:"env"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `json:"db_name"`
}

type QdrantConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	APIKey   string `yaml:"api_key"`
}

type ESConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type DorisConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
}

type ScyllaConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
}

type LLMConfig struct {
	BaseURL          string        `yaml:"base_url"`
	Model            string        `yaml:"model"`
	Timeout          time.Duration `yaml:"timeout"`
	MaxTokens        *int          `yaml:"max_tokens,omitempty"`
	Temperature      *float32      `yaml:"temperature,omitempty"`
	TopP             *float32      `yaml:"top_p,omitempty"`
	PresencePenalty  *float32      `yaml:"presence_penalty,omitempty"`
	FrequencyPenalty *float32      `yaml:"frequency_penalty,omitempty"`
}

type EmbeddingConfig struct {
	BaseURL string `yaml:"base_url"`
}

type JWTConfig struct {
	SecretKey string `yaml:"secret_key"`
}

func LoadConfig() (*Config, error) {
	var once sync.Once
	var cfg *Config
	var err error

	once.Do(func() {
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
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
