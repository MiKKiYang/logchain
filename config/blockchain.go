package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// BlockchainConfig stores common blockchain configuration across all blockchain types
type BlockchainConfig struct {
	// --- Blockchain Type Selection ---
	BlockchainType string `yaml:"blockchain_type"` // "chainmaker", "ethereum", etc.

	// --- Common Behavior Configuration ---
	RetryLimit    int `yaml:"retry_limit"`
	RetryInterval int `yaml:"retry_interval"`
	TimeoutSeconds int `yaml:"timeout_seconds"`

	// --- Chain-specific Configuration ---
	// This will be loaded separately based on blockchain type
	ChainSpecific any `yaml:"-"`
}

// LoadBlockchainConfig loads blockchain configuration from the specified YAML file path
func LoadBlockchainConfig(path string) (*BlockchainConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path of config file: %w", err)
	}

	fmt.Printf("Loading blockchain configuration from '%s'...\n", absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", absPath, err)
	}

	var cfg BlockchainConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	fmt.Println("Blockchain configuration loaded successfully.")
	return &cfg, nil
}
