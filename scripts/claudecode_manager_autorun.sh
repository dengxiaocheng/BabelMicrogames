#!/usr/bin/env sh

set -eu

workdir=""
manager_workdir="/home/openclaw/claudecode-manager"
manager_repo="${CLAUDECODE_MANAGER_REPO:-dengxiaocheng/BabelMicrogames}"
game_root="/home/openclaw/babel-microgames"
game_workdir=""
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
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --manager-workdir)
      manager_workdir="$2"
      shift 2
      ;;
    --game-root)
      game_root="$2"
      shift 2
      ;;
    --game-workdir)
      game_workdir="$2"
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

abs_dir() {
  (cd "$1" && pwd -P)
}

repo_name_for() {
  git -C "$1" remote get-url origin 2>/dev/null |
    sed -e 's#^git@github.com:##' \
        -e 's#^https://github.com/##' \
        -e 's#^http://github.com/##' \
        -e 's#\.git$##'
}

guard_control_workdir() {
  repo=$(repo_name_for "$1" || true)
  if [ "$repo" = "$manager_repo" ]; then
    sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$1" --expected-repo "$manager_repo"
  else
    sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$1"
  fi
}

workdir=$(abs_dir "$workdir")
manager_workdir=$(abs_dir "$manager_workdir")
if [ -n "$game_workdir" ]; then
  game_workdir=$(abs_dir "$game_workdir")
fi

cd "$workdir"
guard_control_workdir "$workdir"
[ -x "$bridge_cmd" ] || {
  echo "missing bridge command: $bridge_cmd" >&2
  exit 1
}

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

  command="cd $(quote "$workdir") && BRIDGE_CMD=$(quote "$bridge_cmd") exec sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_autorun.sh --workdir $(quote "$workdir") --manager-workdir $(quote "$manager_workdir") --game-root $(quote "$game_root") --tmux-socket $(quote "$tmux_socket") --session-name $(quote "$session_name") --poll-seconds $(quote "$poll_seconds") --quiet-start $(quote "$quiet_start") --quiet-end $(quote "$quiet_end") --max-running $(quote "$max_running") --timeout-seconds $(quote "$timeout_seconds")"
  if [ -n "$game_workdir" ]; then
    command="$command --game-workdir $(quote "$game_workdir")"
  fi
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

has_actionable_worker_in() {
  candidate_workdir="$1"
  sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$candidate_workdir" >/dev/null
  set -- "$bridge_cmd" worker-next --shell --max-running "$max_running"
  if [ -n "$worker_prefix" ]; then
    set -- "$@" --worker-prefix "$worker_prefix"
  fi
  (cd "$candidate_workdir" && "$@") 2>/dev/null | grep -q "^WORKER_ID='"
}

select_dispatch_workdir() {
  if [ -n "$game_workdir" ]; then
    if has_actionable_worker_in "$game_workdir"; then
      printf '%s\n' "$game_workdir"
      return 0
    fi
    return 1
  fi

  repo=$(repo_name_for "$workdir" || true)
  case "$repo" in
    dengxiaocheng/BabelMicrogame-*)
      if has_actionable_worker_in "$workdir"; then
        printf '%s\n' "$workdir"
        return 0
      fi
      ;;
  esac

  if [ -d "$game_root" ]; then
    for candidate in "$game_root"/*; do
      [ -d "$candidate/.git" ] || continue
      sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$candidate" >/dev/null 2>&1 || continue
      if has_actionable_worker_in "$candidate"; then
        printf '%s\n' "$candidate"
        return 0
      fi
    done
  fi

  return 1
}

start_next_worker() {
  dispatch_workdir="$1"
  set -- sh /home/openclaw/claudecode-manager/scripts/claudecode_worker_start_tmux.sh \
    --workdir "$dispatch_workdir" \
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
  sh /home/openclaw/claudecode-manager/scripts/claudecode_microgame_autoseed.sh \
    --workdir "$manager_workdir" \
    --presets "$auto_seed_presets"
}

refresh_state() {
  if [ -x "$manager_workdir/scripts/claudecode_manager_refresh_state.sh" ]; then
    sh "$manager_workdir/scripts/claudecode_manager_refresh_state.sh" \
      --manager-workdir "$manager_workdir" \
      --game-root "$game_root" \
      --tmux-socket "$tmux_socket" \
      --quiet || true
  fi
}

echo "claudecode autorun loop started: manager_workdir=$manager_workdir workdir=$workdir game_root=$game_root poll=${poll_seconds}s quiet=${quiet_start}-${quiet_end} auto_seed_microgames=$auto_seed_microgames"

while :; do
  refresh_state

  if in_quiet_hours; then
    sleep "$poll_seconds"
    continue
  fi

  if has_worker_session; then
    sleep "$poll_seconds"
    continue
  fi

  dispatch_workdir=""
  if dispatch_workdir=$(select_dispatch_workdir); then
    start_next_worker "$dispatch_workdir" || true
  elif seed_microgame_if_possible; then
    if dispatch_workdir=$(select_dispatch_workdir); then
      start_next_worker "$dispatch_workdir" || true
    fi
  fi

  sleep "$poll_seconds"
done
