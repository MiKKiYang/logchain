### Rust Version
```Rust
//! # Log Store Contract
//!
//! A ChainMaker smart contract written in Rust for the Trusted Log Attestation System.
//! This version updates `submit_logs_batch` to return a detailed status list for each log entry.

use contract_sdk_rust::sim_context;
use contract_sdk_rust::sim_context::SimContext;
use serde::{Deserialize, Serialize};

// === Contract Constants ===
const NAMESPACE: &str = "log_store_v1";
const KEY_PREFIX: &str = "log_";
const EVENT_TOPIC_LOG_SUBMITTED: &str = "log_submitted";

// === Helper Structures ===

/// Defines the structure of each log object in the JSON array for batch submission
#[derive(Serialize, Deserialize, Debug)]
struct LogEntry {
    log_hash: String,
    log_content: String,
    sender_org_id: String,
    timestamp: String,
}

/// Defines the processing status enum for a single log entry
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
enum LogProcessingStatus {
    Success,          // Successfully processed and stored on-chain
    SkippedDuplicate, // Skipped due to existing hash
    ErrorValidation,  // Input parameter validation failed (e.g., empty fields)
    ErrorStateCheck,  // Error occurred while checking state database (get_state)
    ErrorPutState,    // Error occurred while writing to state database (put_state)
}

/// Defines the processing result structure returned to the caller for a single log entry
#[derive(Serialize, Deserialize, Debug)]
struct LogStatusInfo {
    log_hash: String,
    status: LogProcessingStatus,
    message: String,
}

// === Required Entry Functions ===

#[no_mangle]
pub extern "C" fn init_contract() {
    let ctx = &mut sim_context::get_sim_context();
    ctx.log("LogStoreContract (Rust V3) initialized successfully.");
    ctx.ok("init success".as_bytes());
}

#[no_mangle]
pub extern "C" fn upgrade() {
    let ctx = &mut sim_context::get_sim_context();
    ctx.log("LogStoreContract (Rust V3) upgraded successfully.");
    ctx.ok("upgrade success".as_bytes());
}

// === Core Business Methods ===

/// Batch submit logs and return detailed status list
#[no_mangle]
pub extern "C" fn submit_logs_batch() {
    let ctx = &mut sim_context::get_sim_context();
    ctx.log("Executing submit_logs_batch (V3)...");

    // Get JSON parameter
    let logs_json_str = ctx.arg_as_utf8_str("logs_json");
    if logs_json_str.is_empty() {
        ctx.error("Missing required argument: logs_json");
        return;
    }

    // Parse JSON
    let log_entries: Result<Vec<LogEntry>, serde_json::Error> = serde_json::from_str(&logs_json_str);
    let entries = match log_entries {
        Ok(entries) => {
            if entries.is_empty() {
                ctx.error("logs_json array cannot be empty");
                return;
            }
            entries
        },
        Err(e) => {
            let err_msg = format!("Failed to parse logs_json: {}", e);
            ctx.error(&err_msg);
            return;
        }
    };

    // Process each log entry and collect status
    let mut results: Vec<LogStatusInfo> = Vec::with_capacity(entries.len());

    for entry in entries {
        let mut current_status = LogProcessingStatus::Success;
        let mut message = String::from("Processed successfully");

        // Validate input for single log entry
        if entry.log_hash.is_empty() || entry.log_content.is_empty() || entry.sender_org_id.is_empty() || entry.timestamp.is_empty() {
            current_status = LogProcessingStatus::ErrorValidation;
            message = "Skipped due to empty fields".to_string();
            ctx.log(&format!("Validation Error for hash '{}': {}", entry.log_hash, message));
        } else {
            // Construct storage key and check state
            let storage_key = format!("{}{}", KEY_PREFIX, entry.log_hash);
            match ctx.get_state(NAMESPACE, &storage_key) {
                Ok(value) => {
                    if !value.is_empty() {
                        current_status = LogProcessingStatus::SkippedDuplicate;
                        message = "Skipped duplicate log hash".to_string();
                        ctx.log(&format!("Duplicate found for hash '{}'", entry.log_hash));
                    }
                },
                Err(code) => {
                    current_status = LogProcessingStatus::ErrorStateCheck;
                    message = format!("Failed to check state, error code: {}", code);
                    ctx.log(&format!("State check error for hash '{}': {}", entry.log_hash, message));
                }
            }
        }

        // Only execute write and event if status is still Success
        if current_status == LogProcessingStatus::Success {
            let storage_value = format!(
                "org_id={}&ts={}&content={}",
                entry.sender_org_id, entry.timestamp, entry.log_content
            );

            ctx.put_state(NAMESPACE, &format!("{}{}", KEY_PREFIX, entry.log_hash), storage_value.as_bytes());

            let event_data = vec![
                entry.log_hash.clone(),
                entry.sender_org_id.clone(),
                entry.timestamp.clone(),
            ];
            ctx.emit_event(EVENT_TOPIC_LOG_SUBMITTED, &event_data);

            ctx.log(&format!("Successfully processed log hash: {}", entry.log_hash));
        }

        // Record processing result
        results.push(LogStatusInfo {
            log_hash: entry.log_hash.clone(),
            status: current_status,
            message,
        });
    }

    // Serialize result list to JSON string
    let result_json = match serde_json::to_string(&results) {
        Ok(json_str) => json_str,
        Err(e) => {
            let err_msg = format!("Failed to serialize results to JSON: {}", e);
            ctx.error(&err_msg);
            return;
        }
    };

    // Return JSON string containing detailed status
    ctx.log(&format!("Batch processing finished. Result JSON length: {}", result_json.len()));
    ctx.ok(result_json.as_bytes());
}


/// Core write method for single log entry
#[no_mangle]
pub extern "C" fn submit_log() {
    let ctx = &mut sim_context::get_sim_context();
    let log_hash = ctx.arg_as_utf8_str("log_hash");
    let log_content = ctx.arg_as_utf8_str("log_content");
    let sender_org_id = ctx.arg_as_utf8_str("sender_org_id");
    let timestamp = ctx.arg_as_utf8_str("timestamp");

    if log_hash.is_empty() || log_content.is_empty() || sender_org_id.is_empty() || timestamp.is_empty() {
        ctx.error("Missing required arguments: log_hash, log_content, sender_org_id, timestamp");
        return;
    }

    let storage_key = format!("{}{}", KEY_PREFIX, log_hash);
    match ctx.get_state(NAMESPACE, &storage_key) {
        Ok(value) => {
            if !value.is_empty() {
                ctx.error("Log with this hash already exists.");
                return;
            }
        },
        Err(_) => {
            ctx.error("Failed to check existing state for log hash.");
            return;
        }
    }

    let storage_value = format!("org_id={}&ts={}&content={}", sender_org_id, timestamp, log_content);
    ctx.put_state(NAMESPACE, &storage_key, storage_value.as_bytes());

    let event_data = vec![
        log_hash.clone(),
        sender_org_id.clone(),
        timestamp.clone(),
    ];
    ctx.emit_event(EVENT_TOPIC_LOG_SUBMITTED, &event_data);

    ctx.log(&format!("Successfully submitted log. Hash: {}", log_hash));
    ctx.ok(log_hash.as_bytes());
}

/// Read-only method to query complete log record by hash
#[no_mangle]
pub extern "C" fn find_log_by_hash() {
    let ctx = &mut sim_context::get_sim_context();
    let log_hash = ctx.arg_as_utf8_str("log_hash");
    if log_hash.is_empty() {
        ctx.error("Missing required argument: log_hash");
        return;
    }

    let storage_key = format!("{}{}", KEY_PREFIX, log_hash);
    match ctx.get_state(NAMESPACE, &storage_key) {
        Ok(value) => {
            if value.is_empty() {
                ctx.log(&format!("Log not found for hash: {}", log_hash));
                ctx.ok("".as_bytes());
            } else {
                ctx.ok(&value);
            }
        },
        Err(code) => {
            let msg = format!("Failed to get log from state, error code: {}", code);
            ctx.error(&msg);
        }
    }
}
```

