# Brreg Delta Tracker

Historical tracking system for Norwegian company data from [Brønnøysundregistrene](https://data.brreg.no).

## Overview

Brreg.no only provides snapshot data. This project tracks changes over time by:
- Fetching company data and roles from Brreg.no API
- Storing data as JSON files in git
- Using hash-based change detection to track only what changed
- Running distributed scraping via GitHub Actions (256 parallel jobs)

## Architecture

### Data Collection
- **Source**: Brreg.no Enhetsregisteret API
- **Data**: Company details + roles/representatives for ~1.1M Norwegian companies
- **Storage**: `data/{orgnum}/company.json` and `data/{orgnum}/roles.json`
- **Change Detection**: SHA256 hash comparison - only saves when data changes

### GitHub Actions Workflow
- **Preprocessing**: Downloads & extracts all companies to individual files
- **No Artifacts**: Company data stored in git (avoids 256 artifact downloads)
- **Matrix Strategy**: 256 parallel jobs
- **Distribution**: Each job processes ~4,470 companies (1.1M / 256)
- **Workers**: 1 worker per job (256 total concurrent requests)
- **Execution Time**: ~75 minutes total
- **Schedule**: Daily at 2 AM UTC (configurable)

### Workflow Steps
1. **Preprocessing job**:
   - Downloads company list from Brreg.no (~200MB gzipped)
   - Extracts each company to `data/{orgnum}/company.json`
   - Counts extracted companies
   - Commits company data to git
2. **256 matrix jobs** run in parallel:
   - Each pulls latest company data from git
   - Processes batch of companies (fetches roles only)
   - Commits role updates to individual branches (`sync-batch-{N}`)
3. **Merge job**:
   - Combines all 256 branches
   - Creates single PR with all changes
   - Auto-squash and merge

## Usage

### Local Testing

Test the workflow locally before pushing to GitHub:

```bash
# Run full test (downloads, extracts, processes batch 0)
./test-local.sh
```

This mimics the GitHub Actions workflow but processes only 1 batch with 5 workers for testing.

### Run Manually

```bash
# Build
go build

# Initial setup: Download and extract companies
curl -L -o companies.json.gz https://data.brreg.no/enhetsregisteret/api/enheter/lastned
./brreg-delta extract --list companies.json.gz --data data

# Count extracted companies
./brreg-delta count --data data

# Or count with shell (faster)
find data -type d -maxdepth 1 | tail -n +2 | wc -l

# Process batch (fetch roles for companies)
./brreg-delta -batch 0 -total 256 -workers 1 -data data

# Or use the test script
./test-local.sh
```

### Commands

**extract** - Extract companies from bulk download:
```bash
./brreg-delta extract --list companies.json.gz --data data
```

**count** - Count extracted companies:
```bash
./brreg-delta count --data data
```

**Main** - Fetch roles for batch:
```bash
./brreg-delta -batch 0 -total 256 -workers 1 -data data
```

### Command Line Options

| Command | Flag | Default | Description |
|---------|------|---------|-------------|
| extract | `--list` | (required) | Path to company list file (gzipped or plain JSON) |
| extract | `--data` | data | Output directory |
| count | `--data` | data | Data directory to count |
| main | `-batch` | 0 | Batch number (0-255) |
| main | `-total` | 256 | Total number of batches |
| main | `-workers` | 1 | Parallel workers per batch (increase if API allows) |
| main | `-data` | data | Data directory |

### GitHub Actions

The workflow runs automatically on schedule, or can be triggered manually:

```bash
# Manual trigger
gh workflow run sync.yml
```

**First run**: Preprocessing extracts all companies to git
**Subsequent runs**: Only fetches roles for existing companies

## Data Structure

```
data/
├── 123456789/
│   ├── company.json    # Company details
│   └── roles.json      # Roles and representatives
├── 987654321/
│   ├── company.json
│   └── roles.json
...
```

## Performance

**Preprocessing (one-time extraction)**:
- Download: ~30 seconds (~200MB gzipped)
- Extract: ~1 minute (1.1M companies to individual files)
- Data size: ~6-8 GB (estimated)

**Regular Runs (roles sync)**:
- API calls: ~1.1M (one per company for roles endpoint)
- With 256 jobs × 1 worker: ~75 minutes (~256 req/sec)
- Conservative rate (can increase workers if API allows)
- Only roles change detection
- Minimal storage growth

## API Endpoints Used

- `GET /api/enheter/lastned` - Bulk company list download (preprocessing)
- `GET /api/enheter/{orgnum}/roller` - Company roles (daily sync)

## Requirements

- Go 1.23+
- GitHub repository with Actions enabled
- Storage space for company data

## Configuration

Edit `.github/workflows/sync.yml` to adjust:
- Schedule: Change `cron` expression
- Workers: Modify `-workers` flag
- Batches: Adjust matrix size (current: 256)

## Development

```bash
# Run tests
go test ./...

# Format code
go fmt ./...

# Build
go build
```

## License

MIT
