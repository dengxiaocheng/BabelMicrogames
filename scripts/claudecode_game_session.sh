#!/usr/bin/env sh

set -eu

workdir=""
session_file=".codex-runtime/claude_session_id"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --session-file)
      session_file="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

if [ -z "$workdir" ]; then
  workdir=$(pwd)
fi

cd "$workdir"
mkdir -p "$(dirname "$session_file")"

if [ ! -s "$session_file" ]; then
  if [ -r /proc/sys/kernel/random/uuid ]; then
    cat /proc/sys/kernel/random/uuid > "$session_file"
  elif command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]' > "$session_file"
  else
    echo "missing uuid source" >&2
    exit 1
  fi
fi

tr -d '[:space:]' < "$session_file"
printf '\n'
