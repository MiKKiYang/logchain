package chainmaker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"tlng/blockchain/types"
	"tlng/config"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

// Client is the wrapper around the ChainMaker SDK client
type Client struct {
	sdkClient sdk.ChainClient
	cfg       *config.BlockchainConfig
	logger    *log.Logger
}

// NewChainMakerClient initializes the ChainMaker SDK client with the combined configuration
func NewChainMakerClient(cfg *config.BlockchainConfig, logger *log.Logger) (*Client, error) {
	logger.Println("Initializing ChainMaker SDK client using builder pattern...")

	// Extract ChainMaker-specific configuration
	chainmakerCfg, ok := cfg.ChainSpecific.(*ChainMakerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid ChainMaker configuration type")
	}

	var clientOptions []sdk.ChainClientOption
	clientOptions = append(clientOptions, sdk.WithChainClientOrgId(chainmakerCfg.OrgID))
	clientOptions = append(clientOptions, sdk.WithChainClientChainId(chainmakerCfg.ChainID))
	clientOptions = append(clientOptions, sdk.WithUserKeyFilePath(chainmakerCfg.UserKeyPath))
	clientOptions = append(clientOptions, sdk.WithUserCrtFilePath(chainmakerCfg.UserCertPath))
	clientOptions = append(clientOptions, sdk.WithUserSignKeyFilePath(chainmakerCfg.UserSignKeyPath))
	clientOptions = append(clientOptions, sdk.WithUserSignCrtFilePath(chainmakerCfg.UserSignCertPath))

	if len(chainmakerCfg.Nodes) == 0 {
		return nil, fmt.Errorf("no node configurations provided in config")
	}
	for _, nodeCfg := range chainmakerCfg.Nodes {
		if nodeCfg.UseTLS && len(nodeCfg.CaPaths) == 0 {
			return nil, fmt.Errorf("node %s has TLS enabled but no CaPaths provided", nodeCfg.Address)
		}
		sdkNodeConfig := sdk.NewNodeConfig(
			sdk.WithNodeAddr(nodeCfg.Address),
			sdk.WithNodeConnCnt(nodeCfg.ConnCount),
			sdk.WithNodeUseTLS(nodeCfg.UseTLS),
			sdk.WithNodeCAPaths(nodeCfg.CaPaths),
			sdk.WithNodeTLSHostName(nodeCfg.TLSHostName),
		)
		clientOptions = append(clientOptions, sdk.AddChainClientNodeConfig(sdkNodeConfig))
	}

	// Apply common configuration (retry, timeout, etc.)
	if cfg.RetryLimit > 0 {
		clientOptions = append(clientOptions, sdk.WithRetryLimit(cfg.RetryLimit))
	}
	if cfg.RetryInterval > 0 {
		clientOptions = append(clientOptions, sdk.WithRetryInterval(cfg.RetryInterval))
	}

	client, err := sdk.NewChainClient(clientOptions...)
	if err != nil {
		logger.Printf("Failed to build ChainMaker SDK client: %v\n", err)
		return nil, err
	}

	err = client.EnableCertHash()
	if err != nil {
		logger.Printf("Warning: Failed to enable cert hash: %v\n", err)
	}

	logger.Println("ChainMaker SDK client initialized successfully.")

	return &Client{
		sdkClient: *client,
		cfg:       cfg,
		logger:    logger,
	}, nil
}

// NewChainMakerClientFromFile initializes the ChainMaker SDK client directly from a configuration file path
func NewChainMakerClientFromFile(configPath string, logger *log.Logger) (*Client, error) {
	// Load ChainMaker-specific config
	chainmakerCfg, err := LoadChainMakerConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ChainMaker config from file '%s': %w", configPath, err)
	}

	// Create a wrapper blockchain config
	blockchainCfg := &config.BlockchainConfig{
		BlockchainType: "chainmaker",
		ChainSpecific:  chainmakerCfg,
		// Use defaults for common settings
		RetryLimit:     20,
		RetryInterval:  500,
		TimeoutSeconds: 15,
	}

	return NewChainMakerClient(blockchainCfg, logger)
}

// Config returns the configuration associated with the client.
func (c *Client) Config() any {
	if c.cfg == nil || c.cfg.ChainSpecific == nil {
		log.Println("Warning: Accessing client config before initialization.")
		return &ChainMakerConfig{} // Return empty config to avoid nil pointer panic
	}
	return c.cfg.ChainSpecific
}

// Close stops the SDK client
func (c *Client) Close() error {
	c.logger.Println("Closing ChainMaker SDK client...")
	if err := c.sdkClient.Stop(); err != nil {
		c.logger.Printf("Error stopping ChainMaker SDK client: %v", err)
		return fmt.Errorf("failed to stop ChainMaker SDK client: %w", err)
	}
	return nil
}

