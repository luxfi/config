// Copyright (C) 2021-2025, Lux Industries Inc. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ChainConfig represents the chain configuration files
type ChainConfig struct {
	Name    string          // Chain name (e.g., "zoo", "mychain")
	Genesis json.RawMessage // Genesis JSON
	Config  json.RawMessage // Chain config JSON (eth APIs, etc.)
	Upgrade json.RawMessage // Upgrade config JSON
}

// ChainManager handles unified chain configuration across all nodes
type ChainManager struct {
	paths *Paths
}

// NewChainManager creates a new chain manager
func NewChainManager(paths *Paths) *ChainManager {
	return &ChainManager{paths: paths}
}

// DefaultChainManager creates a chain manager with default paths
func DefaultChainManager() (*ChainManager, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}
	return NewChainManager(paths), nil
}

// ListChains returns all configured chains
func (cm *ChainManager) ListChains() ([]string, error) {
	chainsDir := cm.paths.ChainsBaseDir()
	entries, err := os.ReadDir(chainsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var chains []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Verify it has a genesis file
			genesisPath := cm.paths.ChainGenesis(entry.Name())
			if Exists(genesisPath) {
				chains = append(chains, entry.Name())
			}
		}
	}
	return chains, nil
}

// ChainExists checks if a chain configuration exists
func (cm *ChainManager) ChainExists(chainName string) bool {
	return Exists(cm.paths.ChainGenesis(chainName))
}

// LoadChain loads all configuration for a chain
func (cm *ChainManager) LoadChain(chainName string) (*ChainConfig, error) {
	cc := &ChainConfig{Name: chainName}

	// Load genesis (required)
	genesis, err := os.ReadFile(cm.paths.ChainGenesis(chainName))
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis for chain %s: %w", chainName, err)
	}
	cc.Genesis = genesis

	// Load config (optional)
	configPath := cm.paths.ChainConfig(chainName)
	if Exists(configPath) {
		config, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config for chain %s: %w", chainName, err)
		}
		cc.Config = config
	}

	// Load upgrade (optional)
	upgradePath := cm.paths.ChainUpgrade(chainName)
	if Exists(upgradePath) {
		upgrade, err := os.ReadFile(upgradePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read upgrade for chain %s: %w", chainName, err)
		}
		cc.Upgrade = upgrade
	}

	return cc, nil
}

// SaveChain saves chain configuration
func (cm *ChainManager) SaveChain(cc *ChainConfig) error {
	// Ensure chain directory exists
	if err := cm.paths.EnsureChainDir(cc.Name); err != nil {
		return fmt.Errorf("failed to create chain directory: %w", err)
	}

	// Save genesis (required)
	if len(cc.Genesis) > 0 {
		if err := os.WriteFile(cm.paths.ChainGenesis(cc.Name), cc.Genesis, 0644); err != nil {
			return fmt.Errorf("failed to write genesis: %w", err)
		}
	}

	// Save config (optional)
	if len(cc.Config) > 0 {
		if err := os.WriteFile(cm.paths.ChainConfig(cc.Name), cc.Config, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	// Save upgrade (optional)
	if len(cc.Upgrade) > 0 {
		if err := os.WriteFile(cm.paths.ChainUpgrade(cc.Name), cc.Upgrade, 0644); err != nil {
			return fmt.Errorf("failed to write upgrade: %w", err)
		}
	}

	return nil
}

// LoadGenesis loads just the genesis file for a chain
func (cm *ChainManager) LoadGenesis(chainName string) ([]byte, error) {
	return os.ReadFile(cm.paths.ChainGenesis(chainName))
}

// SaveGenesis saves just the genesis file for a chain
func (cm *ChainManager) SaveGenesis(chainName string, genesis []byte) error {
	if err := cm.paths.EnsureChainDir(chainName); err != nil {
		return err
	}
	return os.WriteFile(cm.paths.ChainGenesis(chainName), genesis, 0644)
}

// DeleteChain removes all configuration for a chain
func (cm *ChainManager) DeleteChain(chainName string) error {
	chainDir := cm.paths.ChainDir(chainName)
	if !Exists(chainDir) {
		return nil
	}
	return os.RemoveAll(chainDir)
}

// CopyChainConfigsToNode copies chain configs to a node's chain directory
// This is used when starting nodes to provide chain-specific configuration
// Destination: <nodeDir>/configs/chains/<chainID>/
func (cm *ChainManager) CopyChainConfigsToNode(chainName, chainID, nodeDir string) error {
	// Load chain config
	cc, err := cm.LoadChain(chainName)
	if err != nil {
		return err
	}

	// Create node's chain config directory
	nodeChainDir := filepath.Join(nodeDir, "configs", "chains", chainID)
	if err := os.MkdirAll(nodeChainDir, 0755); err != nil {
		return err
	}

	// Copy config file if exists
	if len(cc.Config) > 0 {
		configDest := filepath.Join(nodeChainDir, ConfigFile)
		if err := os.WriteFile(configDest, cc.Config, 0644); err != nil {
			return err
		}
	}

	// Copy upgrade file if exists
	if len(cc.Upgrade) > 0 {
		upgradeDest := filepath.Join(nodeChainDir, UpgradeFile)
		if err := os.WriteFile(upgradeDest, cc.Upgrade, 0644); err != nil {
			return err
		}
	}

	return nil
}

// GetChainIDFromGenesis extracts chainID from an EVM genesis file
func GetChainIDFromGenesis(genesis []byte) (uint64, error) {
	var g struct {
		Config struct {
			ChainID uint64 `json:"chainId"`
		} `json:"config"`
	}
	if err := json.Unmarshal(genesis, &g); err != nil {
		return 0, fmt.Errorf("failed to parse genesis: %w", err)
	}
	return g.Config.ChainID, nil
}
