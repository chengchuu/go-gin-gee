#!/usr/bin/env bash

set -u

test_dir=$(cd "$(dirname "$0")" && pwd)
script_path="$test_dir/../batch-set-git-identity.sh"
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/batch-set-git-identity-test.XXXXXX")

cleanup() {
  rm -rf "$temp_root"
}
trap cleanup EXIT HUP INT TERM

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

assert_equals() {
  expected=$1
  actual=$2
  message=$3
  if [ "$actual" != "$expected" ]; then
    printf 'FAIL: %s\nExpected: %s\nActual: %s\n' "$message" "$expected" "$actual" >&2
    exit 1
  fi
}

assert_status() {
  expected=$1
  shift

  "$@" >"$temp_root/command.out" 2>&1
  actual=$?
  if [ "$actual" -ne "$expected" ]; then
    printf 'FAIL: unexpected exit status\nExpected: %s\nActual: %s\nOutput:\n' "$expected" "$actual" >&2
    cat "$temp_root/command.out" >&2
    exit 1
  fi
}

init_repository() {
  repository=$1
  git init -q "$repository" || fail "Could not initialize $repository"
}

projects="$temp_root/Projects With Spaces"
mkdir -p "$projects"

normal_repository="$projects/normal repository"
hidden_repository="$projects/.hidden-repository"
nested_repository="$projects/container/nested-repository"
worktree_source="$temp_root/worktree-source"
worktree_repository="$projects/worktree repository"

init_repository "$normal_repository"
init_repository "$hidden_repository"
init_repository "$nested_repository"
mkdir -p "$projects/not-a-repository"

init_repository "$worktree_source"
git -C "$worktree_source" config user.name "Setup User"
git -C "$worktree_source" config user.email "setup@example.com"
printf 'worktree fixture\n' >"$worktree_source/README.md"
git -C "$worktree_source" add README.md
git -C "$worktree_source" commit -qm "Create fixture"
git -C "$worktree_source" worktree add -q -b test-worktree "$worktree_repository"

expected_name='Test User & Co'
expected_email='test+git@example.com'

assert_status 0 bash "$script_path" \
  --path="$projects" \
  -username "$expected_name" \
  --useremail="$expected_email"

for repository in "$normal_repository" "$hidden_repository" "$worktree_repository"; do
  assert_equals "$expected_name" "$(git -C "$repository" config --local --get user.name)" "Incorrect user.name for $repository"
  assert_equals "$expected_email" "$(git -C "$repository" config --local --get user.email)" "Incorrect user.email for $repository"
done

if git -C "$nested_repository" config --local --get user.name >/dev/null 2>&1; then
  fail "Nested repository should not be configured"
fi

empty_root="$temp_root/empty"
mkdir -p "$empty_root"
assert_status 1 bash "$script_path" --path "$empty_root" --username Name --useremail user@example.com

assert_status 2 bash "$script_path"
assert_status 2 bash "$script_path" --unknown value
assert_status 2 bash "$script_path" --path "$temp_root/missing" --username Name --useremail user@example.com
assert_status 2 bash "$script_path" --path "$projects" --username= --useremail user@example.com
assert_status 2 bash "$script_path" --path "$projects" --username Name --useremail=

failure_root="$temp_root/failure-projects"
broken_repository="$failure_root/a-broken"
later_repository="$failure_root/z-later"
mkdir -p "$broken_repository"
printf 'gitdir: /path/that/does/not/exist\n' >"$broken_repository/.git"
init_repository "$later_repository"

assert_status 1 bash "$script_path" \
  -path="$failure_root" \
  -username="Later User" \
  -useremail="later@example.com"

assert_equals "Later User" "$(git -C "$later_repository" config --local --get user.name)" "Repository after a failure was not configured"
assert_equals "later@example.com" "$(git -C "$later_repository" config --local --get user.email)" "Repository after a failure has the wrong email"

assert_status 0 bash "$script_path" --help

printf 'All batch-set-git-identity checks passed.\n'
