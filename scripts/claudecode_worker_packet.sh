#!/usr/bin/env sh

set -eu

workdir=""

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

if [ -z "$workdir" ]; then
  workdir=$(pwd)
fi

cd "$workdir"
sh scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"

exec go run ./cmd/babel-issue-bridge worker-packet "$@"
