package adapters

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Cursor struct{}

func (Cursor) ID() string          { return "cursor" }
func (Cursor) DisplayName() string { return "Cursor" }

func (Cursor) Detect() bool {
	_, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".cursor"))
	return err == nil
}

func (c Cursor) Connect(opts ConnectOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cursorDir := filepath.Join(home, ".cursor")
	configPath := filepath.Join(cursorDir, "mcp.json")

	// Read existing config
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config = make(map[string]interface{})
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}

	// Ensure mcpServers exists
	var mcpServers map[string]interface{}
	if rawServers, ok := config["mcpServers"]; ok {
		if casted, ok := rawServers.(map[string]interface{}); ok {
			mcpServers = casted
		} else {
			mcpServers = make(map[string]interface{})
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	// Resolve multiversa binary path
	exePath, err := os.Executable()
	if err != nil {
		exePath = filepath.Join(home, ".local/bin/multiversa")
	}

	// Add/update multiversa
	mcpServers["multiversa"] = map[string]interface{}{
		"command": exePath,
		"args":    []string{"mcp", "serve"},
	}

	config["mcpServers"] = mcpServers

	// Ensure directory exists
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return err
	}

	// Write back
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, newData, 0644)
}

func (c Cursor) Disconnect() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".cursor", "mcp.json")

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // nothing to do
		}
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Remove multiversa from mcpServers
	if rawServers, ok := config["mcpServers"]; ok {
		if mcpServers, ok := rawServers.(map[string]interface{}); ok {
			if _, exists := mcpServers["multiversa"]; exists {
				delete(mcpServers, "multiversa")
				config["mcpServers"] = mcpServers

				// Write back
				newData, err := json.MarshalIndent(config, "", "  ")
				if err != nil {
					return err
				}
				return os.WriteFile(configPath, newData, 0644)
			}
		}
	}

	return nil
}

