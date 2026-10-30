#!/usr/bin/env bash
set -euo pipefail

test_dir=$(cd "$(dirname "$0")" && pwd)
script="$test_dir/../fetch-complete-branches.sh"
fixture=$(mktemp -d "${TMPDIR:-/tmp}/complete-branches-test.XXXXXX")
trap 'rm -rf -- "$fixture"' EXIT

assert_status() {
  local expected=$1 status=0
  shift
  "$@" >"$fixture/output" 2>&1 || status=$?
  if [ "$status" -ne "$expected" ]; then
    cat "$fixture/output" >&2
    printf 'Expected status %s, got %s\n' "$expected" "$status" >&2
    exit 1
  fi
}

git init -q -b main "$fixture/source"
git -C "$fixture/source" config user.name Fixture
git -C "$fixture/source" config user.email fixture@example.com
git -C "$fixture/source" config commit.gpgsign false
printf 'old contents\n' >"$fixture/source/song.txt"
git -C "$fixture/source" add .
git -C "$fixture/source" commit -qm initial
old_blob=$(git -C "$fixture/source" rev-parse HEAD:song.txt)
printf 'new contents\n' >"$fixture/source/song.txt"
git -C "$fixture/source" commit -qam update
git -C "$fixture/source" switch -qc feature/nested
printf 'feature\n' >"$fixture/source/feature.txt"
git -C "$fixture/source" add .
git -C "$fixture/source" commit -qm feature
git -C "$fixture/source" switch -q main
git -C "$fixture/source" config uploadpack.allowFilter true

git clone -q --no-checkout --single-branch --filter=blob:none "file://$fixture/source" "$fixture/partial clone"
repo="$fixture/partial clone"
GIT_NO_LAZY_FETCH=1 git -C "$repo" rev-list --objects --all --missing=print >"$fixture/missing"
grep -q '^?' "$fixture/missing"
git -C "$repo" checkout -q main
printf 'local edits\n' >"$repo/song.txt"
printf 'untracked\n' >"$repo/local.txt"
before=$(git -C "$repo" status --porcelain)
head_before=$(git -C "$repo" rev-parse HEAD)
assert_status 0 bash "$script" --path "$repo"
[ "$(git -C "$repo" status --porcelain)" = "$before" ]
[ "$(git -C "$repo" rev-parse HEAD)" = "$head_before" ]
git -C "$repo" rev-parse --verify refs/remotes/origin/feature/nested >/dev/null
[ "$(git -C "$repo" cat-file blob "$old_blob")" = 'old contents' ]
assert_status 1 git -C "$repo" config --local --get remote.origin.promisor
assert_status 1 git -C "$repo" config --local --get remote.origin.partialclonefilter
GIT_NO_LAZY_FETCH=1 git -C "$repo" rev-list --objects --all --missing=print >"$fixture/missing"
if grep -q '^?' "$fixture/missing"; then exit 1; fi
assert_status 0 bash "$script" --path="$repo"

git clone -q --depth=1 --single-branch "file://$fixture/source" "$fixture/shallow"
assert_status 0 bash "$script" --path "$fixture/shallow"
[ "$(git -C "$fixture/shallow" rev-parse --is-shallow-repository)" = false ]
git -C "$fixture/shallow" cat-file -e "$old_blob"

git clone -q --no-checkout --filter=blob:none "file://$fixture/source" "$fixture/failure"
git -C "$fixture/failure" remote set-url origin "$fixture/missing-origin"
assert_status 128 bash "$script" --path "$fixture/failure"
[ "$(git -C "$fixture/failure" config --get remote.origin.promisor)" = true ]
[ "$(git -C "$fixture/failure" config --get remote.origin.partialclonefilter)" = blob:none ]

# Download missing local-only history when origin still has the objects.
git clone -q --no-checkout --filter=blob:none "file://$fixture/source" "$fixture/incomplete"
git -C "$fixture/incomplete" branch local-only origin/feature/nested >/dev/null
git -C "$fixture/incomplete" update-ref -d refs/remotes/origin/feature/nested
git -C "$fixture/source" branch -D feature/nested >/dev/null
assert_status 0 bash "$script" --path "$fixture/incomplete"
GIT_NO_LAZY_FETCH=1 git -C "$fixture/incomplete" rev-list --objects --all --missing=print >"$fixture/missing"
if grep -q '^?' "$fixture/missing"; then exit 1; fi

# An unchanged, multi-branch partial clone must hydrate blobs without a full refetch.
git clone -q --no-checkout --filter=blob:none "file://$fixture/source" "$fixture/unchanged"
assert_status 0 bash "$script" --path "$fixture/unchanged"
grep -q 'Downloading .* missing Git objects' "$fixture/output"

# Fail specifically at the batch-download process boundary.
git clone -q --no-checkout --filter=blob:none "file://$fixture/source" "$fixture/batch-failure"
mkdir "$fixture/bin"
printf '%s\n' '#!/usr/bin/env bash' \
  'for arg in "$@"; do if [ "$arg" = --stdin ]; then exit 42; fi; done' \
  'exec "$FETCH_TEST_GIT" "$@"' >"$fixture/bin/git"
chmod +x "$fixture/bin/git"
assert_status 42 env FETCH_TEST_GIT="$(command -v git)" PATH="$fixture/bin:$PATH" \
  bash "$script" --path "$fixture/batch-failure"
[ "$(git -C "$fixture/batch-failure" config --get remote.origin.promisor)" = true ]
[ "$(git -C "$fixture/batch-failure" config --get remote.origin.partialclonefilter)" = blob:none ]

assert_status 2 bash "$script"
assert_status 2 bash "$script" --unknown
assert_status 2 bash "$script" --path
assert_status 0 bash "$script" --help
printf 'All complete-branch fetch integration tests passed.\n'
