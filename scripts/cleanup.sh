#!/usr/bin/env bash
set -e

ROOT_DIR="$(dirname "$(dirname "$0")")"
DATA_DIR="$ROOT_DIR/data"
LOGS_DIR="$ROOT_DIR/logs"

echo "Cleaning up data and logs..."

if [ -d "$DATA_DIR" ]; then
    rm -rf "$DATA_DIR"
    echo "Removed $DATA_DIR"
fi

if [ -d "$LOGS_DIR" ]; then
    rm -rf "$LOGS_DIR"
    echo "Removed $LOGS_DIR"
fi

# Also clean test project
# Clean test project build artifacts (optional)
# rm -f "$ROOT_DIR/test-project/go.sum"

echo "Done."
