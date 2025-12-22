// Copyright (C) 2024-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.PluginDir == "" {
		t.Error("PluginDir should not be empty")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", cfg.Log.Level)
	}
	if cfg.Network.ID != 96369 {
		t.Errorf("Expected network ID 96369, got %d", cfg.Network.ID)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*LuxConfig)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *LuxConfig) {},
			wantErr: false,
		},
		{
			name:    "empty data dir",
			modify:  func(c *LuxConfig) { c.DataDir = "" },
			wantErr: true,
		},
		{
			name:    "empty plugin dir",
			modify:  func(c *LuxConfig) { c.PluginDir = "" },
			wantErr: true,
		},
		{
			name:    "invalid log level",
			modify:  func(c *LuxConfig) { c.Log.Level = "invalid" },
			wantErr: true,
		},
		{
			name:    "invalid log format",
			modify:  func(c *LuxConfig) { c.Log.Format = "invalid" },
			wantErr: true,
		},
		{
			name:    "zero network ID",
			modify:  func(c *LuxConfig) { c.Network.ID = 0 },
			wantErr: true,
		},
		{
			name:    "invalid HTTP port",
			modify:  func(c *LuxConfig) { c.Node.HTTPPort = 0 },
			wantErr: true,
		},
		{
			name:    "invalid staking port",
			modify:  func(c *LuxConfig) { c.Node.StakingPort = 70000 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPluginManager(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "lux-plugin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm := NewPluginManagerWithDir(tmpDir)

	// Test GetPluginDir
	if pm.GetPluginDir() != tmpDir {
		t.Errorf("GetPluginDir() = %q, want %q", pm.GetPluginDir(), tmpDir)
	}

	// Test List on empty directory
	ctx := context.Background()
	plugins, err := pm.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("List() returned %d plugins, want 0", len(plugins))
	}

	// Test Exists on non-existent plugin
	vmID := "test-vm-id"
	if pm.Exists(vmID) {
		t.Error("Exists() returned true for non-existent plugin")
	}

	// Create a test plugin file
	testPluginPath := filepath.Join(tmpDir, "source-plugin")
	if err := os.WriteFile(testPluginPath, []byte("test content"), 0755); err != nil {
		t.Fatalf("Failed to create test plugin: %v", err)
	}

	// Test Install
	if err := pm.Install(ctx, testPluginPath, vmID); err != nil {
		t.Errorf("Install() error = %v", err)
	}

	// Verify plugin exists
	if !pm.Exists(vmID) {
		t.Error("Exists() returned false after Install()")
	}

	// Test Get
	info, err := pm.Get(ctx, vmID)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !info.Installed {
		t.Error("Get() returned Installed=false for installed plugin")
	}

	// Test List after install
	plugins, err = pm.List(ctx)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	// Check that our installed plugin is in the list
	found := false
	for _, p := range plugins {
		if p.VMID == vmID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() did not include installed plugin %s", vmID)
	}

	// Test Uninstall
	if err := pm.Uninstall(ctx, vmID); err != nil {
		t.Errorf("Uninstall() error = %v", err)
	}

	// Verify plugin is gone
	if pm.Exists(vmID) {
		t.Error("Exists() returned true after Uninstall()")
	}
}

func TestLoaderWithEnvVars(t *testing.T) {
	// Save original values
	origDataDir := os.Getenv("LUX_DATA_DIR")
	origPluginDir := os.Getenv("LUX_PLUGIN_DIR")
	defer func() {
		os.Setenv("LUX_DATA_DIR", origDataDir)
		os.Setenv("LUX_PLUGIN_DIR", origPluginDir)
	}()

	// Set environment variables
	tmpDir, err := os.MkdirTemp("", "lux-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("LUX_DATA_DIR", tmpDir)
	os.Setenv("LUX_PLUGIN_DIR", filepath.Join(tmpDir, "custom-plugins"))

	loader := NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DataDir != tmpDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, tmpDir)
	}
	if cfg.PluginDir != filepath.Join(tmpDir, "custom-plugins") {
		t.Errorf("PluginDir = %q, want %q", cfg.PluginDir, filepath.Join(tmpDir, "custom-plugins"))
	}
}

func TestLogFactory(t *testing.T) {
	cfg := LogConfig{
		Level:      "debug",
		Format:     "terminal",
		ShowCaller: true,
		ShowColors: true,
	}

	factory := NewLogFactory(cfg)
	logger, err := factory.CreateLogger("test")
	if err != nil {
		t.Fatalf("CreateLogger() error = %v", err)
	}

	// Just verify logger is usable
	logger.Info("test message")
	logger.Sync()
}

func TestResolvePluginDir(t *testing.T) {
	// Save original values
	origPluginDir := os.Getenv("LUX_PLUGIN_DIR")
	origDataDir := os.Getenv("LUX_DATA_DIR")
	defer func() {
		os.Setenv("LUX_PLUGIN_DIR", origPluginDir)
		os.Setenv("LUX_DATA_DIR", origDataDir)
	}()

	// Clear environment
	os.Unsetenv("LUX_PLUGIN_DIR")
	os.Unsetenv("LUX_DATA_DIR")

	// Test with LUX_PLUGIN_DIR set
	os.Setenv("LUX_PLUGIN_DIR", "/custom/plugins")
	result := ResolvePluginDir()
	if result != "/custom/plugins" {
		t.Errorf("ResolvePluginDir() = %q, want '/custom/plugins'", result)
	}

	// Test without LUX_PLUGIN_DIR but with LUX_DATA_DIR
	os.Unsetenv("LUX_PLUGIN_DIR")
	os.Setenv("LUX_DATA_DIR", "/custom/data")
	result = ResolvePluginDir()
	if result != "/custom/data/plugins" {
		t.Errorf("ResolvePluginDir() = %q, want '/custom/data/plugins'", result)
	}
}
