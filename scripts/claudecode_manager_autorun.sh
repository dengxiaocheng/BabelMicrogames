#!/usr/bin/env sh

set -eu

workdir=""
tmux_socket="claudecode_manager"
session_name="claudecode_manager_autorun"
poll_seconds="60"
quiet_start="1400"
quiet_end="1800"
max_running="1"
timeout_seconds="1800"
worker_prefix=""
daemon="0"
auto_seed_microgames="0"
auto_seed_presets="dianming shuiyuan huijiang yinji bingpeng"

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
    --session-name)
      session_name="$2"
      shift 2
      ;;
    --poll-seconds)
      poll_seconds="$2"
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
    --max-running)
      max_running="$2"
      shift 2
      ;;
    --timeout-seconds)
      timeout_seconds="$2"
      shift 2
      ;;
    --worker-prefix)
      worker_prefix="$2"
      shift 2
      ;;
    --daemon)
      daemon="1"
      shift
      ;;
    --auto-seed-microgames)
      auto_seed_microgames="1"
      shift
      ;;
    --auto-seed-presets)
      auto_seed_presets="$2"
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

quote() {
  printf "'"
  printf "%s" "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

if [ "$daemon" = "1" ]; then
  if tmux -L "$tmux_socket" has-session -t "$session_name" 2>/dev/null; then
    echo "autorun 已在运行: $session_name"
    exit 0
  fi

  command="cd $(quote "$workdir") && exec sh scripts/claudecode_manager_autorun.sh --workdir $(quote "$workdir") --tmux-socket $(quote "$tmux_socket") --session-name $(quote "$session_name") --poll-seconds $(quote "$poll_seconds") --quiet-start $(quote "$quiet_start") --quiet-end $(quote "$quiet_end") --max-running $(quote "$max_running") --timeout-seconds $(quote "$timeout_seconds")"
  if [ -n "$worker_prefix" ]; then
    command="$command --worker-prefix $(quote "$worker_prefix")"
  fi
  if [ "$auto_seed_microgames" = "1" ]; then
    command="$command --auto-seed-microgames --auto-seed-presets $(quote "$auto_seed_presets")"
  fi

  tmux -L "$tmux_socket" new-session -d -s "$session_name" "$command"
  echo "autorun 已启动: $session_name"
  exit 0
fi

in_quiet_hours() {
  now=$(date +%H%M)
  if [ "$quiet_start" -le "$quiet_end" ]; then
    [ "$now" -ge "$quiet_start" ] && [ "$now" -lt "$quiet_end" ]
  else
    [ "$now" -ge "$quiet_start" ] || [ "$now" -lt "$quiet_end" ]
  fi
}

has_worker_session() {
  tmux -L "$tmux_socket" list-sessions -F '#S' 2>/dev/null | grep -q '^claudecode_worker_'
}

has_actionable_worker() {
  set -- go run ./cmd/babel-issue-bridge worker-next --shell --max-running "$max_running"
  if [ -n "$worker_prefix" ]; then
    set -- "$@" --worker-prefix "$worker_prefix"
  fi
  "$@" 2>/dev/null | grep -q "^WORKER_ID='"
}

start_next_worker() {
  set -- sh scripts/claudecode_worker_start_tmux.sh \
    --workdir "$workdir" \
    --tmux-socket "$tmux_socket" \
    --max-running "$max_running" \
    --timeout-seconds "$timeout_seconds"
  if [ -n "$worker_prefix" ]; then
    set -- "$@" --worker-prefix "$worker_prefix"
  fi
  "$@"
}

seed_microgame_if_possible() {
  [ "$auto_seed_microgames" = "1" ] || return 1
  sh scripts/claudecode_microgame_autoseed.sh \
    --workdir "$workdir" \
    --presets "$auto_seed_presets"
}

echo "claudecode autorun loop started: workdir=$workdir poll=${poll_seconds}s quiet=${quiet_start}-${quiet_end} auto_seed_microgames=$auto_seed_microgames"

while :; do
  if in_quiet_hours; then
    sleep "$poll_seconds"
    continue
  fi

  if has_worker_session; then
    sleep "$poll_seconds"
    continue
  fi

  if has_actionable_worker; then
    start_next_worker || true
  elif seed_microgame_if_possible; then
    start_next_worker || true
  fi

  sleep "$poll_seconds"
done
