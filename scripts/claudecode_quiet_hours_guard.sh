#!/usr/bin/env sh

set -eu

workdir=""
tmux_socket="claudecode_manager"
watcher_session="claudecode_manager_watch"
quiet_start="1400"
quiet_end="1800"
poll_seconds="30"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --tmux-socket)
      tmux_socket="$2"
      shift 2
      ;;
    --watcher-session)
      watcher_session="$2"
      shift 2
      ;;
    --quiet-start)
      quiet_start="$2"
      shift 2
      ;;
    --quiet-end)
      quiet_end="$2"
      shift 2
      ;;
    --poll-seconds)
      poll_seconds="$2"
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

now=$(date +%H%M)
in_quiet="0"
if [ "$quiet_start" -le "$quiet_end" ]; then
  if [ "$now" -ge "$quiet_start" ] && [ "$now" -lt "$quiet_end" ]; then
    in_quiet="1"
  fi
else
  if [ "$now" -ge "$quiet_start" ] || [ "$now" -lt "$quiet_end" ]; then
    in_quiet="1"
  fi
fi

list_sessions() {
  tmux -L "$tmux_socket" list-sessions -F '#S' 2>/dev/null || true
}

kill_session() {
  tmux -L "$tmux_socket" kill-session -t "$1" 2>/dev/null || true
}

mark_running_rework() {
  [ -f .codex-runtime/claudecode_workers.json ] || return 0
  jq -r '.workers[] | select(.status=="running") | .worker_id' .codex-runtime/claudecode_workers.json 2>/dev/null |
    while IFS= read -r worker_id; do
      [ -n "$worker_id" ] || continue
      go run ./cmd/babel-issue-bridge worker-set-status \
        --worker-id "$worker_id" \
        --status rework \
        --note "paused by daily ClaudeCode quiet hours ${quiet_start}-${quiet_end}" >/dev/null || true
    done
}

if [ "$in_quiet" = "1" ]; then
  list_sessions | while IFS= read -r session; do
    case "$session" in
      "$watcher_session"|claudecode_worker_*|claudecode_manager_resume_*)
        kill_session "$session"
        ;;
    esac
  done
  mark_running_rework
  echo "quiet-hours active: $now"
  exit 0
fi

sh scripts/claudecode_manager_start_watcher.sh \
  --workdir "$workdir" \
  --session-name "$watcher_session" \
  --tmux-socket "$tmux_socket" \
  --poll-seconds "$poll_seconds" >/dev/null
echo "quiet-hours inactive: $now"
