package blockchain

import (
	"context"
	"tlng/blockchain/types"
)

// BlockchainClient defines the generic interface for blockchain interactions
// This interface is blockchain-agnostic and can be implemented by different blockchain clients
type BlockchainClient interface {
	// SubmitLog submits a single log entry to the blockchain
	SubmitLog(ctx context.Context, logHash, logContent, senderOrgID, timestamp string) (*types.Proof, error)

	// SubmitLogsBatch submits a batch of logs in a single transaction
	SubmitLogsBatch(ctx context.Context, entries []types.LogEntry) (*types.BatchProof, []types.LogStatusInfo, error)

	// FindLogByHash queries the blockchain for a log record by its hash
	FindLogByHash(ctx context.Context, logHash string) (string, error)

	// GetLogByTxHash performs the "on-chain public audit" by querying transaction details
	GetLogByTxHash(ctx context.Context, txHash string) (*types.AuditData, error)

	// Close closes the blockchain client and releases resources
	Close() error

	// Config returns the configuration associated with the client
	Config() any // Return any to accommodate different config types
}