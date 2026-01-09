// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"github.com/spf13/pflag"
)

// Flag keys used across all components
const (
	// Core directories
	DataDirKey   = "data-dir"
	PluginDirKey = "plugin-dir"

	// Logging
	LogLevelKey      = "log-level"
	LogFormatKey     = "log-format"
	LogDirKey        = "log-dir"
	LogMaxSizeKey    = "log-max-size"
	LogMaxFilesKey   = "log-max-files"
	LogMaxAgeKey     = "log-max-age"
	LogCompressKey   = "log-compress"
	LogShowCallerKey = "log-show-caller"
	LogShowColorsKey = "log-show-colors"

	// Network
	NetworkIDKey          = "network-id"
	NetworkNameKey        = "network-name"
	NetworkAPIEndpointKey = "api-endpoint"

	// Node
	HTTPPortKey    = "http-port"
	StakingPortKey = "staking-port"
	DBTypeKey      = "db-type"

	// Config file
	ConfigFileKey = "config-file"
)

// AddGlobalFlags adds common flags used by all Lux components
// This should be called by root command of each component
func AddGlobalFlags(fs *pflag.FlagSet) {
	// Core directories
	fs.String(DataDirKey, DefaultDataDir, "Base directory for Lux data")
	fs.String(PluginDirKey, "", "Directory for VM plugins (default: $DATA_DIR/plugins)")

	// Logging
	fs.String(LogLevelKey, "info", "Log level (verbo, debug, trace, info, warn, error, fatal, off)")
	fs.String(LogFormatKey, "terminal", "Log format (terminal, json, plain)")
	fs.String(LogDirKey, "", "Log directory (default: $DATA_DIR/logs)")

	// Config file
	fs.String(ConfigFileKey, "", "Path to config file")
}

// AddLogFlags adds logging-specific flags
func AddLogFlags(fs *pflag.FlagSet) {
	fs.Int(LogMaxSizeKey, 8, "Maximum log file size in MB before rotation")
	fs.Int(LogMaxFilesKey, 7, "Maximum number of rotated log files to retain")
	fs.Int(LogMaxAgeKey, 0, "Maximum age in days for log files (0 = no limit)")
	fs.Bool(LogCompressKey, false, "Compress rotated log files")
	fs.Bool(LogShowCallerKey, false, "Show caller information in logs")
	fs.Bool(LogShowColorsKey, true, "Show colors in terminal output")
}

// AddNetworkFlags adds network-related flags
func AddNetworkFlags(fs *pflag.FlagSet) {
	fs.Uint32(NetworkIDKey, 96369, "Network ID")
	fs.String(NetworkNameKey, "mainnet", "Network name (mainnet, testnet, local)")
	fs.String(NetworkAPIEndpointKey, "http://127.0.0.1:9630", "API endpoint")
}

// AddNodeFlags adds node-specific flags
func AddNodeFlags(fs *pflag.FlagSet) {
	fs.Int(HTTPPortKey, 9630, "HTTP API port")
	fs.Int(StakingPortKey, 9631, "Staking/P2P port")
	fs.String(DBTypeKey, "badgerdb", "Database type (badgerdb, leveldb, pebbledb, memdb)")
}

// AddAllFlags adds all available flags
func AddAllFlags(fs *pflag.FlagSet) {
	AddGlobalFlags(fs)
	AddLogFlags(fs)
	AddNetworkFlags(fs)
	AddNodeFlags(fs)
}

// FlagDescription provides descriptions for flags
var FlagDescriptions = map[string]string{
	DataDirKey:            "Base directory for all Lux data including plugins, logs, database, and configuration files",
	PluginDirKey:          "Directory containing VM plugin binaries. Defaults to $DATA_DIR/plugins if not specified",
	LogLevelKey:           "Minimum log level to output. Available levels: verbo, debug, trace, info, warn, error, fatal, off",
	LogFormatKey:          "Output format for logs. 'terminal' for colored output, 'json' for structured logs, 'plain' for uncolored text",
	LogDirKey:             "Directory where log files are written. Defaults to $DATA_DIR/logs if not specified",
	LogMaxSizeKey:         "Maximum size of a single log file in megabytes before rotation",
	LogMaxFilesKey:        "Maximum number of old log files to retain after rotation",
	LogMaxAgeKey:          "Maximum age in days for old log files. Files older than this are deleted. 0 means no age limit",
	LogCompressKey:        "Whether to compress rotated log files using gzip",
	LogShowCallerKey:      "Include file name and line number in log entries",
	LogShowColorsKey:      "Use ANSI colors in terminal output",
	NetworkIDKey:          "Network ID for the blockchain network",
	NetworkNameKey:        "Human-readable network name (mainnet, testnet, local, or custom)",
	NetworkAPIEndpointKey: "HTTP endpoint for the node's API",
	HTTPPortKey:           "Port for HTTP API server",
	StakingPortKey:        "Port for staking and P2P connections",
	DBTypeKey:             "Database backend type. Options: badgerdb (default), leveldb, pebbledb, memdb",
	ConfigFileKey:         "Path to configuration file. Supports JSON, YAML, and TOML formats",
}

// GetFlagDescription returns the description for a flag
func GetFlagDescription(key string) string {
	if desc, ok := FlagDescriptions[key]; ok {
		return desc
	}
	return ""
}
