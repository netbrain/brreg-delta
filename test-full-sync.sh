#!/usr/bin/env bash
set -e

echo "=== Brreg Delta - Full Sync Test ==="
echo ""
echo "WARNING: This will:"
echo "  - Download ~530MB (enheter + underenheter + roller bulk files)"
echo "  - Create ~6-8GB of data in data-test/"
echo "  - Take ~5 minutes"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

# Cleanup previous test
echo "Cleaning up previous test data..."
rm -rf data-test
mkdir -p data-test

# Build
echo "Building..."
go build -o brreg-delta-test

# Download bulk files (mimics GitHub Actions)
echo ""
echo "Downloading bulk files..."
echo "This may take a few minutes (~530MB total)..."

if [ ! -f enheter.json.gz ]; then
    echo "Downloading enheter..."
    curl -L -o enheter.json.gz https://data.brreg.no/enhetsregisteret/api/enheter/lastned
    echo "Downloaded: $(du -h enheter.json.gz | cut -f1)"
else
    echo "Using existing enheter.json.gz ($(du -h enheter.json.gz | cut -f1))"
fi

if [ ! -f underenheter.json.gz ]; then
    echo "Downloading underenheter..."
    curl -L -o underenheter.json.gz https://data.brreg.no/enhetsregisteret/api/underenheter/lastned
    echo "Downloaded: $(du -h underenheter.json.gz | cut -f1)"
else
    echo "Using existing underenheter.json.gz ($(du -h underenheter.json.gz | cut -f1))"
fi

if [ ! -f roller.json.gz ]; then
    echo "Downloading roller..."
    curl -L -o roller.json.gz https://data.brreg.no/enhetsregisteret/api/roller/totalbestand
    echo "Downloaded: $(du -h roller.json.gz | cut -f1)"
else
    echo "Using existing roller.json.gz ($(du -h roller.json.gz | cut -f1))"
fi

echo ""
echo "Running full initial sync..."
echo "This will extract all entities to data-test/"
echo ""

# Run sync (will detect no state file and do initial sync)
./brreg-delta-test -data data-test

echo ""
echo "=== Test Complete ==="
echo ""
echo "Results:"
echo "  - Data directory: data-test/"
echo "  - Size: $(du -sh data-test 2>/dev/null | cut -f1)"
echo "  - State file: data-test/.sync-state.json"
echo ""

# Show summary
if [ -f data-test/.sync-state.json ]; then
    echo "Final state:"
    cat data-test/.sync-state.json | jq '.' 2>/dev/null || cat data-test/.sync-state.json
fi

echo ""
echo "Check extracted data:"
echo "  - Count entities: find data-test -type d -maxdepth 1 | tail -n +2 | wc -l"
echo "  - View example: cat data-test/*/enhet.json | head -20"
echo ""
echo "To clean up: rm -rf data-test brreg-delta-test enheter.json.gz underenheter.json.gz roller.json.gz"
