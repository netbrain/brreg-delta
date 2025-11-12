# Brreg Delta Tracker
[![Sync Brreg Data](https://github.com/netbrain/brreg-delta/actions/workflows/sync.yml/badge.svg)](https://github.com/netbrain/brreg-delta/actions/workflows/sync.yml)

Historical tracking system for Norwegian company data from [Brønnøysundregistrene](https://data.brreg.no).

## Overview

Brreg.no only provides snapshot data. This project tracks changes over time by:
- Fetching enheter (companies) and underenheter (sub-units) from Brreg.no API
- Storing data as prettified JSON files in git
- Using rolling updates to track only what changed
- Running efficient incremental sync via GitHub Actions

### Available at  https://netbrain.github.io/brreg-delta/

## Architecture

### Data Collection
- **Source**: Brreg.no Enhetsregisteret API
- **Data**: Enheter + underenheter + roles for ~1.1M Norwegian entities
- **Storage**: Triple-digit sharding (e.g., `810/034/882/`)
  - `data/{xxx}/{yyy}/{zzz}/enhet.json` - Entity details
  - `data/{xxx}/{yyy}/{zzz}/roller.json` - Roles and representatives
  - `data/{xxx}/{yyy}/{zzz}/underenhet.json` - Sub-unit details (if underenhet)
  - Symlinks in parent's `underenhet/` directory for navigation
- **Change Detection**: Git-native (no manual hashing)
- **State Management**: `.sync-state.json` tracks last oppdateringsid for both enheter and underenheter

### Sync Strategy

**Initial Sync** (first run):
- Downloads `/api/enheter/lastned` (gzipped JSON)
- Downloads `/api/underenheter/lastned` (gzipped JSON)
- Downloads `/api/roller/totalbestand` (gzipped JSON)
- Extracts all to individual files with triple-digit sharding
- Creates symlinks for underenheter navigation
- Fetches latest oppdateringsid from API and saves to state

**Incremental Sync** (daily):
- Fetches `/api/oppdateringer/enheter?oppdateringsid={last}&size=1000`
- Fetches `/api/oppdateringer/underenheter?oppdateringsid={last}&size=1000`
- For each changed entity, fetches current data + roles
- Updates state file with new oppdateringsid values

### GitHub Actions Workflow
- **Single Job**: No matrix complexity
- **Auto-detection**: Initial vs incremental based on state file
- **Smart Downloads**: Only downloads bulk files on first run
- **Sparse Checkout**: Fetches only .sync-state.json from data branch
- **Schedule**: Daily at 5:30 AM UTC
- **Data Branch**: Commits to separate `data` branch for clean history

## Usage

### Run Sync

```bash
# Build
go build

# Run sync (auto-detects initial vs incremental)
./brreg-delta -data data
```

**First run**: Downloads bulk files and extracts all entities
**Subsequent runs**: Fetches only updates since last sync

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-data` | data | Data directory |

### GitHub Actions

The workflow runs automatically on schedule, or can be triggered manually:

```bash
# Manual trigger
gh workflow run sync.yml
```

**First run**: Downloads bulk files and performs initial sync
**Subsequent runs**: Incremental updates only

## Data Structure

Using triple-digit sharding for optimal Git performance:

```
data/
├── .sync-state.json                     # Sync state tracking
├── 810/                                 # First 3 digits
│   └── 034/                             # Next 3 digits
│       └── 882/                         # Last 3 digits (orgnum: 810034882)
│           ├── enhet.json               # Entity details
│           ├── roller.json              # Roles and representatives
│           └── underenhet/              # Sub-units directory
│               └── 914930553 -> ../../../../914/930/553  # Symlink to underenhet
├── 914/
│   └── 930/
│       └── 553/                         # Underenhet (orgnum: 914930553)
│           ├── underenhet.json          # Sub-unit details
│           └── roller.json              # Sub-unit roles
└── README.md                            # Data branch documentation
```

**Sharding Benefits**:
- Optimal Git tree performance (hundreds of entries per directory, not thousands)
- Fast filesystem operations
- Efficient git operations with smaller tree objects

## Performance

**Initial Sync** (first run):
- Downloads 3 bulk files (enheter, underenheter, roller)
- Extracts all entities to individual files with triple-digit sharding
- Fetches latest oppdateringsid values

**Incremental Sync** (subsequent runs):
- Fetches only changed entities since last sync
- Updates only modified files
- Storage growth: Minimal (only changed files)

## API Endpoints Used

**Initial Sync:**
- `GET /api/enheter/lastned` - Bulk enheter download
- `GET /api/underenheter/lastned` - Bulk underenheter download
- `GET /api/roller/totalbestand` - Bulk roller download
- `GET /api/oppdateringer/enheter?oppdateringsid={high}&size=1000` - Find latest oppdateringsid
- `GET /api/oppdateringer/underenheter?oppdateringsid={high}&size=1000` - Find latest oppdateringsid

**Incremental Sync:**
- `GET /api/oppdateringer/enheter?oppdateringsid={id}&size=1000` - Enheter updates (paginated)
- `GET /api/oppdateringer/underenheter?oppdateringsid={id}&size=1000` - Underenheter updates (paginated)
- `GET /api/enheter/{orgnum}` - Single enhet details
- `GET /api/underenheter/{orgnum}` - Single underenhet details
- `GET /api/enheter/{orgnum}/roller` - Enhet roles
- `GET /api/underenheter/{orgnum}/roller` - Underenhet roles

## Requirements

- Go 1.23+
- GitHub repository with Actions enabled
- Storage space for entity data

## Configuration

Edit `.github/workflows/sync.yml` to adjust:
- Schedule: Change `cron` expression (default: daily at 5:30 AM UTC)
- Data directory: Modify `-data` flag (default: `data-worktree/data`)
- Timeout: Adjust job timeout (default: 30 minutes)

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
