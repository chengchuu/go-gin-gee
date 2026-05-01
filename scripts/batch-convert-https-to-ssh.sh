#!/bin/bash
# batch-convert-https-to-ssh.sh

PROJECT_PATH="web"  # change this
ROOT="$HOME/$PROJECT_PATH"
find "$ROOT" -type d -name .git -prune | while read -r gitdir; do
  repo="${gitdir%/.git}"
  echo "==> $repo"
  git -C "$repo" remote -v | awk '{print $1}' | sort -u | while read -r r; do
    url="$(git -C "$repo" remote get-url "$r" 2>/dev/null || true)"
    case "$url" in
      https://github.com/*)
        new="git@github.com:${url#https://github.com/}"
        git -C "$repo" remote set-url "$r" "$new"
        echo "  $r: $url -> $new"
        ;;
      *)
        echo "  $r: skip ($url)"
        ;;
    esac
  done
done
