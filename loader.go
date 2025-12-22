// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "LUX"

	// ConfigFileName is the default config file name (without extension)
	ConfigFileName = "config"

	// DefaultDataDir is the default data directory
	DefaultDataDir = "~/.lux"
)

var (
	// globalConfig is the singleton configuration instance
	globalConfig *LuxConfig
	configOnce   sync.Once
	configMutex  sync.RWMutex
)

// Loader handles configuration loading from all sources
type Loader struct {
	v           *viper.Viper
	flagSet     *pflag.FlagSet
	configPaths []string
	configFile  string // Explicit config file path
}

// LoaderOption is a functional option for the Loader
type LoaderOption func(*Loader)

// WithConfigFile sets an explicit config file path
func WithConfigFile(path string) LoaderOption {
	return func(l *Loader) {
		l.configFile = path
	}
}

// WithConfigPaths sets custom config search paths
func WithConfigPaths(paths ...string) LoaderOption {
	return func(l *Loader) {
		l.configPaths = paths
	}
}

// NewLoader creates a new configuration loader
func NewLoader(opts ...LoaderOption) *Loader {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	l := &Loader{
		v:           v,
		configPaths: defaultConfigPaths(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// defaultConfigPaths returns the default configuration search paths
func defaultConfigPaths() []string {
	paths := []string{}

	// 1. Current directory
	paths = append(paths, ".")

	// 2. XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "lux"))
	}

	// 3. Home directory locations
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".lux"))
		paths = append(paths, filepath.Join(home, ".config", "lux"))
	}

	// 4. System-wide
	paths = append(paths, "/etc/lux")

	return paths
}

// BindFlags binds CLI flags to the configuration
func (l *Loader) BindFlags(fs *pflag.FlagSet) error {
	l.flagSet = fs
	return l.v.BindPFlags(fs)
}

// Load loads configuration from all sources following precedence:
// CLI Flags > Environment Variables > Config File > Defaults
func (l *Loader) Load() (*LuxConfig, error) {
	// Set defaults first
	l.setDefaults()

	// Configure viper for config file
	l.v.SetConfigName(ConfigFileName)
	l.v.SetConfigType("json") // Default type, viper auto-detects yaml/toml

	// Add search paths
	for _, path := range l.configPaths {
		l.v.AddConfigPath(expandPath(path))
	}

	// Use explicit config file if set
	if l.configFile != "" {
		l.v.SetConfigFile(expandPath(l.configFile))
	}

	// Try to read config file (optional - missing file is OK)
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	var cfg LuxConfig
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand paths
	cfg.DataDir = expandPath(cfg.DataDir)
	cfg.PluginDir = expandPath(cfg.PluginDir)
	cfg.Log.Directory = expandPath(cfg.Log.Directory)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for all configuration options
func (l *Loader) setDefaults() {
	// Get the data directory (may be set via env or flag)
	dataDir := l.v.GetString("data-dir")
	if dataDir == "" {
		dataDir = DefaultDataDir
	}
	dataDir = expandPath(dataDir)

	// Core directories
	l.v.SetDefault("data-dir", dataDir)
	l.v.SetDefault("plugin-dir", filepath.Join(dataDir, "plugins"))

	// Logging defaults
	l.v.SetDefault("log.level", "info")
	l.v.SetDefault("log.format", "terminal")
	l.v.SetDefault("log.directory", filepath.Join(dataDir, "logs"))
	l.v.SetDefault("log.max-size", 8)     // 8 MB
	l.v.SetDefault("log.max-files", 7)    // 7 rotated files
	l.v.SetDefault("log.max-age", 0)      // 0 = don't remove by age
	l.v.SetDefault("log.compress", false) // Don't compress by default
	l.v.SetDefault("log.show-caller", false)
	l.v.SetDefault("log.show-colors", true)

	// Network defaults (mainnet)
	l.v.SetDefault("network.id", 96369)
	l.v.SetDefault("network.name", "mainnet")
	l.v.SetDefault("network.api-endpoint", "http://127.0.0.1:9630")

	// Node defaults
	l.v.SetDefault("node.http-port", 9630)
	l.v.SetDefault("node.staking-port", 9631)
	l.v.SetDefault("node.db-type", "badgerdb")
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand ~
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			path = home
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path
}

// GetConfigFilePath returns the path of the config file that was loaded
func (l *Loader) GetConfigFilePath() string {
	return l.v.ConfigFileUsed()
}

// Global returns the global configuration instance (singleton)
// This lazily loads configuration on first call
func Global() *LuxConfig {
	configOnce.Do(func() {
		loader := NewLoader()
		var err error
		globalConfig, err = loader.Load()
		if err != nil {
			// Fall back to defaults on error
			// Log warning would go here
			globalConfig = DefaultConfig()
		}
	})
	return globalConfig
}

// SetGlobal sets the global configuration instance
// This should be called early in application startup
func SetGlobal(cfg *LuxConfig) {
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = cfg
}

// DefaultConfig returns the default configuration
func DefaultConfig() *LuxConfig {
	dataDir := expandPath(DefaultDataDir)
	return &LuxConfig{
		DataDir:   dataDir,
		PluginDir: filepath.Join(dataDir, "plugins"),
		Log: LogConfig{
			Level:      "info",
			Format:     "terminal",
			Directory:  filepath.Join(dataDir, "logs"),
			MaxSize:    8,
			MaxFiles:   7,
			MaxAge:     0,
			Compress:   false,
			ShowCaller: false,
			ShowColors: true,
		},
		Network: NetworkConfig{
			ID:          96369,
			Name:        "mainnet",
			APIEndpoint: "http://127.0.0.1:9630",
		},
		Node: NodeConfig{
			HTTPPort:    9630,
			StakingPort: 9631,
			DBType:      "badgerdb",
		},
	}
}

// MustLoad loads configuration and panics on error
func MustLoad(opts ...LoaderOption) *LuxConfig {
	loader := NewLoader(opts...)
	cfg, err := loader.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
