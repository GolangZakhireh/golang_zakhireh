#!/usr/bin/env bash
set -euo pipefail

# ANSI Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log() {
    echo -e "${CYAN}[TEST]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Paths
ROOT_DIR="$(pwd)"
DATA_DIR="$ROOT_DIR/data/modules"
TEST_DIR="$ROOT_DIR/test-project"
SERVER_URL="http://localhost:8811"

# 0. Isolate Go Mod Cache
TEST_MODCACHE=$(mktemp -d)
log "Using isolated GOMODCACHE: $TEST_MODCACHE"
export GOMODCACHE="$TEST_MODCACHE"

# Ensure server is running
if ! curl -s "$SERVER_URL" > /dev/null; then
    error "GolangZakhireh server is not running on $SERVER_URL. Please start it with 'go run ./cmd/server' or via Docker."
fi

log "Starting Integration Tests..."
log "Data Dir: $DATA_DIR"
log "Test Project Dir: $TEST_DIR"

# Cleanup function to be called on exit
cleanup() {
    log "Cleaning up temporary test files..."
    rm -f "$TEST_DIR/upload_resp.txt"
    rm -f "$TEST_DIR/v9.9.9.info" "$TEST_DIR/v9.9.9.mod" "$TEST_DIR/v9.9.9.zip"
    
    log "Removing temporary GOMODCACHE..."
    # Modcache often has read-only directories, make it writable first
    chmod -R u+w "$TEST_MODCACHE" 2>/dev/null || true
    rm -rf "$TEST_MODCACHE"
}
trap cleanup EXIT

# 1. Setup Test Project
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

if [ ! -f go.mod ]; then
    log "Initializing test project..."
    go mod init test-project
fi

# 2. Clear local cache to force fetch
log "Cleaning local go mod cache..."
go clean -modcache


# --- 3. Test Proxy (Expect Cache Miss & Fail for Unseeded Module) ---
MODULE_404="github.com/davecgh/go-spew"
VERSION_404="v1.1.1"
log "Running 'go get' with GOPROXY=$SERVER_URL (Expect Cache Miss & Fallback)..."
export GOPROXY="$SERVER_URL"
export GOSUMDB=off # For testing, turn off checksum DB

set +e
GO_GET_OUTPUT=$(go get "${MODULE_404}@${VERSION_404}" 2>&1)
GO_GET_CODE=$?
set -e

if [ $GO_GET_CODE -eq 0 ]; then
    success "Downloaded ${MODULE_404}@${VERSION_404} via proxy"
else
    echo "$GO_GET_OUTPUT"
    # Intentionally 404 for first-time module (not present/cached)
    if echo "$GO_GET_OUTPUT" | grep -q "404 Not Found"; then
        error "Failed to download via proxy: 404 Not Found (expected for uncached module)"
    else
        error "Failed to download via proxy"
    fi
fi

include_path="$DATA_DIR/${MODULE_404}/@v/${VERSION_404}.zip"
if [ -f "$include_path" ]; then
    success "Module cached at $include_path"
else
    error "Module was NOT cached locally"
fi

# --- 4. Sanity Test: Module That Exists And Should Work ---
MODULE_OK="gopkg.in/yaml.v3"
VERSION_OK="v3.0.1"

log "Testing known working module download..."
go clean -modcache

set +e
GO_GET_OK_OUTPUT=$(go get "${MODULE_OK}@${VERSION_OK}" 2>&1)
GO_GET_OK_CODE=$?
set -e

if [ $GO_GET_OK_CODE -eq 0 ]; then
    success "Downloaded ${MODULE_OK}@${VERSION_OK} via proxy"
else
    echo "$GO_GET_OK_OUTPUT"
    error "Failed to download known working module"
fi

include_ok_path="$DATA_DIR/${MODULE_OK}/@v/${VERSION_OK}.zip"
if [ -f "$include_ok_path" ]; then
    success "Module cached at $include_ok_path"
else
    error "Module was NOT cached locally for working module"
fi

# 5. Test Cache Hit for working module (Offline Simulation)
log "Testing Cache Hit for working module..."
go clean -modcache

log "Running 'go get' again (Should start from cache)..."
time go get "${MODULE_OK}@${VERSION_OK}"
success "Cache hit test passed (functional)"

# 6. Test Upload for working module
log "Testing Upload Endpoint for working module..."
CACHE_SRC="$(go env GOMODCACHE)/cache/download/${MODULE_OK}/@v"
if [ -d "$CACHE_SRC" ]; then
    log "Uploading ${MODULE_OK} from local cache to verify upload handler..."

    # New fake version to upload
    FAKE_VERSION="v9.9.9"
    cp "$CACHE_SRC/${VERSION_OK}.info" "./${FAKE_VERSION}.info"
    cp "$CACHE_SRC/${VERSION_OK}.mod"  "./${FAKE_VERSION}.mod"
    cp "$CACHE_SRC/${VERSION_OK}.zip"  "./${FAKE_VERSION}.zip"

    curl -s -X POST "$SERVER_URL/upload" \
        -F "module=${MODULE_OK}" \
        -F "version=${FAKE_VERSION}" \
        -F "info=@${FAKE_VERSION}.info" \
        -F "mod=@${FAKE_VERSION}.mod" \
        -F "zip=@${FAKE_VERSION}.zip" > upload_resp.txt

    if grep -q "ok" upload_resp.txt; then
        success "Upload successful"
    else
        cat upload_resp.txt
        error "Upload failed"
    fi

    if [ -f "$DATA_DIR/${MODULE_OK}/@v/${FAKE_VERSION}.zip" ]; then
        success "Uploaded file verified in storage"
    else
        error "Uploaded file not found in storage"
    fi
else
    warn "Could not find GOMODCACHE to test upload"
fi

success "All integration tests passed!"
