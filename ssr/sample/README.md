# Sample Data for Testing gen-history

This directory contains representative sample data with git history for testing the gen-history tool during development.

## Structure

```
sample/
├── data/
│   ├── 810/034/882/enhet.json      # EVRY ASA (parent entity)
│   └── 923/456/789/underenhet.json # Oslo Consulting (sub-entity)
├── output/                          # Generated HTML files
├── generate-history.sh              # Script to regenerate git history
└── README.md
```

## Sample Entities

### 810034882 - EVRY ASA
A fictional main entity (enhet) with 5 commits spanning 2020-2023:
- Employee count changes (3850 → 4100 → 4234)
- MVA registration added
- Annual reports submitted (2021, 2022)
- Various registry statuses added

### 923456789 - Oslo Consulting Avdeling
A fictional sub-entity (underenhet) with 2 commits:
- Initial creation (2021)
- Employee growth (32 → 45)

## Regenerating Sample Data

If you need to reset or regenerate the git history:

```bash
./generate-history.sh
```

This will:
1. Initialize a fresh git repository
2. Create 7 commits with realistic timestamps and changes
3. Prepare the data for testing

## Testing gen-history

From this directory, run:

```bash
# Single entity
nix develop ../.. --command bash -c \
  "../gen-history/gen-history generate --orgnum 810034882 --output ./output --template ../entity-template"

# All entities
nix develop ../.. --command bash -c \
  "../gen-history/gen-history generate --all --output ./output --template ../entity-template --workers 2"

# Generate search index
nix develop ../.. --command bash -c \
  "../gen-history/gen-history index --output ./output/search-index.json"
```

## Viewing Results

Open the generated HTML files in a browser:

```bash
firefox output/810/034/882.html
firefox output/923/456/789.html
```

## What the Sample Tests

- Git history parsing with multiple commits
- Field-level diffing for various data types
- Timeline generation with Norwegian labels
- Hugo template rendering
- Sharded directory output structure
- Both enhet and underenhet types
- Nested object changes (organisasjonsform, addresses, etc.)
- New field additions over time
- Employee count updates
- Registry status changes