### Go Version
```Go
// Package main - Log Store Contract
//
// A ChainMaker smart contract written in Go for the Trusted Log Attestation System.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sandbox"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
)

// === Contract Constants ===
const (
	Namespace              = "log_store_v1"
	KeyPrefix              = "log_"
	EventTopicLogSubmitted = "log_submitted"
)

// === Helper Structures ===

// LogEntry defines the structure of each log object in the JSON array for batch submission
type LogEntry struct {
	LogHash     string `json:"log_hash"`
	LogContent  string `json:"log_content"`
	SenderOrgID string `json:"sender_org_id"`
	Timestamp   string `json:"timestamp"`
}

// LogProcessingStatus defines the processing status enum for a single log entry
type LogProcessingStatus string

const (
	StatusSuccess          LogProcessingStatus = "Success"          // Successfully processed and stored on-chain
	StatusSkippedDuplicate LogProcessingStatus = "SkippedDuplicate" // Skipped due to existing hash
	StatusErrorValidation  LogProcessingStatus = "ErrorValidation"  // Input parameter validation failed
	StatusErrorStateCheck  LogProcessingStatus = "ErrorStateCheck"  // Error occurred while checking state database
	StatusErrorPutState    LogProcessingStatus = "ErrorPutState"    // Error occurred while writing to state database
)

// LogStatusInfo defines the processing result structure returned to the caller for a single log entry
type LogStatusInfo struct {
	LogHash string              `json:"log_hash"` // Corresponding log hash
	Status  LogProcessingStatus `json:"status"`   // Processing status
	Message string              `json:"message"`  // Additional information (e.g., error reason)
}

// === Contract Structure ===

// LogStoreContract is the main contract structure
type LogStoreContract struct {
}

// === Required Entry Functions ===

// InitContract initializes the contract
func (c *LogStoreContract) InitContract() protogo.Response {
	sdk.Instance.Infof("LogStoreContract (Go V3) initialized successfully.")
	return sdk.Success([]byte("init success"))
}

// UpgradeContract upgrades the contract
func (c *LogStoreContract) UpgradeContract() protogo.Response {
	sdk.Instance.Infof("LogStoreContract (Go V3) upgraded successfully.")
	return sdk.Success([]byte("upgrade success"))
}

// InvokeContract routes contract method calls
func (c *LogStoreContract) InvokeContract(method string) protogo.Response {
	switch method {
	case "submit_logs_batch":
		return c.submitLogsBatch()
	case "submit_log":
		return c.submitLog()
	case "find_log_by_hash":
		return c.findLogByHash()
	default:
		return sdk.Error("invalid method: " + method)
	}
}

// === Core Business Methods ===

// submitLogsBatch batch submit logs, returns detailed status list
func (c *LogStoreContract) submitLogsBatch() protogo.Response {
	sdk.Instance.Infof("Executing submit_logs_batch (V3)...")

	// Get JSON parameter
	logsJSON, ok := sdk.Instance.GetArgs()["logs_json"]
	if !ok || len(logsJSON) == 0 {
		return sdk.Error("Missing required argument: logs_json")
	}

	// Parse JSON
	var entries []LogEntry
	if err := json.Unmarshal(logsJSON, &entries); err != nil {
		errMsg := fmt.Sprintf("Failed to parse logs_json: %v", err)
		return sdk.Error(errMsg)
	}

	if len(entries) == 0 {
		return sdk.Error("logs_json array cannot be empty")
	}

	// Process each log entry and collect status
	results := make([]LogStatusInfo, 0, len(entries))

	for _, entry := range entries {
		currentStatus := StatusSuccess
		message := "Processed successfully"

		// Validate input for single log entry
		if entry.LogHash == "" || entry.LogContent == "" || entry.SenderOrgID == "" || entry.Timestamp == "" {
			currentStatus = StatusErrorValidation
			message = "Skipped due to empty fields"
			sdk.Instance.Infof("Validation Error for hash '%s': %s", entry.LogHash, message)
		} else {
			// Construct storage key and check state
			storageKey := KeyPrefix + entry.LogHash
			value, err := sdk.Instance.GetState(Namespace, storageKey)
			if err != nil {
				// Error during get_state
				currentStatus = StatusErrorStateCheck
				message = fmt.Sprintf("Failed to check state: %v", err)
				sdk.Instance.Infof("State check error for hash '%s': %s", entry.LogHash, message)
			} else if len(value) > 0 {
				// Hash already exists
				currentStatus = StatusSkippedDuplicate
				message = "Skipped duplicate log hash"
				sdk.Instance.Infof("Duplicate found for hash '%s'", entry.LogHash)
			} else {
				// Only execute write and event if status is still Success
				storageValue := fmt.Sprintf("org_id=%s&ts=%s&content=%s",
					entry.SenderOrgID, entry.Timestamp, entry.LogContent)

				// Write to state database
				if err := sdk.Instance.PutState(Namespace, storageKey, []byte(storageValue)); err != nil {
					currentStatus = StatusErrorPutState
					message = fmt.Sprintf("Failed to put state: %v", err)
					sdk.Instance.Infof("Put state error for hash '%s': %s", entry.LogHash, message)
				} else {
					// Emit single event
					eventData := []string{
						entry.LogHash,
						entry.SenderOrgID,
						entry.Timestamp,
					}
					sdk.Instance.EmitEvent(EventTopicLogSubmitted, eventData)
					sdk.Instance.Infof("Successfully processed log hash: %s", entry.LogHash)
				}
			}
		}

		// Record processing result
		results = append(results, LogStatusInfo{
			LogHash: entry.LogHash,
			Status:  currentStatus,
			Message: message,
		})
	}

	// Serialize result list to JSON string
	resultJSON, err := json.Marshal(results)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to serialize results to JSON: %v", err)
		return sdk.Error(errMsg)
	}

	// Return JSON string containing detailed status
	sdk.Instance.Infof("Batch processing finished. Result JSON length: %d", len(resultJSON))
	return sdk.Success(resultJSON)
}

// submitLog core write method for single log entry
func (c *LogStoreContract) submitLog() protogo.Response {
	args := sdk.Instance.GetArgs()

	logHash, _ := args["log_hash"]
	logContent, _ := args["log_content"]
	senderOrgID, _ := args["sender_org_id"]
	timestamp, _ := args["timestamp"]

	if len(logHash) == 0 || len(logContent) == 0 || len(senderOrgID) == 0 || len(timestamp) == 0 {
		return sdk.Error("Missing required arguments: log_hash, log_content, sender_org_id, timestamp")
	}

	storageKey := KeyPrefix + string(logHash)
	value, err := sdk.Instance.GetState(Namespace, storageKey)
	if err != nil {
		return sdk.Error("Failed to check existing state for log hash")
	}
	if len(value) > 0 {
		return sdk.Error("Log with this hash already exists")
	}

	storageValue := fmt.Sprintf("org_id=%s&ts=%s&content=%s",
		string(senderOrgID), string(timestamp), string(logContent))

	if err := sdk.Instance.PutState(Namespace, storageKey, []byte(storageValue)); err != nil {
		return sdk.Error(fmt.Sprintf("Failed to put state: %v", err))
	}

	eventData := []string{
		string(logHash),
		string(senderOrgID),
		string(timestamp),
	}
	sdk.Instance.EmitEvent(EventTopicLogSubmitted, eventData)

	sdk.Instance.Infof("Successfully submitted log. Hash: %s", string(logHash))
	return sdk.Success(logHash)
}

// findLogByHash read-only method to query complete log record by hash
func (c *LogStoreContract) findLogByHash() protogo.Response {
	args := sdk.Instance.GetArgs()
	logHash, ok := args["log_hash"]

	if !ok || len(logHash) == 0 {
		return sdk.Error("Missing required argument: log_hash")
	}

	storageKey := KeyPrefix + string(logHash)
	value, err := sdk.Instance.GetState(Namespace, storageKey)
	if err != nil {
		return sdk.Error(fmt.Sprintf("Failed to get log from state: %v", err))
	}

	if len(value) == 0 {
		sdk.Instance.Infof("Log not found for hash: %s", string(logHash))
		return sdk.Success([]byte(""))
	}

	return sdk.Success(value)
}

// === Main Function (Required) ===
func main() {
	err := sandbox.Start(new(LogStoreContract))
	if err != nil {
		log.Fatal(err)
	}
}
```