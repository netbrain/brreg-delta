#!/usr/bin/env bash
set -e

DATA_DIR="${1:-data}"

echo "Migrating from single-digit to triple-digit sharding..."
echo "Data directory: $DATA_DIR"

# Create temporary directory for new structure
TEMP_DIR="${DATA_DIR}.3digit-tmp"
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

# Copy .sync-state.json if it exists
if [ -f "$DATA_DIR/.sync-state.json" ]; then
  cp "$DATA_DIR/.sync-state.json" "$TEMP_DIR/"
fi

# Function to convert single-digit path to triple-digit
# Input: 1/2/3/4/5/6/7/8/9
# Output: 123/456/789
single_to_triple() {
  local orgnum="$1"
  # Remove slashes to get orgnum
  orgnum="${orgnum//\//}"

  # Split into triple-digit groups
  local result=""
  for ((i=0; i<${#orgnum}; i+=3)); do
    local chunk="${orgnum:$i:3}"
    if [ -n "$result" ]; then
      result="$result/$chunk"
    else
      result="$chunk"
    fi
  done
  echo "$result"
}

# Find all enhet.json files (these mark entity directories)
echo "Finding all entities..."
find "$DATA_DIR" -name "enhet.json" -type f | while read -r enhet_file; do
  # Get the directory containing enhet.json
  entity_dir=$(dirname "$enhet_file")

  # Extract orgnum from path (remove $DATA_DIR prefix)
  rel_path="${entity_dir#$DATA_DIR/}"

  # Skip if this is already in temp dir or has underenhet in path
  if [[ "$rel_path" == *"underenhet"* ]]; then
    continue
  fi

  # Convert path to orgnum (remove slashes)
  orgnum="${rel_path//\//}"

  # Skip if not exactly 9 or 8 digits
  if [[ ! "$orgnum" =~ ^[0-9]{8,9}$ ]]; then
    continue
  fi

  # Convert to triple-digit path
  new_path=$(single_to_triple "$orgnum")
  new_dir="$TEMP_DIR/$new_path"

  # Copy entity directory (enhet.json, roller.json)
  mkdir -p "$new_dir"
  if [ -f "$entity_dir/enhet.json" ]; then
    cp "$entity_dir/enhet.json" "$new_dir/"
  fi
  if [ -f "$entity_dir/roller.json" ]; then
    cp "$entity_dir/roller.json" "$new_dir/"
  fi

  # Handle underenheter if they exist
  if [ -d "$entity_dir/underenhet" ]; then
    mkdir -p "$new_dir/underenhet"

    # Copy each underenhet
    for underenhet_link in "$entity_dir/underenhet"/*; do
      if [ -L "$underenhet_link" ]; then
        # It's a symlink - we'll recreate it later
        underenhet_orgnum=$(basename "$underenhet_link")
        echo "$underenhet_orgnum" >> "$TEMP_DIR/.underenhet-links-$orgnum"
      elif [ -d "$underenhet_link" ]; then
        # It's a directory with underenhet data
        underenhet_orgnum=$(basename "$underenhet_link")

        # Copy to canonical location
        underenhet_new_path=$(single_to_triple "$underenhet_orgnum")
        underenhet_new_dir="$TEMP_DIR/$underenhet_new_path"
        mkdir -p "$underenhet_new_dir"

        if [ -f "$underenhet_link/underenhet.json" ]; then
          cp "$underenhet_link/underenhet.json" "$underenhet_new_dir/"
        fi
        if [ -f "$underenhet_link/roller.json" ]; then
          cp "$underenhet_link/roller.json" "$underenhet_new_dir/"
        fi

        # Mark that we need to create a symlink
        echo "$underenhet_orgnum" >> "$TEMP_DIR/.underenhet-links-$orgnum"
      fi
    done
  fi

  # Progress indicator
  if [ $((RANDOM % 1000)) -eq 0 ]; then
    echo "  Migrated entity $orgnum..."
  fi
done

echo "Creating symlinks..."
# Recreate symlinks
for link_file in "$TEMP_DIR"/.underenhet-links-*; do
  if [ ! -f "$link_file" ]; then
    continue
  fi

  parent_orgnum="${link_file##*-}"
  parent_path=$(single_to_triple "$parent_orgnum")

  while read -r underenhet_orgnum; do
    underenhet_path=$(single_to_triple "$underenhet_orgnum")

    # Calculate relative path
    parent_depth=$(echo "$parent_path" | tr -cd '/' | wc -c)
    up_levels=$((parent_depth + 1))  # +1 for underenhet dir

    link_dir="$TEMP_DIR/$parent_path/underenhet"
    mkdir -p "$link_dir"

    # Build relative path
    rel_path=""
    for ((i=0; i<up_levels; i++)); do
      rel_path="../$rel_path"
    done
    rel_path="${rel_path}$underenhet_path"

    ln -sf "$rel_path" "$link_dir/$underenhet_orgnum"
  done < "$link_file"

  rm "$link_file"
done

echo "Replacing old data with new structure..."
rm -rf "${DATA_DIR}.backup"
mv "$DATA_DIR" "${DATA_DIR}.backup"
mv "$TEMP_DIR" "$DATA_DIR"

echo ""
echo "Migration complete!"
echo "Old data backed up to: ${DATA_DIR}.backup"
echo "New triple-digit sharded data in: $DATA_DIR"
echo ""
echo "To verify and commit:"
echo "  git add $DATA_DIR"
echo "  git commit -m 'Migrate to triple-digit sharding'"
