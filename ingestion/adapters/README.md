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

🚧 **TODO**: Not yet implemented

This directory is prepared for future Benthos adapter implementations.

## Configuration Files

When implemented, this directory will contain:
- `syslog.yml` - Syslog adapter configuration
- `kafka-consumer.yml` - Kafka topic adapter configuration
- `s3-processor.yml` - S3 file adapter configuration
- `common/` - Shared configuration fragments