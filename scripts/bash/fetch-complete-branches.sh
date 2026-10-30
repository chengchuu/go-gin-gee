#!/usr/bin/env bash
# bash scripts/bash/fetch-complete-branches.sh --path "/Users/Example"

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: fetch-complete-branches.sh --path PATH

Fetch all origin branches and their complete Git history, including file contents.
Requires Git 2.36+ and Bash 3.2+. Does not check out or merge branches.
Git LFS payloads and submodule repositories are not downloaded.
EOF
}

usage_error() {
  printf 'Error: %s\n' "$1" >&2
  usage >&2
  exit 2
}

repository=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --path=*|-path=*) repository=${1#*=} ;;
    --path|-path)
      [ "$#" -ge 2 ] || usage_error "Missing path value."
      repository=$2
      shift
      ;;
    -h|--help) usage; exit 0 ;;
    *) usage_error "Unknown argument: $1" ;;
  esac
  shift
done
[ -n "$repository" ] || usage_error "A repository path is required."
git -C "$repository" rev-parse --git-dir >/dev/null
git -C "$repository" remote get-url origin >/dev/null

unset_config() {
  local status
  if git -C "$repository" config --local --unset-all "$1"; then
    return
  else
    status=$?
    # Git returns 5 when the key is already absent.
    [ "$status" -eq 5 ] || return "$status"
  fi
}

fetch_options=(--no-filter --no-tags --no-recurse-submodules --refmap=)
if [ "$(git -C "$repository" rev-parse --is-shallow-repository)" = true ]; then
  fetch_options+=(--unshallow)
fi

printf 'Fetching complete history for all origin branches...\n'
git -C "$repository" fetch "${fetch_options[@]}" origin '+refs/heads/*:refs/remotes/origin/*'
if [ "$(git -C "$repository" rev-parse --is-shallow-repository)" = true ]; then
  printf 'Error: origin did not provide complete history; partial-clone settings retained.\n' >&2
  exit 1
fi

verification_dir=$(mktemp -d "${TMPDIR:-/tmp}/fetch-complete-branches.XXXXXX")
trap 'rm -rf -- "$verification_dir"' EXIT
verification="$verification_dir/objects"
missing="$verification_dir/missing"
previous_missing="$verification_dir/previous"
: >"$previous_missing"

# Check every locally referenced object before disabling the promisor remote.
# This also protects local-only history and refs belonging to other remotes.
printf 'Verifying local Git objects without fetching on demand...\n'
while :; do
  GIT_NO_LAZY_FETCH=1 git -C "$repository" rev-list --objects --all --missing=print >"$verification"
  LC_ALL=C sed -n 's/^?//p' "$verification" | LC_ALL=C sort >"$missing"
  [ -s "$missing" ] || break
  if cmp -s "$missing" "$previous_missing"; then
    printf 'Error: missing-object download made no progress; partial-clone settings retained.\n' >&2
    exit 1
  fi
  cp "$missing" "$previous_missing"
  printf 'Downloading %s missing Git objects...\n' "$(wc -l <"$missing" | tr -d ' ')"
  # Send object IDs over stdin to avoid shell argument limits and per-object fetches.
  git -C "$repository" fetch --no-filter --no-tags --no-write-fetch-head \
    --no-recurse-submodules --refmap= origin --stdin <"$missing"
done

git -C "$repository" config --local --replace-all remote.origin.fetch '+refs/heads/*:refs/remotes/origin/*'
unset_config remote.origin.partialclonefilter
unset_config remote.origin.promisor
printf 'Complete: all origin branches fetched and locally referenced Git objects verified.\n'
