// Copyright (C) 2021-2025, Lux Industries Inc. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

// Package config provides unified configuration and path management for all Lux tools.
// This is the single source of truth for directory structures across CLI, netrunner, and node.
//
// Directory Structure:
//
//	~/.lux/                          # Root data directory
//	├── chains/                      # Unified chain configs (shared by all nodes)
//	│   └── <chainName>/
//	│       ├── genesis.json
//	│       ├── config.json
//	│       └── upgrade.json
//	├── networks/                    # Network-specific data
//	│   └── <networkName>/           # mainnet, testnet, local
//	│       └── runs/
//	│           └── <runID>/         # run_20251222_102823
//	│               ├── node1/
//	│               ├── node2/
//	│               └── ...
//	├── plugins/                     # VM plugins
//	│   └── current/                 # Active plugins (symlinks)
//	│       └── <vmid>               # e.g., ag3GReYPNuSR17rUP8acMdZipQBikdXNRKDyFszAysmy3vDXE
//	├── keys/                        # Validator keys
//	│   └── <networkName>/
//	│       └── <nodeName>/
//	│           ├── staking.key
//	│           ├── staking.crt
//	│           └── signer.key
//	└── snapshots/                   # Network snapshots for save/restore
//	    └── <snapshotName>/
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Directory names - single source of truth
const (
	// Root directory name
	LuxDir = ".lux"

	// Top-level directories under ~/.lux/
	ChainsDir    = "chains"
	NetworksDir  = "networks"
	PluginsDir   = "plugins"
	KeysDir      = "keys"
	SnapshotsDir = "snapshots"

	// Subdirectories
	RunsDir           = "runs"
	CurrentPluginsDir = "current"

	// File names for chain configs
	GenesisFile = "genesis.json"
	ConfigFile  = "config.json"
	UpgradeFile = "upgrade.json"

	// File names for node keys
	StakingKeyFile  = "staking.key"
	StakingCertFile = "staking.crt"
	SignerKeyFile   = "signer.key"

	// Network names
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"
	NetworkLocal   = "local"

	// Run directory prefix
	RunPrefix = "run"
)

// Paths provides unified path management for Lux tools.
// Create one instance and use it throughout your application.
type Paths struct {
	// BaseDir is the root data directory (default: ~/.lux)
	BaseDir string
}

// DefaultPaths returns a Paths instance using the default base directory (~/.lux)
func DefaultPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return &Paths{BaseDir: filepath.Join(homeDir, LuxDir)}, nil
}

// NewPaths creates a Paths instance with a custom base directory
func NewPaths(baseDir string) *Paths {
	return &Paths{BaseDir: baseDir}
}

// --- Chain Config Paths ---

// ChainsBaseDir returns the base directory for all chain configs
// Returns: ~/.lux/chains/
func (p *Paths) ChainsBaseDir() string {
	return filepath.Join(p.BaseDir, ChainsDir)
}

// ChainDir returns the config directory for a specific chain
// Returns: ~/.lux/chains/<chainName>/
func (p *Paths) ChainDir(chainName string) string {
	return filepath.Join(p.ChainsBaseDir(), chainName)
}

// ChainGenesis returns the genesis file path for a chain
// Returns: ~/.lux/chains/<chainName>/genesis.json
func (p *Paths) ChainGenesis(chainName string) string {
	return filepath.Join(p.ChainDir(chainName), GenesisFile)
}

// ChainConfig returns the config file path for a chain
// Returns: ~/.lux/chains/<chainName>/config.json
func (p *Paths) ChainConfig(chainName string) string {
	return filepath.Join(p.ChainDir(chainName), ConfigFile)
}

// ChainUpgrade returns the upgrade file path for a chain
// Returns: ~/.lux/chains/<chainName>/upgrade.json
func (p *Paths) ChainUpgrade(chainName string) string {
	return filepath.Join(p.ChainDir(chainName), UpgradeFile)
}

// --- Network Paths ---

// NetworksBaseDir returns the base directory for all networks
// Returns: ~/.lux/networks/
func (p *Paths) NetworksBaseDir() string {
	return filepath.Join(p.BaseDir, NetworksDir)
}

// NetworkDir returns the directory for a specific network
// Returns: ~/.lux/networks/<networkName>/
func (p *Paths) NetworkDir(networkName string) string {
	return filepath.Join(p.NetworksBaseDir(), networkName)
}

// NetworkRunsDir returns the runs directory for a network
// Returns: ~/.lux/networks/<networkName>/runs/
func (p *Paths) NetworkRunsDir(networkName string) string {
	return filepath.Join(p.NetworkDir(networkName), RunsDir)
}

// NetworkRunDir returns a specific run directory
// Returns: ~/.lux/networks/<networkName>/runs/<runID>/
func (p *Paths) NetworkRunDir(networkName, runID string) string {
	return filepath.Join(p.NetworkRunsDir(networkName), runID)
}

// NodeDir returns the directory for a specific node within a run
// Returns: ~/.lux/networks/<networkName>/runs/<runID>/<nodeName>/
func (p *Paths) NodeDir(networkName, runID, nodeName string) string {
	return filepath.Join(p.NetworkRunDir(networkName, runID), nodeName)
}

// --- Plugin Paths ---

// PluginsBaseDir returns the base directory for all plugins
// Returns: ~/.lux/plugins/
func (p *Paths) PluginsBaseDir() string {
	return filepath.Join(p.BaseDir, PluginsDir)
}

