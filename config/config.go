// Package config provides configuration management for bifrost-extensions.
// It uses Viper for YAML file loading with environment variable overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for bifrost-extensions
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Routing RoutingConfig `mapstructure:"routing"`
	Agents  AgentsConfig  `mapstructure:"agents"`
	OAuth   OAuthConfig   `mapstructure:"oauth"`
	Logging LoggingConfig `mapstructure:"logging"`
	Plugins PluginsConfig `mapstructure:"plugins"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	MaxRequestSize int           `mapstructure:"max_request_size_mb"`
	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	AllowedHosts   []string      `mapstructure:"allowed_hosts"`
}

// RoutingConfig holds intelligent routing settings
type RoutingConfig struct {
	RouteLLM   RouteLLMConfig   `mapstructure:"routellm"`
	ArchRouter ArchRouterConfig `mapstructure:"arch_router"`
	MIRT       MIRTConfig       `mapstructure:"mirt"`
	Semantic   SemanticConfig   `mapstructure:"semantic"`
}

// RouteLLMConfig holds RouteLLM endpoint settings
type RouteLLMConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Endpoint  string  `mapstructure:"endpoint"`
	Model     string  `mapstructure:"model"`
	Timeout   int     `mapstructure:"timeout_ms"`
	Threshold float64 `mapstructure:"threshold"`
}

// ArchRouterConfig holds Arch-Router settings
type ArchRouterConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout_ms"`
}

// MIRTConfig holds MIRT scoring settings
type MIRTConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Dimensions int     `mapstructure:"dimensions"`
	MinScore   float64 `mapstructure:"min_score"`
}

// SemanticConfig holds semantic classification settings
type SemanticConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// AgentsConfig holds CLI agent settings
type AgentsConfig struct {
	AgentAPI AgentAPIConfig `mapstructure:"agentapi"`
}

// AgentAPIConfig holds agentapi settings
type AgentAPIConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	BaseURL        string        `mapstructure:"base_url"`
	Port           int           `mapstructure:"port"`
	Timeout        time.Duration `mapstructure:"timeout"`
	PollInterval   time.Duration `mapstructure:"poll_interval"`
	MaxWaitTime    time.Duration `mapstructure:"max_wait_time"`
	TerminalWidth  int           `mapstructure:"terminal_width"`
	TerminalHeight int           `mapstructure:"terminal_height"`
	DefaultAgent   string        `mapstructure:"default_agent"`
}

// OAuthConfig holds OAuth provider settings
type OAuthConfig struct {
	Enabled   bool                     `mapstructure:"enabled"`
	AuthDir   string                   `mapstructure:"auth_dir"`
	Providers map[string]OAuthProvider `mapstructure:"providers"`
}

// OAuthProvider holds settings for a single OAuth provider
type OAuthProvider struct {
	Enabled      bool   `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	RedirectURI  string `mapstructure:"redirect_uri"`
	TokenURL     string `mapstructure:"token_url"`
	AuthURL      string `mapstructure:"auth_url"`
	Scopes       string `mapstructure:"scopes"`
	RefreshToken string `mapstructure:"refresh_token"`
	AccessToken  string `mapstructure:"access_token"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// PluginsConfig holds plugin settings
type PluginsConfig struct {
	IntelligentRouter bool `mapstructure:"intelligent_router"`
	Learning          bool `mapstructure:"learning"`
	SmartFallback     bool `mapstructure:"smart_fallback"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   120 * time.Second,
			MaxRequestSize: 10,
			AllowedOrigins: []string{"*"},
			AllowedHosts:   []string{"localhost"},
		},
		Routing: RoutingConfig{
			RouteLLM: RouteLLMConfig{
				Enabled:   false,
				Endpoint:  "http://localhost:6060/route",
				Model:     "router",
				Timeout:   5000,
				Threshold: 0.5,
			},
			ArchRouter: ArchRouterConfig{
				Enabled:  false,
				Endpoint: "http://localhost:7070/classify",
				Timeout:  5000,
			},
			MIRT: MIRTConfig{
				Enabled:    true,
				Dimensions: 25,
				MinScore:   0.3,
			},
			Semantic: SemanticConfig{
				Enabled: true,
			},
		},
		// continued in next section...
	}
}
