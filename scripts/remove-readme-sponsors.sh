#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
MODE="apply"

if [ "${1:-}" = "--check" ]; then
  MODE="check"
elif [ "$#" -ne 0 ]; then
  echo "Usage: $0 [--check]" >&2
  exit 2
fi

strip_section() {
  local file="$1"
  local start_pattern="$2"
  local end_heading="$3"
  local temp_file

  if ! grep -Eq "$start_pattern" "$file"; then
    return 0
  fi
  if [ "$MODE" = "check" ]; then
    echo "README sponsor section is not allowed: $file" >&2
    return 1
  fi

  temp_file="$(mktemp "${file}.tmp.XXXXXX")"
  if ! awk -v start="$start_pattern" -v finish="$end_heading" '
    $0 ~ start { skipping = 1; found_start = 1; next }
    skipping && $0 == finish { skipping = 0; found_end = 1; print; next }
    !skipping { print }
    END {
      if (!found_start || !found_end || skipping) exit 42
    }
  ' "$file" > "$temp_file"; then
    rm -f -- "$temp_file"
    echo "Unable to remove complete sponsor section from $file" >&2
    return 1
  fi

  chmod --reference="$file" "$temp_file"
  mv -f -- "$temp_file" "$file"
  echo "Removed README sponsor section: ${file#"$ROOT_DIR"/}"
}

strip_section "$ROOT_DIR/README.md" '^## .*Sponsors$' '## Overview'
strip_section "$ROOT_DIR/README_CN.md" '^## .*赞助商$' '## 项目概述'
strip_section "$ROOT_DIR/README_JA.md" '^## .*スポンサー$' '## 概要'
