#!/usr/bin/env sh

set -eu

workdir=""
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/scripts/claudecode_issue_bridge.sh}"

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
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"
[ -x "$bridge_cmd" ] || {
  echo "missing bridge command: $bridge_cmd" >&2
  exit 1
}

exec "$bridge_cmd" worker-packet "$@"
