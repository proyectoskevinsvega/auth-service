#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Base URL
BASE_URL="${BASE_URL:-http://localhost:8080}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}🚀 k6 Load Testing Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Target: ${BASE_URL}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}❌ k6 is not installed!${NC}"
    echo -e "${YELLOW}Please install k6 first:${NC}"
    echo -e "  - Windows (Chocolatey): choco install k6"
    echo -e "  - Windows (Scoop): scoop install k6"
    echo -e "  - macOS: brew install k6"
    echo -e "  - Or download from: https://github.com/grafana/k6/releases"
    exit 1
fi

echo -e "${GREEN}✓ k6 is installed${NC}"
echo ""

# Check if service is running
echo -e "${YELLOW}Checking if service is running...${NC}"
if ! curl -s "${BASE_URL}/health" > /dev/null; then
    echo -e "${RED}❌ Service is not responding at ${BASE_URL}${NC}"
    echo -e "${YELLOW}Please start the service first:${NC}"
    echo -e "  make run"
    exit 1
fi

echo -e "${GREEN}✓ Service is running${NC}"
echo ""

# Create results directory
mkdir -p results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    local result_file="results/${test_name}_${TIMESTAMP}.json"

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running: ${test_name}${NC}"
    echo -e "${BLUE}========================================${NC}"

    k6 run \
        -e BASE_URL="${BASE_URL}" \
        --out json="${result_file}" \
        "${test_file}"

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✓ ${test_name} completed successfully${NC}"
        echo -e "${GREEN}  Results saved to: ${result_file}${NC}"
    else
        echo -e "${RED}✗ ${test_name} failed${NC}"
    fi

    echo ""
    sleep 5  # Cool down between tests

    return $exit_code
}

# Run all tests
echo -e "${YELLOW}Starting test suite...${NC}"
echo ""

FAILED_TESTS=0

# Test 1: Validate Token (most important - cache performance)
run_test "validate-token" "validate-token.js"
if [ $? -ne 0 ]; then ((FAILED_TESTS++)); fi

# Test 2: Login
run_test "login" "login.js"
if [ $? -ne 0 ]; then ((FAILED_TESTS++)); fi

# Test 3: Register
run_test "register" "register.js"
if [ $? -ne 0 ]; then ((FAILED_TESTS++)); fi

# Test 4: Mixed Load (realistic scenario)
run_test "mixed-load" "mixed-load.js"
if [ $? -ne 0 ]; then ((FAILED_TESTS++)); fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}📊 TEST SUITE COMPLETE${NC}"
echo -e "${BLUE}========================================${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "Results saved in: ./results/"
    exit 0
else
    echo -e "${RED}✗ ${FAILED_TESTS} test(s) failed${NC}"
    echo -e "Check the results in: ./results/"
    exit 1
fi
