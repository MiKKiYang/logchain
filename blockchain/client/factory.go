package blockchain

import (
	"fmt"
	"log"
	"path/filepath"

	"tlng/blockchain/client/chainmaker"
	"tlng/config"
)

// BlockchainType represents the type of blockchain client
type BlockchainType string

const (
	ChainMaker BlockchainType = "chainmaker"
	// Future blockchain types can be added here:
	// Ethereum   BlockchainType = "ethereum"
	// HyperledgerFabric BlockchainType = "hyperledger_fabric"
)

// LoadChainSpecificConfig loads chain-specific configuration based on blockchain type
func LoadChainSpecificConfig(blockchainType string, configDir string) (any, error) {
	switch BlockchainType(blockchainType) {
	case ChainMaker:
		chainmakerConfigPath := filepath.Join(configDir, "clients", "chainmaker.yml")
		return chainmaker.LoadChainMakerConfig(chainmakerConfigPath)
	case "":
		// Default to ChainMaker if not specified
		chainmakerConfigPath := filepath.Join(configDir, "clients", "chainmaker.yml")
		return chainmaker.LoadChainMakerConfig(chainmakerConfigPath)
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainType)
	}
}

// NewBlockchainClient creates a blockchain client based on the configuration
func NewBlockchainClient(cfg *config.BlockchainConfig, logger *log.Logger) (BlockchainClient, error) {
	switch BlockchainType(cfg.BlockchainType) {
	case ChainMaker:
		return chainmaker.NewChainMakerClient(cfg, logger)
	case "":
		// Default to ChainMaker if not specified
		return chainmaker.NewChainMakerClient(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", cfg.BlockchainType)
	}
}

// NewBlockchainClientFromFile creates a blockchain client from configuration files
func NewBlockchainClientFromFile(configPath string, logger *log.Logger) (BlockchainClient, error) {
	// Load common configuration
	cfg, err := config.LoadBlockchainConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load common config from file '%s': %w", configPath, err)
	}

	// Load chain-specific configuration
	configDir := filepath.Dir(configPath)
	chainSpecificCfg, err := LoadChainSpecificConfig(cfg.BlockchainType, configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load chain-specific config: %w", err)
	}

	cfg.ChainSpecific = chainSpecificCfg
	return NewBlockchainClient(cfg, logger)
}