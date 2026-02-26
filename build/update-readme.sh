#!/usr/bin/env bash

README_SECTIONS_DIR="./docs/readme-sections"
README_PATH="./README.md"
BADGES_PATH="$README_SECTIONS_DIR/badges.md"
MANIFEST="$README_SECTIONS_DIR/github-manifest"

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 {check|write}"
  exit 1
fi

MODE="$1"

# Function to update badges with current OPA version
update_badges() {
  # Extract OPA version using go command and jq
  OPA_VERSION=$(go list -m -json github.com/open-policy-agent/opa | jq -r '.Version')

  if [[ -z "$OPA_VERSION" ]]; then
    echo "Error: Could not find OPA version using go list command" >&2
    exit 1
  fi

  # Extract base version (remove pre-release suffix for badge text)
  BASE_VERSION=$(echo "$OPA_VERSION" | sed 's/-.*$//')

  # Create a temporary file for badges
  badges_tmpfile=$(mktemp)

  # Read the badges file and update the OPA badge line
  while IFS= read -r line; do
    if [[ $line =~ ^\!\[OPA\ v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
      echo "![OPA $BASE_VERSION](https://www.openpolicyagent.org/badge/$BASE_VERSION)" >> "$badges_tmpfile"
    else
      echo "$line" >> "$badges_tmpfile"
    fi
  done < "$BADGES_PATH"

  # Check if badges need updating
  if ! cmp -s "$badges_tmpfile" "$BADGES_PATH"; then
    if [[ "$MODE" == "write" ]]; then
      mv "$badges_tmpfile" "$BADGES_PATH"
      echo "Updated badges.md with OPA version $OPA_VERSION."
    else
      rm "$badges_tmpfile"
      echo "README.md is out of date (badges need OPA version update to $OPA_VERSION). Please run '$0 write' to update it."
      exit 1
    fi
  else
    rm "$badges_tmpfile"
  fi
}

# Update badges first
update_badges

# Create a temporary file to hold the new content
tmpfile=$(mktemp)

# Build new content into tmpfile
while IFS= read -r file; do
  section_path="$README_SECTIONS_DIR/$file"

  if [[ -f "$section_path" ]]; then
    cat "$section_path" >> "$tmpfile"
    echo -e "\n" >> "$tmpfile"
  else
    echo "Section file not found: $section_path" >&2
    exit 1
  fi
done < "$MANIFEST"

if [[ "$MODE" == "check" ]]; then
  if ! cmp -s "$tmpfile" "$README_PATH"; then
    echo "README.md is out of date. Please run '$0 write' to update it."
    rm "$tmpfile"
    exit 1
  else
    echo "README.md is up to date."
  fi
elif [[ "$MODE" == "write" ]]; then
  mv "$tmpfile" "$README_PATH"
  echo "README.md has been updated."
else
  echo "Unknown mode: $MODE"
  echo "Usage: $0 {check|write}"
  rm "$tmpfile"
  exit 1
fi
