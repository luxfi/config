// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Plugin directory structure:
// ~/.lux/plugins/
// ├── packages/                          # Actual plugin packages
// │   ├── luxfi/                         # Organization (GitHub username/org)
// │   │   ├── evm/                       # Package name
// │   │   │   ├── v1.0.0/                # Version
// │   │   │   │   ├── evm                # Binary
// │   │   │   │   └── manifest.json      # Metadata (vmid, aliases, etc)
// │   │   │   └── latest -> v1.0.0/      # Latest version symlink
// │   │   └── timestampvm/
// │   │       └── v0.1.0/
// │   └── myuser/
// │       └── myvm/
// ├── current/                           # VMID symlinks (what node uses)
// │   └── ag3GReYPNuSR... -> ../packages/luxfi/evm/v1.0.0/evm
// └── registry.json                      # Local registry of installed packages

const (
	packagesDir  = "packages"
	activeDir    = "current" // Symlinks by VMID for node compatibility (unified with SDK constants.CurrentPluginDir)
	registryFile = "registry.json"
)

// PluginManifest contains metadata about an installed plugin
type PluginManifest struct {
	// Name is the package name (e.g., "evm")
	Name string `json:"name"`

	// Org is the organization/username (e.g., "luxfi")
	Org string `json:"org"`

	// Version is the semantic version (e.g., "v1.0.0")
	Version string `json:"version"`

	// VMID is the computed VM identifier hash
	VMID string `json:"vmid"`

	// VMName is the canonical VM name used to compute VMID
	VMName string `json:"vm_name,omitempty"`

	// Aliases are alternative names for this VM
	Aliases []string `json:"aliases,omitempty"`

	// Binary is the executable filename
	Binary string `json:"binary"`

	// Description is a human-readable description
	Description string `json:"description,omitempty"`

	// Repository is the source repository URL
	Repository string `json:"repository,omitempty"`

	// InstalledAt is when the plugin was installed
	InstalledAt time.Time `json:"installed_at"`

	// Size is the binary size in bytes
	Size int64 `json:"size,omitempty"`
}

