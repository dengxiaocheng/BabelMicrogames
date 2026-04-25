#!/usr/bin/env sh

set -eu

workdir=""
session_name="claudecode_manager_watch"
tmux_socket="claudecode_manager"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --session-name)
      session_name="$2"
      shift 2
      ;;
    --tmux-socket)
      tmux_socket="$2"
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
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"
[ -x "$bridge_cmd" ] || {
  echo "missing bridge command: $bridge_cmd" >&2
  exit 1
}

if [ -n "$tmux_socket" ]; then
  if ! tmux -L "$tmux_socket" has-session -t "$session_name" 2>/dev/null; then
    echo "watcher 不存在: $session_name"
    exit 0
  fi
  tmux -L "$tmux_socket" kill-session -t "$session_name"
  echo "watcher 已停止: $session_name"
  exit 0
fi

"$bridge_cmd" stop-watcher --session-name "$session_name"
