package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// KafkaProducerConfig defines configuration for Kafka producer
type KafkaProducerConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`

	// Batch processing settings
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	BatchBytes   int           `yaml:"batch_bytes"`

	// Reliability settings
	RequiredAcks string `yaml:"required_acks"`
	Async        bool   `yaml:"async"`

	// Performance settings
	WriteTimeout time.Duration `yaml:"write_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
}

// BatchProcessorConfig defines configuration for batch processing
type BatchProcessorConfig struct {
	BatchSize           int           `yaml:"batch_size"`
	BatchTimeout        time.Duration `yaml:"batch_timeout"`
	MaxBufferSize       int           `yaml:"max_buffer_size"`
	FlushChannelBuffer  int           `yaml:"flush_channel_buffer"`  // Buffer size for flush channel
}

// SetDefaults sets reasonable default values for batch processor configuration
func (c *BatchProcessorConfig) SetDefaults() {
	if c.BatchSize == 0 {
		c.BatchSize = 100
		fmt.Printf("Warning: batch_processor.batch_size not set, defaulting to %d\n", c.BatchSize)
	}
	if c.BatchTimeout == 0 {
		c.BatchTimeout = 100 * time.Millisecond
		fmt.Printf("Warning: batch_processor.batch_timeout not set, defaulting to %v\n", c.BatchTimeout)
	}
	if c.MaxBufferSize == 0 {
		c.MaxBufferSize = 10000
		fmt.Printf("Warning: batch_processor.max_buffer_size not set, defaulting to %d\n", c.MaxBufferSize)
	}
	if c.FlushChannelBuffer == 0 {
		c.FlushChannelBuffer = 100
		fmt.Printf("Warning: batch_processor.flush_channel_buffer not set, defaulting to %d\n", c.FlushChannelBuffer)
	}
}


// HttpServerConfig defines HTTP server configuration
type HttpServerConfig struct {
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// GatewayMonitoringConfig defines monitoring configuration for API gateway
type GatewayMonitoringConfig struct {
	EnableMetrics   bool   `yaml:"enable_metrics"`
	MetricsPath     string `yaml:"metrics_path"`
	HealthCheckPath string `yaml:"health_check_path"`
}

// ApiGatewayConfig defines all configurations required for the API gateway
type ApiGatewayConfig struct {
	HttpListenAddr string `yaml:"http_listen_addr"`
	GrpcListenAddr string `yaml:"grpc_listen_addr"`

	Database       DatabaseConfig       `yaml:"database"`       // Use unified DatabaseConfig
	KafkaProducer  KafkaProducerConfig  `yaml:"kafka_producer"` // Local Kafka producer config
	BatchProcessor BatchProcessorConfig `yaml:"batch_processor"`
	HttpServer     HttpServerConfig     `yaml:"http_server"`
	Monitoring     GatewayMonitoringConfig     `yaml:"monitoring"`
}

// LoadApiGatewayConfig loads API gateway configuration from the specified YAML file path
func LoadApiGatewayConfig(path string) (*ApiGatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read API Gateway config file '%s': %w", path, err)
	}

	var cfg ApiGatewayConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse API Gateway YAML config file: %w", err)
	}

	// Set defaults for database configuration
	cfg.Database.SetDefaults()

	// Set defaults for batch processor configuration
	cfg.BatchProcessor.SetDefaults()

	// Validation
	if cfg.HttpListenAddr == "" && cfg.GrpcListenAddr == "" {
		return nil, fmt.Errorf("configuration error: at least one of http_listen_addr or grpc_listen_addr must be configured")
	}

	// Validate database configuration
	if err := cfg.Database.Validate(); err != nil {
		return nil, fmt.Errorf("database configuration error: %w", err)
	}

	return &cfg, nil
}
