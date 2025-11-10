# Brreg Data Branch

This branch contains data for ~2M Norwegian entities (companies and sub-units) from Brønnøysundregistrene.

## Structure

```
data/
├── .sync-state.json                     # Sync state tracking
├── {enhet_orgnum}/                      # 3-digit sharding for enheter
│   ├── enhet.json                       # Entity details
│   ├── roller.json                      # Roles and representatives
│   └── underenhet/
│       └── {underenhet_orgnum} -> ../../../../{underenhet_orgnum}/  # Symlink to underenhet
└── {underenhet_orgnum}/                 # Top-level underenhet directory (3-digit sharding)
    └── underenhet.json                  # Sub-unit details (no roller.json)
```

## 3-Digit Sharding

Organization numbers are sharded into 3-digit directories for optimal Git performance:
- `810359862` → `810/359/862/enhet.json`
- Underenheter have their own top-level paths with symlinks from parent enhet directories for easy navigation

## Data Files

- **enhet.json**: Entity details including name, address, business codes, employee count, and registration dates
- **roller.json**: Roles and representatives (board members, CEO, auditors, etc.) - only exists for enheter
- **underenhet.json**: Sub-unit details with similar structure to enhet.json
- **.sync-state.json**: Sync state tracking with last update IDs and timestamps
