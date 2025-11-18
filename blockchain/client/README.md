# Blockchain Client Architecture

This directory contains the generic blockchain client architecture that supports multiple blockchain implementations.

## Architecture Overview

```
blockchain/
├── types/                # Unified type definitions
│   └── types.go          # Common data structures (LogEntry, Proof, etc.)
└── client/
    ├── interface.go      # Generic BlockchainClient interface
    ├── factory.go        # Factory for creating blockchain clients
    ├── chainmaker/       # ChainMaker-specific implementation
    │   └── client.go     # ChainMaker client
    └── README.md         # This file
```

## Design Principles

1. **Interface First**: All blockchain interactions go through the `BlockchainClient` interface
2. **Unified Types**: Single set of data structures used across all implementations
3. **Factory Pattern**: Clients are created through factory methods based on configuration
4. **Implementation Separation**: Each blockchain type has its own package with no type conversion
5. **Contract Control**: We ensure consistent smart contract interfaces across all blockchains
6. **Simplicity**: No unnecessary complexity or type conversions

## Type Architecture

The system uses a unified type system:

```
Generic Types (blockchain/types)     ← Used everywhere, no conversion needed
├── LogEntry
├── Proof, BatchProof
├── LogStatusInfo
└── AuditData
```

- **Single source of truth** - All implementations use the same types
- **No type conversion** - Direct usage across all blockchain implementations
- **Contract interface control** - Since we design smart contracts, we ensure consistency

## Why No Internal Types?

1. **Contract Control**: We design the smart contract interfaces across all blockchains
2. **Field Consistency**: We ensure identical field names and structures
3. **Enum Alignment**: We standardize status values across implementations
4. **Simplicity**: No unnecessary type conversions or complexity

## Usage

### Creating a Client

```go
import blockchain "tlng/blockchain/client"

// Create client from configuration file
client, err := blockchain.NewBlockchainClientFromFile("./config/blockchain.defaults.yml", logger)

// Or create client from configuration struct
client, err := blockchain.NewBlockchainClient(config, logger)
```

### Using the Client

```go
// Submit a single log
proof, err := client.SubmitLog(ctx, logHash, logContent, orgID, timestamp)

// Submit logs in batch
entries := []chainmaker.LogEntry{...}
batchProof, results, err := client.SubmitLogsBatch(ctx, entries)

// Query by hash
content, err := client.FindLogByHash(ctx, logHash)

// Get transaction details
auditData, err := client.GetLogByTxHash(ctx, txHash)
```

## Configuration

Add the blockchain type to your configuration:

```yaml
# Blockchain type selection
blockchain_type: "chainmaker"  # Options: "chainmaker", "ethereum" (future)

# ChainMaker-specific settings
chain_id: "chain1"
org_id: "wx-org1.chainmaker.org"
# ... other ChainMaker settings
```

## Adding New Blockchain Types

To add support for a new blockchain (e.g., Ethereum):

1. Create a new package: `blockchain/client/ethereum/`
2. Implement the `BlockchainClient` interface
3. Add the blockchain type constant to `factory.go`
4. Update factory method to create the new client
5. Add configuration fields if needed

### Example: Ethereum Implementation

```go
package ethereum

type EthereumClient struct {
    // Ethereum-specific fields
}

func (e *EthereumClient) SubmitLog(ctx context.Context, logHash, logContent, senderOrgID, timestamp string) (*chainmaker.Proof, error) {
    // Ethereum-specific implementation
}
// ... implement other interface methods
```

## Benefits

- **Extensibility**: Easy to add new blockchain support
- **Testability**: Interface allows for easy mocking
- **Maintainability**: Clear separation of concerns
- **Flexibility**: Runtime blockchain selection via configuration
- **Type Safety**: Compile-time checking of interface compliance

## Current Status

✅ **Implemented**:
- Generic `BlockchainClient` interface
- ChainMaker implementation
- Factory pattern for client creation
- Configuration-driven blockchain selection

❌ **Future Work**:
- Ethereum implementation
- Hyperledger Fabric implementation
- Separate configuration packages per blockchain type
- Performance benchmarks
- Client connection pooling