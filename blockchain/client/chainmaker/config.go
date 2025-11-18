package chainmaker

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// NodeConfig stores detailed configuration for a single ChainMaker node
type NodeConfig struct {
	Address     string   `yaml:"address"`
	ConnCount   int      `yaml:"conn_count"`
	UseTLS      bool     `yaml:"use_tls"`
	TLSHostName string   `yaml:"tls_host_name"`
	CaPaths     []string `yaml:"ca_paths"`
}

// ChainMakerConfig stores ChainMaker-specific configuration
type ChainMakerConfig struct {
	// --- SDK Connection Required ---
	ChainID string `yaml:"chain_id"`
	OrgID   string `yaml:"org_id"`

	// TLS Connection Credentials
	UserKeyPath  string `yaml:"user_key_path"`
	UserCertPath string `yaml:"user_cert_path"`

	// Transaction Signing Credentials
	UserSignKeyPath  string `yaml:"user_sign_key_path"`
	UserSignCertPath string `yaml:"user_sign_cert_path"`

	Nodes []NodeConfig `yaml:"nodes"`

	// --- Business Logic Required ---
	ContractName              string `yaml:"contract_name"`
	SubmitLogMethodName       string `yaml:"submit_log_method_name"`
	ParamKeyLogHash           string `yaml:"param_key_log_hash"`
	ParamKeyLogContent        string `yaml:"param_key_log_content"`
	ParamKeySenderOrgID       string `yaml:"param_key_sender_org_id"`
	ParamKeyTimestamp         string `yaml:"param_key_timestamp"`
	FindLogByHashMethodName   string `yaml:"find_log_by_hash_method_name"`
	SubmitEventTopic          string `yaml:"submit_event_topic"`
	SubmitLogsBatchMethodName string `yaml:"submit_logs_batch_method_name"`
	ParamKeyLogsJson          string `yaml:"param_key_logs_json"`
}

// LoadChainMakerConfig loads ChainMaker configuration from the specified YAML file path
func LoadChainMakerConfig(path string) (*ChainMakerConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path of ChainMaker config file: %w", err)
	}

	fmt.Printf("Loading ChainMaker configuration from '%s'...\n", absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ChainMaker config file '%s': %w", absPath, err)
	}

	var cfg ChainMakerConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ChainMaker YAML config file: %w", err)
	}

	fmt.Println("ChainMaker configuration loaded successfully.")
	return &cfg, nil
}