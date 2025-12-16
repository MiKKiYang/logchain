# Benthos Adapters

This directory contains Benthos configuration files and adapters for heterogeneous data source integration.

## Purpose

According to the design document, these adapters handle protocol conversion for non-HTTP/gRPC data sources:

- **Syslog** - UDP/TCP syslog protocol (port 514)
- **Kafka Topics** - Direct Kafka topic consumption
- **S3** - AWS S3 bucket file processing
- **Other protocols** - Any future heterogeneous data sources

## Architecture

These adapters work with the API Gateway to:
1. Receive heterogeneous protocol traffic
2. Parse and standardize data formats
3. Forward processed logs to the Log Ingestion Service

## Implementation Status

✅ **Benthos 适配器配置已提供**

本目录现在包含可直接运行的 Benthos 配置，支持将异构数据源转换为统一的 `log_content` JSON，并转发到日志接入服务（默认 `http://ingestion:8091/v1/logs`）。

## Configuration Files

当前配置文件：
- `syslog.yml` - Syslog 适配器（UDP/TCP 514 口）
- `kafka-consumer.yml` - Kafka Topic 适配器
- `s3-processor.yml` - S3 文件适配器（按行切分）

## 使用方法

### 环境变量（通用）
- `INGESTION_ENDPOINT`: 日志接入服务地址，默认 `http://ingestion:8091/v1/logs`
- `INGESTION_API_KEY`: 可选，若接入服务启用 API Key 则填写
- `DEFAULT_ORG_ID`: 写入 `client_source_org_id` 的默认值
- `HTTP_BATCH_COUNT` / `HTTP_BATCH_PERIOD`: HTTP 批量大小与时间窗口

### Syslog 适配器
```bash
export SYSLOG_UDP_ADDR=0.0.0.0:5514
export SYSLOG_TCP_ADDR=0.0.0.0:5514
export DEFAULT_ORG_ID= 
export INGESTION_ENDPOINT= 
export INGESTION_API_KEY= 
export HTTP_BATCH_COUNT= 
export HTTP_BATCH_PERIOD= 

./redpanda-connect lint ingestion/adapters/syslog.yml
./redpanda-connect run ingestion/adapters/syslog.yml
```

### Kafka 适配器
```bash
export KAFKA_BROKERS=broker:9092
export KAFKA_TOPIC=logs.raw
export KAFKA_CONSUMER_GROUP=benthos-adapter
export KAFKA_CLIENT_ID=
export DEFAULT_ORG_ID=
export INGESTION_ENDPOINT=
export INGESTION_API_KEY=
export HTTP_BATCH_COUNT=
export HTTP_BATCH_PERIOD=

./redpanda-connect lint ingestion/adapters/kafka-consumer.yml
./redpanda-connect run ingestion/adapters/kafka-consumer.yml
```

### S3 适配器
```bash
export S3_BUCKET_NAME=
export S3_PREFIX=
export AWS_REGION=
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=
export AWS_SESSION_TOKEN=
export S3_DELETE_AFTER_READ=
export DEFAULT_ORG_ID=
export INGESTION_ENDPOINT=
export INGESTION_API_KEY=
export HTTP_BATCH_COUNT=
export HTTP_BATCH_PERIOD=
  
./redpanda-connect lint ingestion/adapters/s3-processor.yml
./redpanda-connect run ingestion/adapters/s3-processor.yml
```