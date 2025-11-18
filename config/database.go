package config

import "fmt"

// DatabaseConfig defines the unified database configuration structure
// This is used by both API Gateway and Engine services
type DatabaseConfig struct {
	DSN            string `yaml:"dsn" json:"dsn"`                     // PostgreSQL connection string
	MaxConnections int    `yaml:"max_connections" json:"max_connections"` // Maximum number of connections
	MinConnections int    `yaml:"min_connections" json:"min_connections"` // Minimum number of connections
	MaxIdleTime    string `yaml:"max_idle_time" json:"max_idle_time"`     // Maximum time a connection can be idle
	MaxLifetime    string `yaml:"max_lifetime" json:"max_lifetime"`       // Maximum lifetime of a connection
}

// SetDefaults sets sensible default values for the database configuration
func (c *DatabaseConfig) SetDefaults() {
	if c.MaxConnections <= 0 {
		c.MaxConnections = 50
		fmt.Printf("Warning: database.max_connections not set or invalid, defaulting to %d\n", c.MaxConnections)
	}
	if c.MinConnections <= 0 {
		c.MinConnections = 10
		fmt.Printf("Warning: database.min_connections not set or invalid, defaulting to %d\n", c.MinConnections)
	}
	if c.MaxIdleTime == "" {
		c.MaxIdleTime = "1h"
		fmt.Printf("Warning: database.max_idle_time not set, defaulting to %s\n", c.MaxIdleTime)
	}
	if c.MaxLifetime == "" {
		c.MaxLifetime = "24h"
		fmt.Printf("Warning: database.max_lifetime not set, defaulting to %s\n", c.MaxLifetime)
	}
}

// Validate validates the database configuration
func (c *DatabaseConfig) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}
	if c.MaxConnections <= 0 {
		return fmt.Errorf("database max_connections must be positive")
	}
	if c.MinConnections < 0 {
		return fmt.Errorf("database min_connections cannot be negative")
	}
	if c.MinConnections > c.MaxConnections {
		return fmt.Errorf("database min_connections (%d) cannot be greater than max_connections (%d)",
			c.MinConnections, c.MaxConnections)
	}
	return nil
}

// LogConfiguration logs the database configuration (excluding sensitive DSN)
func (c *DatabaseConfig) LogConfiguration() {
	fmt.Printf("Database Configuration:\n")
	fmt.Printf("  Max Connections: %d\n", c.MaxConnections)
	fmt.Printf("  Min Connections: %d\n", c.MinConnections)
	fmt.Printf("  Max Idle Time: %s\n", c.MaxIdleTime)
	fmt.Printf("  Max Lifetime: %s\n", c.MaxLifetime)
	fmt.Printf("  DSN: [configured]\n") // Don't log the actual DSN for security
}