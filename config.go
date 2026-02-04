// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package config provides unified configuration management for the Lux blockchain stack.
// All components (cli, netrunner, node, evm) should import this package for consistent
// configuration handling.
package config

import (
	"fmt"
	"path/filepath"
)

// LuxConfig is the unified configuration for all Lux components
type LuxConfig struct {
	// DataDir is the base directory for all Lux data
	DataDir string `json:"data-dir" yaml:"data-dir" mapstructure:"data-dir"`

	// PluginDir is the directory for VM plugins
	PluginDir string `json:"plugin-dir" yaml:"plugin-dir" mapstructure:"plugin-dir"`

	// Log contains logging configuration
	Log LogConfig `json:"log" yaml:"log" mapstructure:"log"`

	// Network contains network-related configuration
	Network NetworkConfig `json:"network" yaml:"network" mapstructure:"network"`

	// Node contains node-specific configuration
	Node NodeConfig `json:"node" yaml:"node" mapstructure:"node"`
}

// LogConfig defines unified logging settings
type LogConfig struct {
	// Level is the minimum log level to output
	Level string `json:"level" yaml:"level" mapstructure:"level"`

	// Format is the log output format (terminal, json, plain)
	Format string `json:"format" yaml:"format" mapstructure:"format"`

	// Directory is where log files are written
	Directory string `json:"directory" yaml:"directory" mapstructure:"directory"`

	// MaxSize is the maximum size in megabytes before log rotation
	MaxSize int `json:"max-size" yaml:"max-size" mapstructure:"max-size"`

	// MaxFiles is the maximum number of old log files to retain
	MaxFiles int `json:"max-files" yaml:"max-files" mapstructure:"max-files"`

	// MaxAge is the maximum number of days to retain old log files
	MaxAge int `json:"max-age" yaml:"max-age" mapstructure:"max-age"`

	// Compress enables compression of rotated log files
	Compress bool `json:"compress" yaml:"compress" mapstructure:"compress"`

	// ShowCaller shows caller information in log entries
	ShowCaller bool `json:"show-caller" yaml:"show-caller" mapstructure:"show-caller"`

	// ShowColors enables colored output for terminal format
	ShowColors bool `json:"show-colors" yaml:"show-colors" mapstructure:"show-colors"`
}

// NetworkConfig defines network-related settings
type NetworkConfig struct {
	// ID is the network ID
	ID uint32 `json:"id" yaml:"id" mapstructure:"id"`

	// Name is the network name (mainnet, testnet, local)
	Name string `json:"name" yaml:"name" mapstructure:"name"`

	// APIEndpoint is the primary API endpoint
	APIEndpoint string `json:"api-endpoint" yaml:"api-endpoint" mapstructure:"api-endpoint"`
}

// NodeConfig defines node-specific settings
type NodeConfig struct {
	// HTTPPort is the HTTP API port
	HTTPPort int `json:"http-port" yaml:"http-port" mapstructure:"http-port"`

	// StakingPort is the staking/P2P port
	StakingPort int `json:"staking-port" yaml:"staking-port" mapstructure:"staking-port"`

	// DBType is the database backend type
	DBType string `json:"db-type" yaml:"db-type" mapstructure:"db-type"`
}

// Validate validates the configuration
func (c *LuxConfig) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("data-dir cannot be empty")
	}

	if c.PluginDir == "" {
		return fmt.Errorf("plugin-dir cannot be empty")
	}

	// Validate log level
	validLevels := map[string]bool{
		"verbo": true, "debug": true, "trace": true, "info": true,
		"warn": true, "error": true, "fatal": true, "off": true,
	}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	// Validate log format
	validFormats := map[string]bool{
		"terminal": true, "json": true, "plain": true,
	}
	if !validFormats[c.Log.Format] {
		return fmt.Errorf("invalid log format: %s", c.Log.Format)
	}

	// Validate network
	if c.Network.ID == 0 {
		return fmt.Errorf("network.id cannot be zero")
	}

	// Validate ports
	if c.Node.HTTPPort < 1 || c.Node.HTTPPort > 65535 {
		return fmt.Errorf("invalid http-port: %d", c.Node.HTTPPort)
	}
	if c.Node.StakingPort < 1 || c.Node.StakingPort > 65535 {
		return fmt.Errorf("invalid staking-port: %d", c.Node.StakingPort)
	}

	return nil
}

// GetLogPath returns the full path for a named log file
func (c *LuxConfig) GetLogPath(name string) string {
	return filepath.Join(c.Log.Directory, name+".log")
}

// GetPluginPath returns the full path for a plugin binary
func (c *LuxConfig) GetPluginPath(vmID string) string {
	return filepath.Join(c.PluginDir, vmID)
}

// GetDBPath returns the database path
func (c *LuxConfig) GetDBPath() string {
	return filepath.Join(c.DataDir, "db")
}

// GetStakingPath returns the staking keys path
func (c *LuxConfig) GetStakingPath() string {
	return filepath.Join(c.DataDir, "staking")
}

// GetConfigsPath returns the configs directory path
func (c *LuxConfig) GetConfigsPath() string {
	return filepath.Join(c.DataDir, "configs")
}

// GetChainsConfigPath returns the chains config directory path
func (c *LuxConfig) GetChainsConfigPath() string {
	return filepath.Join(c.GetConfigsPath(), "chains")
}

// GetVMsConfigPath returns the VMs config directory path
func (c *LuxConfig) GetVMsConfigPath() string {
	return filepath.Join(c.GetConfigsPath(), "vms")
}

// GetNetsConfigPath returns the nets config directory path
func (c *LuxConfig) GetNetsConfigPath() string {
	return filepath.Join(c.GetConfigsPath(), "nets")
}
