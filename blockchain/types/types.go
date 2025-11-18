package types

// LogEntry corresponds to the struct sent in the batch JSON
// This is a generic type that can be implemented by any blockchain
type LogEntry struct {
	LogHash     string `json:"log_hash"`
	LogContent  string `json:"log_content"`
	SenderOrgID string `json:"sender_org_id"`
	Timestamp   string `json:"timestamp"`
}

// LogProcessingStatus corresponds to the Rust enum for batch results
type LogProcessingStatus string

const (
	StatusSuccess          LogProcessingStatus = "Success"
	StatusSkippedDuplicate LogProcessingStatus = "SkippedDuplicate"
	StatusErrorValidation  LogProcessingStatus = "ErrorValidation"
	StatusErrorStateCheck  LogProcessingStatus = "ErrorStateCheck"
	StatusErrorPutState    LogProcessingStatus = "ErrorPutState"
)

// LogStatusInfo corresponds to the struct returned in the batch result JSON array
type LogStatusInfo struct {
	LogHash string              `json:"log_hash"`
	Status  LogProcessingStatus `json:"status"`
	Message string              `json:"message"`
}

// BatchProof holds the results common to the entire batch transaction
type BatchProof struct {
	TransactionID string // The TxID for the single batch transaction
	BlockHeight   uint64 // The block height where the batch was included
}

// Proof is the on-chain credential returned after successful single SubmitLog
type Proof struct {
	TransactionID string
	BlockHeight   uint64
	LogHash       string
}

// AuditData is the raw notarization data parsed from on-chain events
type AuditData struct {
	LogHash        string
	SubmitterOrgID string
	Timestamp      string
}