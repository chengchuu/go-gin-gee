#!/usr/bin/env bash

set -u

usage() {
  cat <<'EOF'
Usage:
  batch-set-git-identity.sh -path=PATH -username=NAME -useremail=EMAIL
  batch-set-git-identity.sh --path PATH --username NAME --useremail EMAIL

Options:
  -path, --path           Parent directory containing Git repositories
  -username, --username   Git user.name value
  -useremail, --useremail Git user.email value
  -h, --help              Show this help message
EOF
}

usage_error() {
  printf 'Error: %s\n\n' "$1" >&2
  usage >&2
  exit 2
}

project_path=''
user_name=''
user_email=''

while [ "$#" -gt 0 ]; do
  case "$1" in
    -path=*|--path=*)
      project_path=${1#*=}
      ;;
    -path|--path)
      [ "$#" -ge 2 ] || usage_error "Missing value for $1."
      project_path=$2
      shift
      ;;
    -username=*|--username=*)
      user_name=${1#*=}
      ;;
    -username|--username)
      [ "$#" -ge 2 ] || usage_error "Missing value for $1."
      user_name=$2
      shift
      ;;
    -useremail=*|--useremail=*)
      user_email=${1#*=}
      ;;
    -useremail|--useremail)
      [ "$#" -ge 2 ] || usage_error "Missing value for $1."
      user_email=$2
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage_error "Unknown option: $1"
      ;;
  esac
  shift
done

[ -n "$project_path" ] || usage_error "The path is required."
[ -n "$user_name" ] || usage_error "The user name is required."
[ -n "$user_email" ] || usage_error "The user email is required."
[ -d "$project_path" ] || usage_error "Directory does not exist: $project_path"
command -v git >/dev/null 2>&1 || usage_error "Git is not installed or is not available on PATH."

printf 'Parent path: %s\n' "$project_path"
printf 'Git user.name: %s\n' "$user_name"
printf 'Git user.email: %s\n\n' "$user_email"

processed=0
succeeded=0
failed=0

shopt -s dotglob nullglob
for repository in "$project_path"/*; do
  [ -d "$repository" ] || continue
  if [ ! -d "$repository/.git" ] && [ ! -f "$repository/.git" ]; then
    continue
  fi

  processed=$((processed + 1))
  repository_failed=0
  printf 'Repository: %s\n' "$repository"

  if ! git -C "$repository" config --local user.name "$user_name"; then
    printf '  Failed to set user.name.\n' >&2
    repository_failed=1
  fi

  if ! git -C "$repository" config --local user.email "$user_email"; then
    printf '  Failed to set user.email.\n' >&2
    repository_failed=1
  fi

  actual_name=$(git -C "$repository" config --local --get user.name 2>/dev/null) || actual_name=''
  actual_email=$(git -C "$repository" config --local --get user.email 2>/dev/null) || actual_email=''

  if [ "$actual_name" != "$user_name" ]; then
    printf '  user.name verification failed.\n' >&2
    repository_failed=1
  fi
  if [ "$actual_email" != "$user_email" ]; then
    printf '  user.email verification failed.\n' >&2
    repository_failed=1
  fi

  if [ "$repository_failed" -eq 0 ]; then
    succeeded=$((succeeded + 1))
    printf '  Updated successfully.\n'
  else
    failed=$((failed + 1))
    printf '  Update failed.\n' >&2
  fi
done

printf '\nSummary: processed=%d succeeded=%d failed=%d\n' "$processed" "$succeeded" "$failed"

if [ "$processed" -eq 0 ]; then
  printf 'No immediate child Git repositories were found.\n' >&2
  exit 1
fi

if [ "$failed" -gt 0 ]; then
  exit 1
fi

exit 0