// SubmitLogsBatch submits a batch of logs in a single transaction
func (c *Client) SubmitLogsBatch(ctx context.Context, entries []types.LogEntry) (*types.BatchProof, []types.LogStatusInfo, error) {
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("log entry batch cannot be empty")
	}
	if c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitLogsBatchMethodName == "" || c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyLogsJson == "" {
		return nil, nil, fmt.Errorf("batch configuration fields not set in config")
	}

	// Use generic entries directly - no conversion needed
	logsJsonBytes, err := json.Marshal(entries)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal log entries to JSON: %w", err)
	}
	logsJsonStr := string(logsJsonBytes)

	kvs := []*common.KeyValuePair{
		{
			Key:   c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyLogsJson,
			Value: []byte(logsJsonStr),
		},
	}

	_, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	// c.logger.Printf("Calling contract '%s', batch method '%s' with %d entries...",
	// 	c.cfg.ChainSpecific.(*ChainMakerConfig).ContractName, c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitLogsBatchMethodName, len(entries))

	resp, err := c.sdkClient.InvokeContract(
		c.cfg.ChainSpecific.(*ChainMakerConfig).ContractName,
		c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitLogsBatchMethodName,
		"",
		kvs,
		-1,
		true,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("SDK batch invoke failed: %w", err)
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		return nil, nil, fmt.Errorf("contract batch execution failed: %s (code: %d)", resp.Message, resp.Code)
	}
	if resp.ContractResult == nil {
		return nil, nil, fmt.Errorf("contract batch execution returned nil result (tx: %s)", resp.TxId)
	}

	var results []types.LogStatusInfo
	resultJsonBytes := resp.ContractResult.Result
	if len(resultJsonBytes) == 0 {
		return nil, nil, fmt.Errorf("contract batch execution returned empty result bytes (tx: %s)", resp.TxId)
	}

	err = json.Unmarshal(resultJsonBytes, &results)
	if err != nil {
		c.logger.Printf("Failed to unmarshal batch results JSON (TxID: %s). Raw result: %s", resp.TxId, string(resultJsonBytes))
		return nil, nil, fmt.Errorf("failed to unmarshal contract batch results: %w", err)
	}

	batchProof := &types.BatchProof{
		TransactionID: resp.TxId,
		BlockHeight:   resp.TxBlockHeight,
	}

	// c.logger.Printf("Successfully processed batch submission. TxID: %s, Block: %d, Results count: %d",
	// 	batchProof.TransactionID, batchProof.BlockHeight, len(results))

	return batchProof, results, nil
}

// SubmitLog submits a single log entry
func (c *Client) SubmitLog(ctx context.Context, logHash, logContent, senderOrgID, timestamp string) (*types.Proof, error) {
	kvs := []*common.KeyValuePair{
		{Key: c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyLogHash, Value: []byte(logHash)},
		{Key: c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyLogContent, Value: []byte(logContent)},
		{Key: c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeySenderOrgID, Value: []byte(senderOrgID)},
		{Key: c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyTimestamp, Value: []byte(timestamp)},
	}
	_, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	resp, err := c.sdkClient.InvokeContract(
		c.cfg.ChainSpecific.(*ChainMakerConfig).ContractName, c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitLogMethodName, "", kvs, -1, true)
	if err != nil {
		return nil, fmt.Errorf("SDK invoke failed: %w", err)
	}
	if resp.Code != common.TxStatusCode_SUCCESS {
		return nil, fmt.Errorf("contract execution failed: %s (code: %d)", resp.Message, resp.Code)
	}
	returnedHash := string(resp.ContractResult.Result)
	if returnedHash != logHash {
		return nil, fmt.Errorf("contract returned hash '%s' does not match sent hash '%s'", returnedHash, logHash)
	}
	proof := &types.Proof{TransactionID: resp.TxId, BlockHeight: resp.TxBlockHeight, LogHash: returnedHash}
	return proof, nil
}

// FindLogByHash queries the contract for a log record by its hash
func (c *Client) FindLogByHash(ctx context.Context, logHash string) (string, error) {
	_, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	kvs := []*common.KeyValuePair{{Key: c.cfg.ChainSpecific.(*ChainMakerConfig).ParamKeyLogHash, Value: []byte(logHash)}}
	resp, err := c.sdkClient.QueryContract(c.cfg.ChainSpecific.(*ChainMakerConfig).ContractName, c.cfg.ChainSpecific.(*ChainMakerConfig).FindLogByHashMethodName, kvs, -1)
	if err != nil {
		return "", fmt.Errorf("SDK query failed: %w", err)
	}
	if resp.Code != common.TxStatusCode_SUCCESS {
		return "", fmt.Errorf("contract query failed: %s (code: %d)", resp.Message, resp.Code)
	}
	return string(resp.ContractResult.Result), nil
}

// GetLogByTxHash performs the "on-chain public audit" by querying transaction details
func (c *Client) GetLogByTxHash(ctx context.Context, txHash string) (*types.AuditData, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash cannot be empty")
	}
	txInfo, err := c.sdkClient.GetTxByTxId(txHash)
	if err != nil {
		return nil, fmt.Errorf("SDK get transaction failed: %w", err)
	}
	if txInfo == nil || txInfo.Transaction == nil || txInfo.Transaction.Result == nil || txInfo.Transaction.Result.ContractResult == nil {
		return nil, fmt.Errorf("transaction data is incomplete or nil for tx: %s", txHash)
	}
	if txInfo.Transaction.Result.Code != common.TxStatusCode_SUCCESS {
		return nil, fmt.Errorf("transaction execution failed: %s", txInfo.Transaction.Result.Message)
	}
	events := txInfo.Transaction.Result.ContractResult.ContractEvent
	for _, event := range events {
		if event.Topic == c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitEventTopic {
			eventData := event.EventData
			if len(eventData) != 3 {
				return nil, fmt.Errorf("malformed event data: expected 3 fields, got %d", len(eventData))
			}
			auditData := &types.AuditData{LogHash: eventData[0], SubmitterOrgID: eventData[1], Timestamp: eventData[2]}
			return auditData, nil
		}
	}
	return nil, fmt.Errorf("event '%s' not found in transaction %s", c.cfg.ChainSpecific.(*ChainMakerConfig).SubmitEventTopic, txHash)
}
