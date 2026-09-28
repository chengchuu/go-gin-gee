#!/usr/bin/env bash
set -euo pipefail

image=${1:?Usage: bash scripts/docker-smoke-test.sh IMAGE}
container=

cleanup() {
  local status=$?
  if [[ -n "$container" ]]; then
    if (( status != 0 )); then
      docker logs "$container" >&2 || true
    fi
    if ! docker rm --force --volumes "$container" >/dev/null; then
      status=1
    fi
  fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Use only the image's disposable filesystem and loopback interface.
container=$(docker run --detach --network none "$image")

request() {
  docker exec "$container" curl --fail --silent --show-error \
    --connect-timeout 2 --max-time 5 "$@"
}

check_health() {
  request --retry 30 --retry-connrefused --retry-delay 1 --retry-max-time 60 \
    http://127.0.0.1:3000/api/ping |
    python3 -c 'import json, sys; sys.exit(not json.load(sys.stdin)["message"].startswith("pong/"))'
}

check_health
request http://127.0.0.1:3000/docs/index.html >/dev/null
created=$(request --header 'Content-Type: application/json' \
  --data '{"ori_link":"https://example.invalid/container-smoke","base_url":"http://localhost:3000"}' \
  http://127.0.0.1:3000/api/gee/generate-short-link)
key=$(python3 -c '
import json, sys
payload = json.load(sys.stdin)
link = payload["data"]
if "link" in payload:
    sys.exit("response contains removed link field")
if payload["tiny_link"] != link:
    sys.exit("deprecated tiny_link alias differs from data")
prefix = "http://localhost:3000/t/"
if not link.startswith(prefix) or not link[len(prefix):]:
    sys.exit("unexpected short-link response")
print(link[len(prefix):])
' <<< "$created")
before=$(docker exec "$container" cat /web/log/api.log)
test -n "$before"

docker restart "$container" >/dev/null
check_health
request --get --data-urlencode "link_key=$key" \
  http://127.0.0.1:3000/api/gee/query-short-link |
  python3 -c 'import json, sys; sys.exit(json.load(sys.stdin)["ori_link"] != "https://example.invalid/container-smoke")'
after=$(docker exec "$container" cat /web/log/api.log)
[[ "$after" == "$before"$'\n'* ]]

printf 'Container smoke tests passed: health, Swagger, SQLite persistence, and append-safe logs.\n'
