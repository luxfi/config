// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// PluginInfo describes an installed plugin
type PluginInfo struct {
	// VMID is the VM identifier (hash)
	VMID string `json:"vm_id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Version is the plugin version
	Version string `json:"version"`

	// Path is the full path to the plugin binary
	Path string `json:"path"`

	// Description is an optional description
	Description string `json:"description"`

	// Installed indicates if the plugin is installed
	Installed bool `json:"installed"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// ModTime is the last modification time
	ModTime time.Time `json:"mod_time"`
}

// PluginManager handles plugin discovery, installation, and loading
type PluginManager interface {
	// GetPluginDir returns the resolved plugin directory
	GetPluginDir() string

	// List returns all installed plugins
	List(ctx context.Context) ([]PluginInfo, error)

	// Get returns info about a specific plugin
	Get(ctx context.Context, vmID string) (*PluginInfo, error)

	// Install installs a plugin from a source path
	Install(ctx context.Context, source string, vmID string) error

	// Uninstall removes a plugin
	Uninstall(ctx context.Context, vmID string) error

	// GetPath returns the full path for a plugin binary
	GetPath(vmID string) string

	// Exists checks if a plugin is installed
	Exists(vmID string) bool

	// EnsureDir ensures the plugin directory exists
	EnsureDir() error
}

// DefaultPluginManager implements PluginManager
type DefaultPluginManager struct {
	pluginDir string
	config    *LuxConfig
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(cfg *LuxConfig) PluginManager {
	return &DefaultPluginManager{
		pluginDir: cfg.PluginDir,
		config:    cfg,
	}
}

// NewPluginManagerWithDir creates a plugin manager with a specific directory
func NewPluginManagerWithDir(pluginDir string) PluginManager {
	return &DefaultPluginManager{
		pluginDir: pluginDir,
	}
}

// GetPluginDir returns the plugin directory
func (pm *DefaultPluginManager) GetPluginDir() string {
	return pm.pluginDir
}

// EnsureDir ensures the plugin directory exists
func (pm *DefaultPluginManager) EnsureDir() error {
	return os.MkdirAll(pm.pluginDir, 0755)
}

// List returns all installed plugins
func (pm *DefaultPluginManager) List(ctx context.Context) ([]PluginInfo, error) {
	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		plugins = append(plugins, PluginInfo{
			VMID:      entry.Name(),
			Name:      entry.Name(),
			Path:      filepath.Join(pm.pluginDir, entry.Name()),
			Installed: true,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
		})
	}

	return plugins, nil
}

// Get returns info about a specific plugin
func (pm *DefaultPluginManager) Get(ctx context.Context, vmID string) (*PluginInfo, error) {
	path := pm.GetPath(vmID)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PluginInfo{
				VMID:      vmID,
				Name:      vmID,
				Path:      path,
				Installed: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to stat plugin: %w", err)
	}

	return &PluginInfo{
		VMID:      vmID,
		Name:      vmID,
		Path:      path,
		Installed: !info.IsDir(),
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}, nil
}

// Install installs a plugin from a source path
func (pm *DefaultPluginManager) Install(ctx context.Context, source string, vmID string) error {
	// Ensure plugin directory exists
	if err := pm.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	destPath := pm.GetPath(vmID)

	// Check if source exists
	srcInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("source file not found: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("source is a directory, expected file")
	}

	// Open source file
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination file (executable)
	dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create plugin file: %w", err)
	}
	defer dstFile.Close()

	// Copy content with context cancellation check
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			// Clean up partial file on cancellation
			os.Remove(destPath)
			return ctx.Err()
		default:
		}

		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := dstFile.Write(buf[:n]); writeErr != nil {
				os.Remove(destPath)
				return fmt.Errorf("failed to write plugin: %w", writeErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(destPath)
			return fmt.Errorf("failed to read source: %w", err)
		}
	}

	return nil
}

// Uninstall removes a plugin
func (pm *DefaultPluginManager) Uninstall(ctx context.Context, vmID string) error {
	path := pm.GetPath(vmID)

	// Check if exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Already uninstalled
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	return nil
}

// GetPath returns the full path for a plugin binary
func (pm *DefaultPluginManager) GetPath(vmID string) string {
	return filepath.Join(pm.pluginDir, vmID)
}

// Exists checks if a plugin is installed
func (pm *DefaultPluginManager) Exists(vmID string) bool {
	info, err := os.Stat(pm.GetPath(vmID))
	return err == nil && !info.IsDir()
}

// ResolvePluginBaseDir returns the base plugin directory
// This contains packages/, active/, and registry.json
func ResolvePluginBaseDir() string {
	// 1. Check environment variable first
	if dir := os.Getenv("LUX_PLUGIN_DIR"); dir != "" {
		return expandPath(dir)
	}

	// 2. Check legacy environment variable
	if dir := os.Getenv("LUXD_PLUGIN_DIR"); dir != "" {
		return expandPath(dir)
	}

	// 3. Use global config
	cfg := Global()
	if cfg != nil && cfg.PluginDir != "" {
		return cfg.PluginDir
	}

	// 4. Default based on data directory
	dataDir := os.Getenv("LUX_DATA_DIR")
	if dataDir == "" {
		dataDir = os.Getenv("LUXD_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = DefaultDataDir
	}

	return filepath.Join(expandPath(dataDir), "plugins")
}

// ResolvePluginDir resolves the plugin directory using the configuration stack
// This returns the "active" directory where VMID symlinks live for node compatibility
// Structure:
//   ~/.lux/plugins/
//   ├── packages/luxfi/evm/v1.0.0/  # Actual packages
//   ├── active/ag3GReY.../          # VMID symlinks (what node uses)
//   └── registry.json
func ResolvePluginDir() string {
	baseDir := ResolvePluginBaseDir()

	// Check if new structure exists (has active/ subdirectory)
	activeDir := filepath.Join(baseDir, "active")
	if info, err := os.Stat(activeDir); err == nil && info.IsDir() {
		return activeDir
	}

	// Fall back to legacy structure (plugins directly in base dir)
	return baseDir
}
