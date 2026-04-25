#!/usr/bin/env sh

set -eu

workdir=""
session_name="claudecode_manager_watch"
tmux_socket="claudecode_manager"

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

if [ -n "$tmux_socket" ]; then
  if ! tmux -L "$tmux_socket" has-session -t "$session_name" 2>/dev/null; then
    echo "watcher 不存在: $session_name"
    exit 0
  fi
  tmux -L "$tmux_socket" kill-session -t "$session_name"
  echo "watcher 已停止: $session_name"
  exit 0
fi

go run ./cmd/babel-issue-bridge stop-watcher --session-name "$session_name"
