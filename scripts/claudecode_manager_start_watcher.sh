#!/usr/bin/env sh

set -eu

workdir=""
session_name="claudecode_manager_watch"
resume_prefix="claudecode_manager_resume"
poll_seconds="10"
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
    --resume-prefix)
      resume_prefix="$2"
      shift 2
      ;;
    --poll-seconds)
      poll_seconds="$2"
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
sh scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"

if [ -n "$tmux_socket" ]; then
  quote() {
    printf "'"
    printf "%s" "$1" | sed "s/'/'\\\\''/g"
    printf "'"
  }
  if tmux -L "$tmux_socket" has-session -t "$session_name" 2>/dev/null; then
    echo "watcher 已在运行: $session_name"
    exit 0
  fi
  command="cd $(quote "$workdir") && exec go run ./cmd/babel-issue-bridge watch --poll-seconds $(quote "$poll_seconds") --watcher-session-name $(quote "$session_name") --resume-session-prefix $(quote "$resume_prefix") --resume-mode exec"
  tmux -L "$tmux_socket" new-session -d -s "$session_name" "$command"
  echo "watcher 已启动: $session_name"
  exit 0
fi

go run ./cmd/babel-issue-bridge start-watcher \
  --session-name "$session_name" \
  -- \
  --poll-seconds "$poll_seconds" \
  --watcher-session-name "$session_name" \
  --resume-session-prefix "$resume_prefix" \
  --resume-mode exec
