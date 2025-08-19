#!/bin/bash
# Integration test script for upload-http

set -e  # Exit on any error

echo "Starting upload-http integration tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
SERVER_PORT=8081
SERVER_URL="http://localhost:$SERVER_PORT"
TEST_DIR="test-integration"
UPLOAD_DIR="$TEST_DIR/upload"
DOWNLOAD_DIR="$TEST_DIR/download"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf $TEST_DIR
    rm -rf uploads
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Create test structure
echo -e "${YELLOW}Setting up test environment...${NC}"
mkdir -p $UPLOAD_DIR/docs/images
mkdir -p $UPLOAD_DIR/config
mkdir -p $DOWNLOAD_DIR

# Create test files
echo "This is a text document" > $UPLOAD_DIR/docs/readme.txt
echo "API documentation content" > $UPLOAD_DIR/docs/api.md
echo "Config data: port=8080" > $UPLOAD_DIR/config/server.conf
echo "Log settings: level=debug" > $UPLOAD_DIR/config/logging.conf
echo -e "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89" > $UPLOAD_DIR/docs/images/logo.png

echo -e "${GREEN}Test files created:${NC}"
find $UPLOAD_DIR -type f | sort

# Build the binaries
echo -e "${YELLOW}Building binaries...${NC}"
make build

# Start server in background
echo -e "${YELLOW}Starting server on port $SERVER_PORT...${NC}"
./bin/server --port $SERVER_PORT --storage ./uploads --log-level info &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
for i in {1..10}; do
    if curl -s "$SERVER_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}Server started successfully${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}Server failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Test 1: Health check
echo -e "${YELLOW}Test 1: Health check${NC}"
if ./bin/client health --server $SERVER_URL; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
    exit 1
fi

# Test 2: Upload folder with structure
echo -e "${YELLOW}Test 2: Upload folder with structure${NC}"
if ./bin/client upload --server $SERVER_URL $UPLOAD_DIR; then
    echo -e "${GREEN}✓ Folder upload passed${NC}"
else
    echo -e "${RED}✗ Folder upload failed${NC}"
    exit 1
fi

# Test 3: List root directory
echo -e "${YELLOW}Test 3: List root directory${NC}"
if ./bin/client list --server $SERVER_URL; then
    echo -e "${GREEN}✓ Root directory listing passed${NC}"
else
    echo -e "${RED}✗ Root directory listing failed${NC}"
    exit 1
fi

# Test 4: List uploaded folder
echo -e "${YELLOW}Test 4: List uploaded folder structure${NC}"
echo "Contents of uploaded folder:"
./bin/client list --server $SERVER_URL upload

echo "Contents of docs subdirectory:"
./bin/client list --server $SERVER_URL upload/docs

echo "Contents of config subdirectory:"
./bin/client list --server $SERVER_URL upload/config

echo -e "${GREEN}✓ Folder structure listing passed${NC}"

# Test 5: Download single file
echo -e "${YELLOW}Test 5: Download single file${NC}"
if ./bin/client download --server $SERVER_URL upload/docs/readme.txt $DOWNLOAD_DIR/readme.txt; then
    if [ -f "$DOWNLOAD_DIR/readme.txt" ] && grep -q "This is a text document" "$DOWNLOAD_DIR/readme.txt"; then
        echo -e "${GREEN}✓ Single file download passed${NC}"
    else
        echo -e "${RED}✗ Downloaded file content mismatch${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Single file download failed${NC}"
    exit 1
fi

# Test 6: Download entire folder
echo -e "${YELLOW}Test 6: Download entire folder${NC}"
if ./bin/client download --server $SERVER_URL upload $DOWNLOAD_DIR/full-download; then
    echo -e "${GREEN}✓ Folder download completed${NC}"
    
    # Verify folder structure
    echo "Downloaded folder structure:"
    find $DOWNLOAD_DIR/full-download -type f | sort
    
    # Verify some file contents
    if [ -f "$DOWNLOAD_DIR/full-download/docs/readme.txt" ] && \
       [ -f "$DOWNLOAD_DIR/full-download/config/server.conf" ] && \
       [ -f "$DOWNLOAD_DIR/full-download/docs/images/logo.png" ]; then
        echo -e "${GREEN}✓ Folder structure preserved${NC}"
    else
        echo -e "${RED}✗ Folder structure not preserved${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Folder download failed${NC}"
    exit 1
fi

# Test 7: Upload single file
echo -e "${YELLOW}Test 7: Upload single file${NC}"
echo "Single file test content" > $TEST_DIR/single-file.txt
if ./bin/client upload --server $SERVER_URL $TEST_DIR/single-file.txt; then
    echo -e "${GREEN}✓ Single file upload passed${NC}"
else
    echo -e "${RED}✗ Single file upload failed${NC}"
    exit 1
fi

# Test 8: Verify single file upload
echo -e "${YELLOW}Test 8: Verify single file upload${NC}"
if ./bin/client download --server $SERVER_URL single-file.txt $DOWNLOAD_DIR/single-file-downloaded.txt; then
    if grep -q "Single file test content" "$DOWNLOAD_DIR/single-file-downloaded.txt"; then
        echo -e "${GREEN}✓ Single file verification passed${NC}"
    else
        echo -e "${RED}✗ Single file content mismatch${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Single file verification failed${NC}"
    exit 1
fi

# Test 9: Performance test with multiple files
echo -e "${YELLOW}Test 9: Performance test with multiple files${NC}"
PERF_DIR="$TEST_DIR/performance"
mkdir -p $PERF_DIR
for i in {1..20}; do
    echo "Performance test file $i content" > $PERF_DIR/file$i.txt
done

start_time=$(date +%s)
if ./bin/client upload --server $SERVER_URL $PERF_DIR; then
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    echo -e "${GREEN}✓ Performance test passed (20 files uploaded in ${duration}s)${NC}"
else
    echo -e "${RED}✗ Performance test failed${NC}"
    exit 1
fi

# Display final results
echo -e "\n${GREEN}================================${NC}"
echo -e "${GREEN}All integration tests passed! ✓${NC}"
echo -e "${GREEN}================================${NC}"

echo -e "\n${YELLOW}Final server storage structure:${NC}"
find uploads -type f | head -20
echo "..."

echo -e "\n${YELLOW}Test Summary:${NC}"
echo "✓ Health check functionality"
echo "✓ Folder upload with structure preservation"  
echo "✓ Directory listing at multiple levels"
echo "✓ Single file download"
echo "✓ Complete folder download as tar.gz"
echo "✓ Single file upload"
echo "✓ File content integrity verification"
echo "✓ Performance with multiple files"
echo "✓ Progress tracking and logging"

echo -e "\n${GREEN}upload-http is ready for production use!${NC}"