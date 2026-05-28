#!/usr/bin/env bash

CAPABILITIES_PATH="./build/capabilities.json"

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 {check|write}"
  exit 1
fi

MODE="$1"

tmpfile=$(mktemp)

if ! ./regal capabilities > "$tmpfile"; then
  echo "Error: failed to generate capabilities JSON" >&2
  rm "$tmpfile"
  exit 1
fi

if [[ "$MODE" == "check" ]]; then
  if ! cmp -s "$tmpfile" "$CAPABILITIES_PATH"; then
    echo "build/capabilities.json is out of date. Please run '$0 write' to update it."
    rm "$tmpfile"
    exit 1
  else
    echo "build/capabilities.json is up to date."
    rm "$tmpfile"
  fi
elif [[ "$MODE" == "write" ]]; then
  mv "$tmpfile" "$CAPABILITIES_PATH"
  echo "build/capabilities.json has been updated."
else
  echo "Unknown mode: $MODE"
  echo "Usage: $0 {check|write}"
  rm "$tmpfile"
  exit 1
fi
