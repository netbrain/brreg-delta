#!/usr/bin/env bash
set -e

echo "=== Brreg Delta - Local Test ==="
echo ""

# Cleanup previous test
echo "Cleaning up previous test data..."
rm -rf data-test
mkdir -p data-test

# Create minimal state file to trigger incremental sync
# This avoids downloading 200MB+ bulk files
echo "Creating test state file (triggers incremental sync)..."
cat > data-test/.sync-state.json <<EOF
{
  "lastEnheterOppdateringsid": 13970000,
  "lastUnderenheterOppdateringsid": 0,
  "lastRollerOppdateringsid": 0,
  "syncStartedAt": "2025-01-01T00:00:00Z",
  "lastSuccessfulSync": "2025-01-01T00:00:00Z"
}
EOF

echo "State file created - will fetch updates since oppdateringsid 13970000"
echo ""

# Build
echo "Building..."
go build -o brreg-delta-test

echo ""
echo "Running incremental sync (fetches recent updates only)..."
echo "This tests the full incremental update flow against live API"
echo ""

# Run sync
./brreg-delta-test -data data-test

echo ""
echo "=== Test Complete ==="
echo ""
echo "Results:"
echo "  - State file: data-test/.sync-state.json"
echo "  - Updated entities in: data-test/"
echo ""
echo "Check the data-test/ directory to see fetched updates"
echo ""

# Show summary
if [ -f data-test/.sync-state.json ]; then
    echo "Final state:"
    cat data-test/.sync-state.json | jq '.' 2>/dev/null || cat data-test/.sync-state.json
fi

echo ""
echo "To clean up: rm -rf data-test brreg-delta-test"
