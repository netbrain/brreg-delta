#!/usr/bin/env bash

# Local testing script that mimics the GitHub Actions workflow
# Runs with 1 batch instead of 256 for faster testing

set -e

echo "=== Brreg Delta Local Test ==="
echo ""

# Step 1: Download company list
echo "[1/4] Downloading company list..."
if [ ! -f companies.json.gz ]; then
  curl -L -o companies.json.gz https://data.brreg.no/enhetsregisteret/api/enheter/lastned
  echo "Downloaded: $(du -h companies.json.gz | cut -f1)"
else
  echo "Using existing companies.json.gz"
fi

# Step 2: Extract companies
echo ""
echo "[2/4] Extracting companies..."
./brreg-delta extract --list companies.json.gz --data data

# Step 3: Count extracted
echo ""
echo "[3/4] Counting extracted companies..."
ORG_COUNT=$(find data -type d -maxdepth 1 | tail -n +2 | wc -l)
echo "Total companies: $ORG_COUNT"
echo "Per batch (256 total): ~$((ORG_COUNT / 256))"

# Step 4: Run one batch
echo ""
echo "[4/4] Processing batch 0 (roles sync)..."
./brreg-delta -batch 0 -total 256 -workers 5 -data data

echo ""
echo "=== Test Complete ==="
echo "Check data/ directory for results"
