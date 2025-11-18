package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// KafkaConsumerConfig defines configuration for Kafka consumer
type KafkaConsumerConfig struct {
	Brokers           []string `yaml:"brokers"`             // e.g., ["kafka1:9092", "kafka2:9092"]
	Topic             string   `yaml:"topic"`               // Topic to consume from
	GroupID           string   `yaml:"group_id"`            // Consumer group ID
	Count             int      `yaml:"count"`               // Number of consumers to create
	SessionTimeout    string   `yaml:"session_timeout"`     // Kafka session timeout
	HeartbeatInterval string   `yaml:"heartbeat_interval"`  // Kafka heartbeat interval
	MaxProcessingTime string   `yaml:"max_processing_time"` // Maximum time for processing a message
	AutoOffsetReset   string   `yaml:"auto_offset_reset"`   // earliest/latest
	EnableAutoCommit  bool     `yaml:"enable_auto_commit"`  // Enable auto offset commit
}

// SetDefaults sets reasonable default values for Kafka consumer configuration
func (c *KafkaConsumerConfig) SetDefaults() {
	if c.Count <= 0 {
		c.Count = 1
		fmt.Printf("Warning: kafka_consumer.count not set or invalid, defaulting to %d\n", c.Count)
	}
	if c.SessionTimeout == "" {
		c.SessionTimeout = "30s"
		fmt.Printf("Warning: kafka_consumer.session_timeout not set, defaulting to %s\n", c.SessionTimeout)
	}
	if c.HeartbeatInterval == "" {
		c.HeartbeatInterval = "3s"
		fmt.Printf("Warning: kafka_consumer.heartbeat_interval not set, defaulting to %s\n", c.HeartbeatInterval)
	}
	if c.MaxProcessingTime == "" {
		c.MaxProcessingTime = "5m"
		fmt.Printf("Warning: kafka_consumer.max_processing_time not set, defaulting to %s\n", c.MaxProcessingTime)
	}
	if c.AutoOffsetReset == "" {
		c.AutoOffsetReset = "earliest"
		fmt.Printf("Warning: kafka_consumer.auto_offset_reset not set, defaulting to %s\n", c.AutoOffsetReset)
	}
}

// WorkerConfig defines configuration for worker processing
type WorkerConfig struct {
	Concurrency       int    `yaml:"concurrency"`        // Number of concurrent workers per consumer
	BatchSize         int    `yaml:"batch_size"`         // Number of logs per batch for blockchain
	BatchTimeout      string `yaml:"batch_timeout"`      // Maximum wait time for batch
	ConsumerRetryDelay string `yaml:"consumer_retry_delay"` // Delay when consumer encounters errors
	BlockchainTimeout string `yaml:"blockchain_timeout"` // Timeout for blockchain operations
}

// SetDefaults sets reasonable default values for worker configuration
func (c *WorkerConfig) SetDefaults() {
	if c.BatchSize <= 0 {
		c.BatchSize = 100
		fmt.Printf("Warning: worker.batch_size not set or invalid, defaulting to %d\n", c.BatchSize)
	}
	if c.BatchTimeout == "" {
		c.BatchTimeout = "1s"
		fmt.Printf("Warning: worker.batch_timeout not set, defaulting to %s\n", c.BatchTimeout)
	}
	if c.ConsumerRetryDelay == "" {
		c.ConsumerRetryDelay = "5s"
		fmt.Printf("Warning: worker.consumer_retry_delay not set, defaulting to %s\n", c.ConsumerRetryDelay)
	}
	if c.BlockchainTimeout == "" {
		c.BlockchainTimeout = "15s"
		fmt.Printf("Warning: worker.blockchain_timeout not set, defaulting to %s\n", c.BlockchainTimeout)
	}
}

// EngineMonitoringConfig defines monitoring configuration for engine
type EngineMonitoringConfig struct {
	EnableMetrics   bool   `yaml:"enable_metrics"`    // Enable metrics collection
	MetricsPath     string `yaml:"metrics_path"`      // Metrics endpoint path
	HealthCheckPath string `yaml:"health_check_path"` // Health check endpoint path
	LogLevel        string `yaml:"log_level"`         // Logging level
}

// SetDefaults sets reasonable default values for monitoring configuration
func (c *EngineMonitoringConfig) SetDefaults() {
	if c.MetricsPath == "" {
		c.MetricsPath = "/metrics"
		fmt.Printf("Warning: monitoring.metrics_path not set, defaulting to %s\n", c.MetricsPath)
	}
	if c.HealthCheckPath == "" {
		c.HealthCheckPath = "/health"
		fmt.Printf("Warning: monitoring.health_check_path not set, defaulting to %s\n", c.HealthCheckPath)
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
		fmt.Printf("Warning: monitoring.log_level not set, defaulting to %s\n", c.LogLevel)
	}
}

// EngineConfig defines all configuration for the Attestation Engine
type EngineConfig struct {
	// Database Configuration - using unified DatabaseConfig
	Database DatabaseConfig `yaml:"database"`

	// Kafka Consumer Configuration
	KafkaConsumer KafkaConsumerConfig `yaml:"kafka_consumer"`

	// Worker Configuration
	Worker WorkerConfig `yaml:"worker"`

	// Business Rules Configuration
	MaxTaskRetries int `yaml:"max_task_retries"` // Maximum retry attempts per task (business rule)

	// Monitoring Configuration
	Monitoring EngineMonitoringConfig `yaml:"monitoring"`

	// Blockchain Client Configuration
	BlockchainClientConfigPath string `yaml:"blockchain_client_config_path"`
}

// LoadEngineConfig loads configuration from the specified YAML file path
func LoadEngineConfig(path string) (*EngineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	var cfg EngineConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	// Set default values for all configurations
	cfg.Database.SetDefaults()
	cfg.KafkaConsumer.SetDefaults()
	cfg.Worker.SetDefaults()
	cfg.Monitoring.SetDefaults()

	// Set default for business rules
	if cfg.MaxTaskRetries <= 0 {
		cfg.MaxTaskRetries = 3
		fmt.Printf("Warning: max_task_retries not set or invalid, defaulting to %d\n", cfg.MaxTaskRetries)
	}

	// Validate database configuration
	if err := cfg.Database.Validate(); err != nil {
		return nil, fmt.Errorf("database configuration error: %w", err)
	}

	return &cfg, nil
}
