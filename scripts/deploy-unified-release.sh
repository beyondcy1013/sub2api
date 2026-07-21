#!/usr/bin/env bash
set -euo pipefail

EXPECTED_OLD_SHA=""
EXPECTED_MAIN_PID=""
EXPECTED_FREE_PID=""
ARTIFACT="/home/third_party/sub2api/backend/bin/sub2api-unified.new"
MAIN_SERVICE="sub2api.service"
MAIN_BINARY="/home/third_party/bin/sub2api/sub2api"
FREE_SERVICE="sub2freeApi.service"
FREE_BINARY="/home/third_party/bin/sub2freeApi/sub2freeApi"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --expected-old-sha)
      EXPECTED_OLD_SHA="${2:-}"
      shift 2
      ;;
    --expected-main-pid)
      EXPECTED_MAIN_PID="${2:-}"
      shift 2
      ;;
    --expected-free-pid)
      EXPECTED_FREE_PID="${2:-}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if ! [[ "$EXPECTED_OLD_SHA" =~ ^[0-9a-f]{64}$ ]] ||
   ! [[ "$EXPECTED_MAIN_PID" =~ ^[1-9][0-9]*$ ]] ||
   ! [[ "$EXPECTED_FREE_PID" =~ ^[1-9][0-9]*$ ]]; then
  echo "Usage: $0 --expected-old-sha SHA256 --expected-main-pid PID --expected-free-pid PID" >&2
  exit 2
fi

sha_of() {
  sha256sum "$1" | awk '{print $1}'
}

process_env_value() {
  local pid="$1"
  local key="$2"
  tr '\0' '\n' < "/proc/$pid/environ" | sed -n "s/^${key}=//p" | head -n 1
}

wait_for_http() {
  local port="$1"
  local attempt
  for attempt in $(seq 1 30); do
    if curl -fsS --max-time 2 "http://127.0.0.1:$port/" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}

verify_service() {
  local service="$1"
  local binary="$2"
  local port="$3"
  local profile="$4"
  local database="$5"
  local redis_db="$6"
  local data_dir="$7"
  local scheduler_prefix="${8:-}"
  local pid disk_sha running_sha

  systemctl is-active --quiet "$service"
  wait_for_http "$port"
  ss -ltnp | grep -Eq ":${port}[[:space:]]"

  pid="$(systemctl show -p MainPID --value "$service")"
  test "$pid" -gt 0
  disk_sha="$(sha_of "$binary")"
  running_sha="$(sha_of "/proc/$pid/exe")"
  test "$disk_sha" = "$ARTIFACT_SHA"
  test "$running_sha" = "$ARTIFACT_SHA"
  test "$(process_env_value "$pid" DEPLOYMENT_PROFILE)" = "$profile"
  test "$(process_env_value "$pid" DATABASE_DBNAME)" = "$database"
  test "$(process_env_value "$pid" REDIS_DB)" = "$redis_db"
  test "$(process_env_value "$pid" SERVER_PORT)" = "$port"
  test "$(process_env_value "$pid" DATA_DIR)" = "$data_dir"
  if [ -n "$scheduler_prefix" ]; then
    test "$(process_env_value "$pid" REDIS_SCHEDULER_KEY_PREFIX)" = "$scheduler_prefix"
  fi

  printf 'OK service=%s pid=%s port=%s profile=%s database=%s redis_db=%s sha=%s\n' \
    "$service" "$pid" "$port" "$profile" "$database" "$redis_db" "$disk_sha"
}

install_binary() {
  local binary="$1"
  local swap="${binary}.new.$$"
  install -o root -g root -m 0755 "$ARTIFACT" "$swap"
  mv -f -- "$swap" "$binary"
}

rollback_service() {
  local service="$1"
  local binary="$2"
  local backup="$3"
  local swap="${binary}.rollback.$$"
  echo "ERROR: rolling back $service" >&2
  install -o root -g root -m 0755 "$backup" "$swap"
  mv -f -- "$swap" "$binary"
  systemctl restart "$service"
}

deploy_service() {
  local service="$1"
  local binary="$2"
  local port="$3"
  local profile="$4"
  local database="$5"
  local redis_db="$6"
  local data_dir="$7"
  local scheduler_prefix="${8:-}"
  local expected_pid="$9"
  local backup="${binary}.bak.${TIMESTAMP}"
  local current_pid

  current_pid="$(systemctl show -p MainPID --value "$service")"
  test "$current_pid" = "$expected_pid"
  test "$(sha_of "$binary")" = "$EXPECTED_OLD_SHA"
  test "$(sha_of "/proc/$current_pid/exe")" = "$EXPECTED_OLD_SHA"
  cp -a -- "$binary" "$backup"
  install_binary "$binary"
  if ! systemctl restart "$service" || \
     ! verify_service "$service" "$binary" "$port" "$profile" "$database" "$redis_db" "$data_dir" "$scheduler_prefix"; then
    rollback_service "$service" "$binary" "$backup"
    return 1
  fi
}

test -s "$ARTIFACT"
go version -m "$ARTIFACT" | grep -Eq 'build[[:space:]]+-tags=embed'
go version -m "$ARTIFACT" | grep -Eq 'build[[:space:]]+CGO_ENABLED=0'
ARTIFACT_SHA="$(sha_of "$ARTIFACT")"
test "$ARTIFACT_SHA" != "$EXPECTED_OLD_SHA"

test "$(sha_of "$MAIN_BINARY")" = "$EXPECTED_OLD_SHA"
test "$(sha_of "$FREE_BINARY")" = "$EXPECTED_OLD_SHA"
MAIN_PID="$(systemctl show -p MainPID --value "$MAIN_SERVICE")"
FREE_PID="$(systemctl show -p MainPID --value "$FREE_SERVICE")"
test "$(sha_of "/proc/$MAIN_PID/exe")" = "$EXPECTED_OLD_SHA"
test "$(sha_of "/proc/$FREE_PID/exe")" = "$EXPECTED_OLD_SHA"

deploy_service \
  "$MAIN_SERVICE" "$MAIN_BINARY" 18381 main sub2api 0 \
  /home/third_party/sub2api/deploy/data '' "$EXPECTED_MAIN_PID"
deploy_service \
  "$FREE_SERVICE" "$FREE_BINARY" 18382 free sub2freeApi 1 \
  /home/third_party/sub2freeApi/deploy/data sub2freeApi "$EXPECTED_FREE_PID"

MAIN_PID="$(systemctl show -p MainPID --value "$MAIN_SERVICE")"
FREE_PID="$(systemctl show -p MainPID --value "$FREE_SERVICE")"
sha256sum "$MAIN_BINARY" "$FREE_BINARY" "/proc/$MAIN_PID/exe" "/proc/$FREE_PID/exe"