// PluginRegistry tracks all installed plugins
type PluginRegistry struct {
	// Plugins maps "org/name" to list of installed versions
	Plugins map[string][]string `json:"plugins"`

	// Active maps VMID to active package reference
	Active map[string]string `json:"active"`

	// UpdatedAt is when the registry was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// PluginPackageManager provides proper package manager functionality
type PluginPackageManager struct {
	baseDir  string
	registry *PluginRegistry
}

// NewPluginPackageManager creates a new package manager
func NewPluginPackageManager(baseDir string) (*PluginPackageManager, error) {
	if baseDir == "" {
		baseDir = ResolvePluginBaseDir()
	}

	pm := &PluginPackageManager{
		baseDir: baseDir,
	}

	// Ensure directory structure exists
	if err := pm.ensureDirectories(); err != nil {
		return nil, err
	}

	// Load or create registry
	if err := pm.loadRegistry(); err != nil {
		return nil, err
	}

	return pm, nil
}

// ensureDirectories creates the required directory structure
func (pm *PluginPackageManager) ensureDirectories() error {
	dirs := []string{
		pm.baseDir,
		filepath.Join(pm.baseDir, packagesDir),
		filepath.Join(pm.baseDir, activeDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// loadRegistry loads or creates the plugin registry
func (pm *PluginPackageManager) loadRegistry() error {
	registryPath := filepath.Join(pm.baseDir, registryFile)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			pm.registry = &PluginRegistry{
				Plugins:   make(map[string][]string),
				Active:    make(map[string]string),
				UpdatedAt: time.Now(),
			}
			return nil
		}
		return fmt.Errorf("failed to read registry: %w", err)
	}

	pm.registry = &PluginRegistry{}
	if err := json.Unmarshal(data, pm.registry); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	return nil
}

// saveRegistry persists the registry to disk
func (pm *PluginPackageManager) saveRegistry() error {
	pm.registry.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(pm.registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	registryPath := filepath.Join(pm.baseDir, registryFile)
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// PackagePath returns the path for a specific package version
func (pm *PluginPackageManager) PackagePath(org, name, version string) string {
	return filepath.Join(pm.baseDir, packagesDir, org, name, version)
}

// ActivePath returns the path for VMID symlinks (node compatibility)
func (pm *PluginPackageManager) ActivePath(vmid string) string {
	return filepath.Join(pm.baseDir, activeDir, vmid)
}

// Install installs a plugin from a binary path
func (pm *PluginPackageManager) Install(ctx context.Context, manifest *PluginManifest, binaryPath string) error {
	// Validate manifest
	if manifest.Org == "" || manifest.Name == "" || manifest.Version == "" {
		return fmt.Errorf("manifest must have org, name, and version")
	}
	if manifest.VMID == "" {
		return fmt.Errorf("manifest must have vmid")
	}

	// Create package directory
	pkgPath := pm.PackagePath(manifest.Org, manifest.Name, manifest.Version)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Determine binary name
	binaryName := manifest.Binary
	if binaryName == "" {
		binaryName = manifest.Name
	}

	// Copy binary to package directory
	destBinaryPath := filepath.Join(pkgPath, binaryName)
	if err := copyFile(binaryPath, destBinaryPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(destBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Get binary size
	info, _ := os.Stat(destBinaryPath)
	if info != nil {
		manifest.Size = info.Size()
	}
	manifest.InstalledAt = time.Now()

	// Write manifest
	manifestPath := filepath.Join(pkgPath, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Update registry
	pkgKey := fmt.Sprintf("%s/%s", manifest.Org, manifest.Name)
	versions := pm.registry.Plugins[pkgKey]
	if !contains(versions, manifest.Version) {
		pm.registry.Plugins[pkgKey] = append(versions, manifest.Version)
	}

	// Activate this version (create VMID symlink)
	if err := pm.Activate(ctx, manifest.Org, manifest.Name, manifest.Version); err != nil {
		return fmt.Errorf("failed to activate plugin: %w", err)
	}

	// Create "latest" symlink
	latestPath := filepath.Join(pm.baseDir, packagesDir, manifest.Org, manifest.Name, "latest")
	_ = os.Remove(latestPath)
	if err := os.Symlink(manifest.Version, latestPath); err != nil {
		// Non-fatal, just log
		fmt.Printf("warning: failed to create latest symlink: %v\n", err)
	}

	return pm.saveRegistry()
}

// Link creates a symlink-based installation (for development)
// Unlike Install which copies the binary, Link creates a symlink to the source
func (pm *PluginPackageManager) Link(ctx context.Context, manifest *PluginManifest, binaryPath string) error {
	// Validate manifest
	if manifest.Org == "" || manifest.Name == "" || manifest.Version == "" {
		return fmt.Errorf("manifest must have org, name, and version")
	}
	if manifest.VMID == "" {
		return fmt.Errorf("manifest must have vmid")
	}

	// Resolve binary path to absolute
	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Verify binary exists
	info, err := os.Stat(absBinaryPath)
	if err != nil {
		return fmt.Errorf("binary not found: %w", err)
	}
	manifest.Size = info.Size()
	manifest.InstalledAt = time.Now()

	// Create package directory
	pkgPath := pm.PackagePath(manifest.Org, manifest.Name, manifest.Version)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Determine binary name
	binaryName := manifest.Binary
	if binaryName == "" {
		binaryName = manifest.Name
	}

	// Create symlink to binary in package directory (NOT copy)
	destBinaryPath := filepath.Join(pkgPath, binaryName)
	if _, err := os.Lstat(destBinaryPath); err == nil {
		if err := os.Remove(destBinaryPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}
	if err := os.Symlink(absBinaryPath, destBinaryPath); err != nil {
		return fmt.Errorf("failed to create binary symlink: %w", err)
	}

	// Write manifest
	manifestPath := filepath.Join(pkgPath, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Update registry
	pkgKey := fmt.Sprintf("%s/%s", manifest.Org, manifest.Name)
	versions := pm.registry.Plugins[pkgKey]
	if !contains(versions, manifest.Version) {
		pm.registry.Plugins[pkgKey] = append(versions, manifest.Version)
	}

	// Activate this version (create VMID symlink pointing directly to source binary)
	vmidPath := pm.ActivePath(manifest.VMID)
	if _, err := os.Lstat(vmidPath); err == nil {
		if err := os.Remove(vmidPath); err != nil {
			return fmt.Errorf("failed to remove existing VMID symlink: %w", err)
		}
	}
	// For linked packages, VMID symlink points directly to source binary
	if err := os.Symlink(absBinaryPath, vmidPath); err != nil {
		return fmt.Errorf("failed to create VMID symlink: %w", err)
	}

	// Update registry
	pm.registry.Active[manifest.VMID] = fmt.Sprintf("%s/%s@%s", manifest.Org, manifest.Name, manifest.Version)

	// Create "latest" symlink
	latestPath := filepath.Join(pm.baseDir, packagesDir, manifest.Org, manifest.Name, "latest")
	_ = os.Remove(latestPath)
	_ = os.Symlink(manifest.Version, latestPath)

	return pm.saveRegistry()
}

// Activate creates the VMID symlink for a specific version
func (pm *PluginPackageManager) Activate(ctx context.Context, org, name, version string) error {
	// Load manifest to get VMID
	manifest, err := pm.GetManifest(org, name, version)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Binary path
	binaryName := manifest.Binary
	if binaryName == "" {
		binaryName = name
	}
	binaryPath := filepath.Join(pm.PackagePath(org, name, version), binaryName)

	// Create VMID symlink in active directory
	vmidPath := pm.ActivePath(manifest.VMID)

	// Remove existing symlink if present
	if _, err := os.Lstat(vmidPath); err == nil {
		if err := os.Remove(vmidPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(binaryPath, vmidPath); err != nil {
		return fmt.Errorf("failed to create VMID symlink: %w", err)
	}

	// Update registry
	pm.registry.Active[manifest.VMID] = fmt.Sprintf("%s/%s@%s", org, name, version)

	return pm.saveRegistry()
}

// GetManifest loads the manifest for a specific package version
func (pm *PluginPackageManager) GetManifest(org, name, version string) (*PluginManifest, error) {
	manifestPath := filepath.Join(pm.PackagePath(org, name, version), "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest := &PluginManifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return manifest, nil
}

// List returns all installed packages
func (pm *PluginPackageManager) List(ctx context.Context) ([]PluginManifest, error) {
	var manifests []PluginManifest

	for pkgKey, versions := range pm.registry.Plugins {
		parts := strings.SplitN(pkgKey, "/", 2)
		if len(parts) != 2 {
			continue
		}
		org, name := parts[0], parts[1]

		for _, version := range versions {
			manifest, err := pm.GetManifest(org, name, version)
			if err != nil {
				continue // Skip packages with invalid manifests
			}
			manifests = append(manifests, *manifest)
		}
	}

	return manifests, nil
}

// ListActive returns all active plugins (those with VMID symlinks)
func (pm *PluginPackageManager) ListActive(ctx context.Context) (map[string]PluginManifest, error) {
	active := make(map[string]PluginManifest)

	entries, err := os.ReadDir(filepath.Join(pm.baseDir, activeDir))
	if err != nil {
		if os.IsNotExist(err) {
			return active, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		vmid := entry.Name()
		// Look up in registry
		if pkgRef, ok := pm.registry.Active[vmid]; ok {
			// Parse org/name@version
			atIdx := strings.LastIndex(pkgRef, "@")
			if atIdx == -1 {
				continue
			}
			pkgKey := pkgRef[:atIdx]
			version := pkgRef[atIdx+1:]
			parts := strings.SplitN(pkgKey, "/", 2)
			if len(parts) != 2 {
				continue
			}

			manifest, err := pm.GetManifest(parts[0], parts[1], version)
			if err != nil {
				continue
			}
			active[vmid] = *manifest
		}
	}

	return active, nil
}

// Uninstall removes a specific version of a package
func (pm *PluginPackageManager) Uninstall(ctx context.Context, org, name, version string) error {
	pkgPath := pm.PackagePath(org, name, version)

	// Load manifest to get VMID before removing
	manifest, err := pm.GetManifest(org, name, version)
	if err == nil && manifest.VMID != "" {
		// Remove VMID symlink
		vmidPath := pm.ActivePath(manifest.VMID)
		_ = os.Remove(vmidPath)
		delete(pm.registry.Active, manifest.VMID)
	}

	// Remove package directory
	if err := os.RemoveAll(pkgPath); err != nil {
		return fmt.Errorf("failed to remove package: %w", err)
	}

	// Update registry
	pkgKey := fmt.Sprintf("%s/%s", org, name)
	versions := pm.registry.Plugins[pkgKey]
	pm.registry.Plugins[pkgKey] = removeString(versions, version)
	if len(pm.registry.Plugins[pkgKey]) == 0 {
		delete(pm.registry.Plugins, pkgKey)
	}

	return pm.saveRegistry()
}

// GetActiveDir returns the directory containing VMID symlinks (for node compatibility)
func (pm *PluginPackageManager) GetActiveDir() string {
	return filepath.Join(pm.baseDir, activeDir)
}

// MigrateFromLegacy migrates plugins from the old VMID-based structure
func (pm *PluginPackageManager) MigrateFromLegacy(ctx context.Context, legacyDir string) error {
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to migrate
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories (new structure uses files/symlinks)
		}

		vmid := entry.Name()
		oldPath := filepath.Join(legacyDir, vmid)

		// Check if it's a symlink
		target, err := os.Readlink(oldPath)
		if err != nil {
			continue // Not a symlink, skip
		}

		// Create a basic manifest for legacy plugins
		manifest := &PluginManifest{
			Org:     "legacy",
			Name:    vmid[:8] + "...", // Truncated VMID as name
			Version: "v0.0.0",
			VMID:    vmid,
			Binary:  filepath.Base(target),
		}

		// Install the legacy plugin
		if err := pm.Install(ctx, manifest, target); err != nil {
			fmt.Printf("warning: failed to migrate legacy plugin %s: %v\n", vmid, err)
		}
	}

	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
