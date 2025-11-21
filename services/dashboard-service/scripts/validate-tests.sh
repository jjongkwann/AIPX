#!/bin/bash

# Dashboard Service Test Validation Script
# This script validates that all test files are properly configured

set -e

echo "🧪 Validating Dashboard Service Test Suite"
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counter for checks
PASSED=0
FAILED=0

check_file() {
    if [ -f "$1" ]; then
        echo -e "${GREEN}✓${NC} $1"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} $1 (missing)"
        ((FAILED++))
    fi
}

check_dir() {
    if [ -d "$1" ]; then
        echo -e "${GREEN}✓${NC} $1"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} $1 (missing)"
        ((FAILED++))
    fi
}

echo "📁 Checking test directories..."
check_dir "src/__tests__/components"
check_dir "src/__tests__/pages"
check_dir "tests/integration"
check_dir "tests/e2e"
check_dir "tests/a11y"
check_dir "tests/performance"
check_dir "tests/visual"
check_dir "tests/mocks"
check_dir "tests/utils"
echo ""

echo "📄 Checking configuration files..."
check_file "jest.config.js"
check_file "playwright.config.ts"
check_file "tests/setup.ts"
check_file ".github/workflows/test.yml"
echo ""

echo "🧩 Checking component tests..."
check_file "src/__tests__/components/TradingViewChart.test.tsx"
check_file "src/__tests__/components/ChatInterface.test.tsx"
check_file "src/__tests__/components/Navbar.test.tsx"
check_file "src/__tests__/components/BacktestResults.test.tsx"
check_file "src/__tests__/components/PortfolioTable.test.tsx"
echo ""

echo "📱 Checking page tests..."
check_file "src/__tests__/pages/login.test.tsx"
echo ""

echo "🔗 Checking integration tests..."
check_file "tests/integration/test_api_client.test.ts"
echo ""

echo "🎭 Checking E2E tests..."
check_file "tests/e2e/auth.spec.ts"
check_file "tests/e2e/chat.spec.ts"
check_file "tests/e2e/portfolio.spec.ts"
check_file "tests/e2e/backtest.spec.ts"
check_file "tests/e2e/navigation.spec.ts"
echo ""

echo "♿ Checking accessibility tests..."
check_file "tests/a11y/wcag.spec.ts"
echo ""

echo "⚡ Checking performance tests..."
check_file "tests/performance/load.spec.ts"
echo ""

echo "📸 Checking visual regression tests..."
check_file "tests/visual/screenshots.spec.ts"
echo ""

echo "🛠️ Checking test utilities..."
check_file "tests/utils/test-utils.tsx"
check_file "tests/mocks/data.ts"
check_file "tests/mocks/handlers.ts"
check_file "tests/mocks/server.ts"
echo ""

echo "📚 Checking documentation..."
check_file "README.test.md"
check_file "TEST_SUMMARY.md"
echo ""

echo "📦 Checking package.json scripts..."
if grep -q '"test":' package.json && \
   grep -q '"test:ci":' package.json && \
   grep -q '"test:e2e":' package.json && \
   grep -q '"test:coverage":' package.json && \
   grep -q '"test:a11y":' package.json; then
    echo -e "${GREEN}✓${NC} Test scripts configured"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Test scripts missing in package.json"
    ((FAILED++))
fi
echo ""

echo "📊 Checking dependencies..."
if grep -q '"@testing-library/react"' package.json && \
   grep -q '"@playwright/test"' package.json && \
   grep -q '"jest"' package.json && \
   grep -q '"msw"' package.json; then
    echo -e "${GREEN}✓${NC} Testing dependencies installed"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Some testing dependencies missing"
    ((FAILED++))
fi
echo ""

# Summary
echo "=========================================="
echo "Test Validation Summary"
echo "=========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All validations passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. npm install"
    echo "  2. npm run test:ci"
    echo "  3. npm run test:e2e"
    echo ""
    exit 0
else
    echo -e "${RED}❌ Some validations failed${NC}"
    echo "Please check the missing files or directories above."
    echo ""
    exit 1
fi
