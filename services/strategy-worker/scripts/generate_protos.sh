#!/bin/bash

# Generate Python gRPC stubs from proto files
# This script should be run from the strategy-worker directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="${SERVICE_DIR}/../../shared/proto"
OUTPUT_DIR="${SERVICE_DIR}/src/grpc_client/generated"

echo "Generating Python gRPC stubs..."
echo "Proto dir: $PROTO_DIR"
echo "Output dir: $OUTPUT_DIR"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate stubs
python -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$OUTPUT_DIR" \
    --grpc_python_out="$OUTPUT_DIR" \
    --pyi_out="$OUTPUT_DIR" \
    "$PROTO_DIR/order.proto" \
    "$PROTO_DIR/strategy.proto"

# Create __init__.py
touch "$OUTPUT_DIR/__init__.py"

# Fix imports (grpc_tools generates relative imports)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/^import \(.*\)_pb2 as/from . import \1_pb2 as/g' "$OUTPUT_DIR"/*.py
else
    # Linux
    sed -i 's/^import \(.*\)_pb2 as/from . import \1_pb2 as/g' "$OUTPUT_DIR"/*.py
fi

echo "✓ gRPC stubs generated successfully"