// CurrentPluginsDir returns the directory for active plugin symlinks
// Returns: ~/.lux/plugins/current/
func (p *Paths) CurrentPluginsDir() string {
	return filepath.Join(p.PluginsBaseDir(), CurrentPluginsDir)
}

// PluginPath returns the path for a specific VM plugin
// Returns: ~/.lux/plugins/current/<vmID>
func (p *Paths) PluginPath(vmID string) string {
	return filepath.Join(p.CurrentPluginsDir(), vmID)
}

// --- Key Paths ---

// KeysBaseDir returns the base directory for all keys
// Returns: ~/.lux/keys/
func (p *Paths) KeysBaseDir() string {
	return filepath.Join(p.BaseDir, KeysDir)
}

// NetworkKeysDir returns the keys directory for a specific network
// Returns: ~/.lux/keys/<networkName>/
func (p *Paths) NetworkKeysDir(networkName string) string {
	return filepath.Join(p.KeysBaseDir(), networkName)
}

// NodeKeysDir returns the keys directory for a specific node
// Returns: ~/.lux/keys/<networkName>/<nodeName>/
func (p *Paths) NodeKeysDir(networkName, nodeName string) string {
	return filepath.Join(p.NetworkKeysDir(networkName), nodeName)
}

// NodeStakingKey returns the staking key path for a node
// Returns: ~/.lux/keys/<networkName>/<nodeName>/staking.key
func (p *Paths) NodeStakingKey(networkName, nodeName string) string {
	return filepath.Join(p.NodeKeysDir(networkName, nodeName), StakingKeyFile)
}

// NodeStakingCert returns the staking cert path for a node
// Returns: ~/.lux/keys/<networkName>/<nodeName>/staking.crt
func (p *Paths) NodeStakingCert(networkName, nodeName string) string {
	return filepath.Join(p.NodeKeysDir(networkName, nodeName), StakingCertFile)
}

// NodeSignerKey returns the signer key path for a node
// Returns: ~/.lux/keys/<networkName>/<nodeName>/signer.key
func (p *Paths) NodeSignerKey(networkName, nodeName string) string {
	return filepath.Join(p.NodeKeysDir(networkName, nodeName), SignerKeyFile)
}

// --- Snapshot Paths ---

// SnapshotsBaseDir returns the base directory for all snapshots
// Returns: ~/.lux/snapshots/
func (p *Paths) SnapshotsBaseDir() string {
	return filepath.Join(p.BaseDir, SnapshotsDir)
}

// SnapshotDir returns the directory for a specific snapshot
// Returns: ~/.lux/snapshots/<snapshotName>/
func (p *Paths) SnapshotDir(snapshotName string) string {
	return filepath.Join(p.SnapshotsBaseDir(), snapshotName)
}

// --- Directory Creation Helpers ---

// EnsureDir creates a directory if it doesn't exist
func (p *Paths) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureChainDir creates the chain config directory
func (p *Paths) EnsureChainDir(chainName string) error {
	return p.EnsureDir(p.ChainDir(chainName))
}

// EnsureNetworkRunsDir creates the runs directory for a network
func (p *Paths) EnsureNetworkRunsDir(networkName string) error {
	return p.EnsureDir(p.NetworkRunsDir(networkName))
}

// EnsureCurrentPluginsDir creates the current plugins directory
func (p *Paths) EnsureCurrentPluginsDir() error {
	return p.EnsureDir(p.CurrentPluginsDir())
}

// EnsureNodeKeysDir creates the keys directory for a node
func (p *Paths) EnsureNodeKeysDir(networkName, nodeName string) error {
	return p.EnsureDir(p.NodeKeysDir(networkName, nodeName))
}

// --- Run Management ---

// NewRunID generates a new timestamped run ID
// Returns: run_20251222_102823
func NewRunID() string {
	return fmt.Sprintf("%s_%s", RunPrefix, time.Now().Format("20060102_150405"))
}

// FindLatestRun finds the most recent run directory with node data
// Returns the run ID (not full path) or empty string if none found
func (p *Paths) FindLatestRun(networkName string) (string, error) {
	runsDir := p.NetworkRunsDir(networkName)
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var latestRunID string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < len(RunPrefix)+1 || name[:len(RunPrefix)+1] != RunPrefix+"_" {
			continue
		}

		// Check if this run has node directories
		runPath := filepath.Join(runsDir, name)
		nodeEntries, _ := os.ReadDir(runPath)
		hasNodes := false
		for _, nodeEntry := range nodeEntries {
			if nodeEntry.IsDir() && len(nodeEntry.Name()) >= 4 && nodeEntry.Name()[:4] == "node" {
				hasNodes = true
				break
			}
		}

		if hasNodes {
			// Timestamps sort lexicographically
			if latestRunID == "" || name > latestRunID {
				latestRunID = name
			}
		}
	}

	return latestRunID, nil
}

// GetOrCreateRun finds existing run or creates new one
// Returns the full path to the run directory
func (p *Paths) GetOrCreateRun(networkName string) (string, error) {
	// Ensure runs directory exists
	if err := p.EnsureNetworkRunsDir(networkName); err != nil {
		return "", err
	}

	// Try to find existing run
	latestRunID, err := p.FindLatestRun(networkName)
	if err != nil {
		return "", err
	}

	if latestRunID != "" {
		return p.NetworkRunDir(networkName, latestRunID), nil
	}

	// Create new run
	runID := NewRunID()
	runDir := p.NetworkRunDir(networkName, runID)
	if err := p.EnsureDir(runDir); err != nil {
		return "", err
	}

	return runDir, nil
}

// --- Utility Functions ---

// Exists checks if a path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsSymlink checks if a path is a symlink
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
