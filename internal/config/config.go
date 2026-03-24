// Package config provides configuration management for bifrost-extensions using phenotype-go-kit.
package config

import (
	"github.com/KooshaPari/phenotype-go-kit/pkg/config"
	"github.com/spf13/viper"
)

// ServerConfig represents the server configuration for bifrost-extensions.
type ServerConfig struct {
	Port           int      `mapstructure:"port"`
	Host           string   `mapstructure:"host"`
	AllowedHosts   []string `mapstructure:"allowed_hosts"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// BifrostConfig represents configuration for bifrost core functionality.
type BifrostConfig struct {
	EnableCache  bool   `mapstructure:"enable_cache"`
	CacheBackend string `mapstructure:"cache_backend"`
	Neo4jURL     string `mapstructure:"neo4j_url"`
	NatsURL      string `mapstructure:"nats_url"`
	RedisURL     string `mapstructure:"redis_url"`
}

// BifrostExtensionsConfig represents the complete configuration for bifrost-extensions.
type BifrostExtensionsConfig struct {
	Server  ServerConfig  `mapstructure:"server"`
	Bifrost BifrostConfig `mapstructure:"bifrost"`
}

// LoadConfig loads the configuration from a file and environment variables.
// It uses phenotype-go-kit's config loader for consistent configuration handling.
func LoadConfig(filePath string) (*BifrostExtensionsConfig, error) {
	loader := config.NewConfigLoader(filePath)

	// Set defaults for bifrost-extensions configuration
	defaults := map[string]any{
		"server.port":            8080,
		"server.host":            "localhost",
		"server.allowed_hosts":   []string{"localhost", "127.0.0.1", "[::1]"},
		"server.allowed_origins": []string{"*"},
		"bifrost.enable_cache":   true,
		"bifrost.cache_backend":  "redis",
		"bifrost.neo4j_url":      "neo4j://localhost:7687",
		"bifrost.nats_url":       "nats://localhost:4222",
		"bifrost.redis_url":      "redis://localhost:6379",
	}

	// Load configuration with defaults
	if err := loader.LoadWithDefaults(defaults); err != nil {
		// If file doesn't exist, continue with defaults from environment
		viper.SetEnvPrefix("BIFROST")
		viper.AutomaticEnv()

		// Set all defaults in viper
		for key, value := range defaults {
			viper.SetDefault(key, value)
		}
	}

	// Unmarshal into config struct
	var cfg BifrostExtensionsConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadConfigWithEnv loads configuration from environment variables and a config file.
// Environment variables take precedence over config file values.
func LoadConfigWithEnv(filePath string) (*BifrostExtensionsConfig, error) {
	// First load from file
	cfg, err := LoadConfig(filePath)
	if err != nil && filePath != "" {
		return nil, err
	}

	// Then override with environment variables
	viper.SetEnvPrefix("BIFROST")
	viper.AutomaticEnv()

	// Re-unmarshal with environment overrides
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// BindEnvVars binds specific environment variables to configuration keys.
func BindEnvVars() error {
	envBindings := map[string]string{
		"server.port":            "BIFROST_PORT",
		"server.host":            "BIFROST_HOST",
		"server.allowed_hosts":   "BIFROST_ALLOWED_HOSTS",
		"server.allowed_origins": "BIFROST_ALLOWED_ORIGINS",
		"bifrost.enable_cache":   "BIFROST_ENABLE_CACHE",
		"bifrost.cache_backend":  "BIFROST_CACHE_BACKEND",
		"bifrost.neo4j_url":      "BIFROST_NEO4J_URL",
		"bifrost.nats_url":       "BIFROST_NATS_URL",
		"bifrost.redis_url":      "BIFROST_REDIS_URL",
	}

	for key, envVar := range envBindings {
		if err := viper.BindEnv(key, envVar); err != nil {
			return err
		}
	}

	return nil
}
