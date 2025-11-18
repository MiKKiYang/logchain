#!/bin/bash

# Proto Generation Script for TLNG
# This script generates Go code from protobuf definitions

set -e

echo "🔄 Generating protobuf code..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc is not installed. Please install Protocol Buffers compiler."
    echo "   Visit: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Check if Go protobuf plugins are installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "📦 Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "📦 Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Create output directory
mkdir -p proto/logingestion

# Generate Go code
echo "📝 Generating Go code from logingestion.proto..."
protoc --go_out=proto --go_opt=paths=import,module=tlng/proto \
       --go-grpc_out=proto --go-grpc_opt=paths=import,module=tlng/proto \
       proto/logingestion.proto

echo "✅ Proto generation completed successfully!"
echo "📁 Generated files:"
echo "   - proto/logingestion/logingestion.pb.go"
echo "   - proto/logingestion/logingestion_grpc.pb.go"

# Show generated files
ls -la proto/logingestion/