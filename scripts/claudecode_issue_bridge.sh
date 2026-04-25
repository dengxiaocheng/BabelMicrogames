#!/usr/bin/env sh

set -eu

workdir="${CLAUDECODE_BRIDGE_WORKDIR:-$(pwd)}"
stage_bridge="${BABEL_RUNTIME_STAGE_BRIDGE:-/home/openclaw/babel-runtime/scripts/stage_issue_bridge.sh}"

[ -f "$stage_bridge" ] || {
  echo "missing s stage issue bridge: $stage_bridge" >&2
  exit 1
}

exec sh "$stage_bridge" --workdir "$workdir" -- "$@"
