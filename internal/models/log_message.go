package models

// LogMessage defines the message structure for log submissions
// Used across ingestion, processing, and messaging layers
type LogMessage struct {
	RequestID         string `json:"RequestID"`
	LogContent        string `json:"LogContent"`
	LogHash           string `json:"LogHash"`
	SourceOrgID       string `json:"SourceOrgID"`
	ReceivedTimestamp string `json:"ReceivedTimestamp"` // Use string for easy JSON serialization
}