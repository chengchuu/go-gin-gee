#!/bin/bash
set -e
echo "Cleaning Assets ..."

DIRS=( "./data" "./log" )

for dir in "${DIRS[@]}"; do
  # safety guard: skip empty or root-like paths
  if [ -z "$dir" ] || [ "$dir" = "/" ]; then
    echo "Refusing to remove unsafe path: '$dir'" >&2
    continue
  fi

  if [ -d "$dir" ]; then
    echo "Removing directory: $dir"
    rm -rf -- "$dir"
  else
    echo "Directory does not exist, skipping: $dir"
  fi
done

echo "Assets cleaned."
