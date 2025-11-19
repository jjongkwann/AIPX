#!/bin/bash

# AIPX Protobuf Compilation Script
# Generates Go and Python code from .proto files

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🚀 AIPX Protobuf Compilation"
echo "============================"
echo ""

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="$PROJECT_ROOT/shared/proto"
GO_OUT_DIR="$PROJECT_ROOT/shared/go/pkg/pb"
PYTHON_OUT_DIR="$PROJECT_ROOT/shared/python/common/pb"

# Track if we should skip Python compilation
SKIP_PYTHON=false

# Function to print colored messages
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    print_error "protoc is not installed"
    echo ""
    echo "Please install Protocol Buffers compiler:"
    echo "  macOS:   brew install protobuf"
    echo "  Linux:   sudo apt-get install -y protobuf-compiler"
    echo "  Windows: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

print_success "protoc found: $(protoc --version)"

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    print_error "protoc-gen-go not found"
    echo ""
    echo "Install via: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

print_success "protoc-gen-go found"

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    print_error "protoc-gen-go-grpc not found"
    echo ""
    echo "Install via: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

print_success "protoc-gen-go-grpc found"

# Check if Python grpc_tools is installed
if ! python3 -c "import grpc_tools" 2>/dev/null; then
    print_warning "grpcio-tools not found for Python"
    print_info "Install via: pip install grpcio-tools"
    print_info "Skipping Python compilation..."
    SKIP_PYTHON=true
else
    print_success "grpcio-tools found for Python"
fi

echo ""

# Create output directories
mkdir -p "$GO_OUT_DIR"
mkdir -p "$PYTHON_OUT_DIR"

# Clean previous generated files
echo "🧹 Cleaning previous generated files..."
if [ -d "$GO_OUT_DIR" ]; then
    find "$GO_OUT_DIR" -name "*.pb.go" -delete 2>/dev/null || true
    find "$GO_OUT_DIR" -name "*_grpc.pb.go" -delete 2>/dev/null || true
fi

if [ -d "$PYTHON_OUT_DIR" ]; then
    find "$PYTHON_OUT_DIR" -name "*_pb2.py" -delete 2>/dev/null || true
    find "$PYTHON_OUT_DIR" -name "*_pb2_grpc.py" -delete 2>/dev/null || true
    find "$PYTHON_OUT_DIR" -name "*.pyc" -delete 2>/dev/null || true
fi

print_success "Cleaned old generated files"
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
GO_FAILED=false

for proto_file in $PROTO_FILES; do
    filename=$(basename "$proto_file")
    echo "   Compiling $filename..."

    if protoc \
        --proto_path="$PROTO_DIR" \
        --go_out="$GO_OUT_DIR" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$GO_OUT_DIR" \
        --go-grpc_opt=paths=source_relative \
        "$proto_file" 2>&1; then
        print_success "   $filename compiled successfully"
    else
        print_error "   Failed to compile $filename"
        GO_FAILED=true
    fi
done

if [ "$GO_FAILED" = true ]; then
    print_error "Some Go compilations failed"
    exit 1
fi

print_success "Go code generated in $GO_OUT_DIR"
echo ""

# Compile for Python
if [ "$SKIP_PYTHON" = false ]; then
    echo "🐍 Compiling for Python..."
    PYTHON_FAILED=false

    for proto_file in $PROTO_FILES; do
        filename=$(basename "$proto_file")
        echo "   Compiling $filename..."

        if python3 -m grpc_tools.protoc \
            --proto_path="$PROTO_DIR" \
            --python_out="$PYTHON_OUT_DIR" \
            --grpc_python_out="$PYTHON_OUT_DIR" \
            "$proto_file" 2>&1; then
            print_success "   $filename compiled successfully"
        else
            print_error "   Failed to compile $filename"
            PYTHON_FAILED=true
        fi
    done

    if [ "$PYTHON_FAILED" = true ]; then
        print_error "Some Python compilations failed"
        exit 1
    fi

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

    print_success "Python code generated in $PYTHON_OUT_DIR"
    echo ""
else
    print_warning "Python compilation skipped (grpcio-tools not installed)"
    echo ""
fi

# Summary
echo "📊 Compilation Summary"
echo "======================"

if [ -d "$GO_OUT_DIR" ]; then
    GO_FILES=$(find "$GO_OUT_DIR" -name "*.go" 2>/dev/null | wc -l | tr -d ' ')
    echo "   Go files generated:     $GO_FILES"
    echo ""
    print_info "Generated Go files:"
    find "$GO_OUT_DIR" -name "*.go" 2>/dev/null | while read -r file; do
        echo "      - ${file#$PROJECT_ROOT/}"
    done
fi

echo ""

if [ "$SKIP_PYTHON" = false ] && [ -d "$PYTHON_OUT_DIR" ]; then
    PY_FILES=$(find "$PYTHON_OUT_DIR" -name "*_pb2*.py" 2>/dev/null | wc -l | tr -d ' ')
    echo "   Python files generated: $PY_FILES"
    echo ""
    print_info "Generated Python files:"
    find "$PYTHON_OUT_DIR" -name "*_pb2*.py" 2>/dev/null | while read -r file; do
        echo "      - ${file#$PROJECT_ROOT/}"
    done
fi

echo ""
print_success "Protobuf compilation complete!"
echo ""
print_info "Next steps:"
echo "   1. Import Go code: import \"github.com/jjongkwann/aipx/shared/go/pkg/pb\""
echo "   2. Import Python code: from common.pb import <module>_pb2"
echo "   3. Run './scripts/proto-compile.sh' to regenerate anytime"
echo ""
