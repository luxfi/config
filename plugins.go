// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

const (
	// Well-known VM names
	VMNameLuxEVM = "Lux EVM"
	VMNameCoreVM = "Core VM"
	VMNameAVM    = "AVM"
)

// VMID computes the VM ID from a VM name.
// This is the standard way to compute VMID: base58check(sha256(pad32(vmName)))
// Example: "Lux EVM" -> "ag3GReYPNuSR17rUP8acMdZipQBikdXNRKDyFszAysmy3vDXE"
func VMID(vmName string) string {
	// Pad to 32 bytes
	padded := make([]byte, 32)
	copy(padded, []byte(vmName))

	// SHA256 hash
	hash := sha256.Sum256(padded)

	// Base58 encode (with checksum)
	return base58.CheckEncode(hash[:], 0)
}

// WellKnownVMIDs returns a map of well-known VM names to their IDs
func WellKnownVMIDs() map[string]string {
	return map[string]string{
		VMNameLuxEVM: VMID(VMNameLuxEVM),
		VMNameCoreVM: VMID(VMNameCoreVM),
		VMNameAVM:    VMID(VMNameAVM),
	}
}

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

// Link creates a symlink from the plugin directory to a VM binary.
// This is the preferred way to "install" a VM for development.
func (pm *DefaultPluginManager) Link(vmID, binaryPath string) error {
	// Ensure plugin directory exists
	if err := pm.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Resolve the binary path to absolute
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Verify binary exists and is executable
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable: %s", absPath)
	}

	linkPath := pm.GetPath(vmID)

	// Remove existing file/symlink if present
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Create symlink
	if err := os.Symlink(absPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// LinkByName creates a symlink using the VM name to compute VMID
func (pm *DefaultPluginManager) LinkByName(vmName, binaryPath string) error {
	vmID := VMID(vmName)
	return pm.Link(vmID, binaryPath)
}

// IsSymlink checks if a plugin path is a symlink
func (pm *DefaultPluginManager) IsSymlink(vmID string) bool {
	path := pm.GetPath(vmID)
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// GetTarget returns the target of a plugin symlink
func (pm *DefaultPluginManager) GetTarget(vmID string) (string, error) {
	path := pm.GetPath(vmID)
	if !pm.IsSymlink(vmID) {
		return "", fmt.Errorf("plugin %s is not a symlink", vmID)
	}
	return os.Readlink(path)
}

// Verify checks if a plugin is properly installed and executable
func (pm *DefaultPluginManager) Verify(vmID string) error {
	pluginPath := pm.GetPath(vmID)

	// Check if exists
	info, err := os.Lstat(pluginPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("plugin %s not installed", vmID)
	}
	if err != nil {
		return fmt.Errorf("failed to check plugin: %w", err)
	}

	// If symlink, verify target
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}

		// Resolve relative symlinks
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(pluginPath), target)
		}

		info, err = os.Stat(target)
		if os.IsNotExist(err) {
			return fmt.Errorf("plugin symlink target missing: %s", target)
		}
		if err != nil {
			return fmt.Errorf("failed to check symlink target: %w", err)
		}
	}

	// Verify executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("plugin is not executable")
	}

	return nil
}

// ResolvePluginBaseDir returns the base plugin directory
// This contains packages/, current/, and registry.json
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
// This returns the "current" directory where VMID symlinks live for node compatibility
// Structure:
//
//	~/.lux/plugins/
//	├── packages/luxfi/evm/v1.0.0/  # Actual packages
//	├── current/ag3GReY.../         # VMID symlinks (what node uses)
//	└── registry.json
func ResolvePluginDir() string {
	baseDir := ResolvePluginBaseDir()

	// Check if new structure exists (has current/ subdirectory)
	currentDir := filepath.Join(baseDir, "current")
	if info, err := os.Stat(currentDir); err == nil && info.IsDir() {
		return currentDir
	}

	// Fall back to legacy structure (plugins directly in base dir)
	return baseDir
}
