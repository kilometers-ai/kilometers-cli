#!/bin/bash

set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🔧 Generating Go code from protocol buffer definitions..."

# Create output directory
PROTO_DIR="internal/plugins/proto"
OUTPUT_DIR="$PROTO_DIR/generated"
mkdir -p "$OUTPUT_DIR"

# Check if protoc is available
if ! command -v protoc &> /dev/null; then
    echo "❌ Error: protoc (Protocol Buffer Compiler) is not installed"
    echo "💡 Install with: brew install protobuf (macOS) or apt-get install protobuf-compiler (Ubuntu)"
    exit 1
fi

# Check if protoc-gen-go and protoc-gen-go-grpc are available
if ! command -v protoc-gen-go &> /dev/null; then
    echo "📦 Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "📦 Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Generate Go code from proto files
echo "🏗️  Generating Go code..."
protoc \
    --go_out="$OUTPUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUTPUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    --proto_path="$PROTO_DIR" \
    "$PROTO_DIR/plugin.proto"

# Verify generated files exist
GENERATED_FILES=(
    "$OUTPUT_DIR/plugin.pb.go"
    "$OUTPUT_DIR/plugin_grpc.pb.go"
)

for file in "${GENERATED_FILES[@]}"; do
    if [[ ! -f "$file" ]]; then
        echo "❌ Error: Failed to generate $file"
        exit 1
    fi
done

echo "✅ Successfully generated Go code from protocol buffers"
echo "📁 Generated files:"
for file in "${GENERATED_FILES[@]}"; do
    echo "   - $(realpath --relative-to="$PROJECT_ROOT" "$file")"
done

echo ""
echo "🚀 Protocol buffer generation complete!"
