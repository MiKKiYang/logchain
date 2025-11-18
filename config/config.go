package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the complete application configuration
type Config struct {
	Engine     *EngineConfig
	ApiGateway *ApiGatewayConfig
	Blockchain *BlockchainConfig
}

// LoadConfig loads all configuration files from a directory
func LoadConfig(configDir string) (*Config, error) {
	absDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path of config directory: %w", err)
	}

	config := &Config{}

	// Load engine config
	enginePath := filepath.Join(absDir, "engine.defaults.yml")
	if _, err := os.Stat(enginePath); err == nil {
		engineCfg, err := LoadEngineConfig(enginePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load engine config: %w", err)
		}
		config.Engine = engineCfg
	}

	// Load API gateway config
	apiGatewayPath := filepath.Join(absDir, "ingestion.defaults.yml")
	if _, err := os.Stat(apiGatewayPath); err == nil {
		apiGatewayCfg, err := LoadApiGatewayConfig(apiGatewayPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load API gateway config: %w", err)
		}
		config.ApiGateway = apiGatewayCfg
	}

	// Load blockchain config
	blockchainPath := filepath.Join(absDir, "client_config.yml")
	if _, err := os.Stat(blockchainPath); err == nil {
		blockchainCfg, err := LoadBlockchainConfig(blockchainPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load blockchain config: %w", err)
		}
		config.Blockchain = blockchainCfg
	}

	return config, nil
}
