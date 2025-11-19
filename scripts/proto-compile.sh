#!/bin/bash

# AIPX Protobuf Compilation Script
# Generates Go and Python code from .proto files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 AIPX Protobuf Compilation"
echo "============================"
echo ""

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$PROJECT_ROOT/shared/proto"
GO_OUT_DIR="$PROJECT_ROOT/shared/go/gen"
PYTHON_OUT_DIR="$PROJECT_ROOT/shared/python/gen"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}❌ protoc is not installed${NC}"
    echo ""
    echo "Please install Protocol Buffers compiler:"
    echo "  macOS:   brew install protobuf"
    echo "  Linux:   sudo apt-get install -y protobuf-compiler"
    echo "  Windows: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${YELLOW}⚠️  protoc-gen-go not found. Installing...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${YELLOW}⚠️  protoc-gen-go-grpc not found. Installing...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

echo -e "${GREEN}✅ All prerequisites installed${NC}"
echo ""

# Create output directories
mkdir -p "$GO_OUT_DIR"
mkdir -p "$PYTHON_OUT_DIR"

# Clean previous generated files
echo "🧹 Cleaning previous generated files..."
rm -rf "$GO_OUT_DIR"/*
rm -rf "$PYTHON_OUT_DIR"/*

echo -e "${GREEN}✅ Cleaned${NC}"
echo ""

# Find all .proto files
PROTO_FILES=$(find "$PROTO_DIR" -name "*.proto")

if [ -z "$PROTO_FILES" ]; then
    echo -e "${RED}❌ No .proto files found in $PROTO_DIR${NC}"
    exit 1
fi

echo "📝 Found .proto files:"
for file in $PROTO_FILES; do
    echo "   - $(basename $file)"
done
echo ""

# Compile for Go
echo "🔨 Compiling for Go..."
for proto_file in $PROTO_FILES; do
    filename=$(basename "$proto_file")
    echo "   Compiling $filename..."

    protoc \
        --proto_path="$PROTO_DIR" \
        --go_out="$GO_OUT_DIR" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$GO_OUT_DIR" \
        --go-grpc_opt=paths=source_relative \
        "$proto_file"
done

echo -e "${GREEN}✅ Go code generated in $GO_OUT_DIR${NC}"
echo ""

# Compile for Python
echo "🐍 Compiling for Python..."
for proto_file in $PROTO_FILES; do
    filename=$(basename "$proto_file")
    echo "   Compiling $filename..."

    python3 -m grpc_tools.protoc \
        --proto_path="$PROTO_DIR" \
        --python_out="$PYTHON_OUT_DIR" \
        --grpc_python_out="$PYTHON_OUT_DIR" \
        "$proto_file"
done

# Fix Python imports (replace relative imports with absolute)
echo "🔧 Fixing Python imports..."
for py_file in "$PYTHON_OUT_DIR"/*_pb2_grpc.py; do
    if [ -f "$py_file" ]; then
        # Change: import xxx_pb2 as xxx__pb2
        # To:     from . import xxx_pb2 as xxx__pb2
        sed -i.bak 's/^import \(.*\)_pb2 as \(.*\)$/from . import \1_pb2 as \2/' "$py_file"
        rm -f "${py_file}.bak"
    fi
done

# Create __init__.py for Python package
cat > "$PYTHON_OUT_DIR/__init__.py" << 'EOF'
"""
AIPX Generated Protobuf Python Modules
"""

__all__ = [
    "market_data_pb2",
    "market_data_pb2_grpc",
    "order_pb2",
    "order_pb2_grpc",
    "user_pb2",
    "user_pb2_grpc",
    "strategy_pb2",
    "strategy_pb2_grpc",
]
EOF

echo -e "${GREEN}✅ Python code generated in $PYTHON_OUT_DIR${NC}"
echo ""

# Summary
echo "📊 Compilation Summary"
echo "======================"
GO_FILES=$(find "$GO_OUT_DIR" -name "*.go" | wc -l | tr -d ' ')
PY_FILES=$(find "$PYTHON_OUT_DIR" -name "*.py" | wc -l | tr -d ' ')

echo "   Go files generated:     $GO_FILES"
echo "   Python files generated: $PY_FILES"
echo ""

echo -e "${GREEN}✅ Protobuf compilation complete!${NC}"
echo ""
echo "Next steps:"
echo "   1. Import generated code in your services"
echo "   2. Run 'make proto' to regenerate anytime"
echo ""
