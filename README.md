# Brreg Data Branch

This branch contains historical data for ~2M Norwegian entities (companies and sub-units) from Brønnøysundregistrene.

## Structure

```
data/
├── .sync-state.json                     # Sync state tracking
├── {orgnum}/                            # Enhet (parent company) - single digit sharding
│   ├── enhet.json                       # Entity details
│   ├── roller.json                      # Roles and representatives
│   └── underenhet/
│       └── {underenhet_orgnum}/
│           ├── underenhet.json          # Sub-unit details
│           └── roller.json              # Sub-unit roles
└── {underenhet_orgnum} -> {parent}/underenhet/{underenhet_orgnum}  # Symlinks for direct lookup
```

## Single-Digit Sharding

Organization numbers are sharded into single-digit directories (0-9) for optimal Git performance:
- `123456789` → `1/2/3/4/5/6/7/8/9/enhet.json`

## Updates

Data is automatically synced daily at 5:30 AM UTC via GitHub Actions.

## Size

~2M entities (companies + sub-units), approximately 6-8 GB of data.
