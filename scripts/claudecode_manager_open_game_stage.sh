#!/usr/bin/env sh

set -eu

game_workdir=""
repo=""
title=""
report=""
decision_request=""
manager_workdir="${CLAUDECODE_MANAGER_WORKDIR:-/home/openclaw/claudecode-manager}"
manager_thread_id="${CLAUDECODE_MANAGER_THREAD_ID:-019dbdc4-8d17-7600-8de1-cdd9d510fa97}"
watcher_session_name=""
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"

while [ $# -gt 0 ]; do
  case "$1" in
    --game-workdir)
      game_workdir="$2"
      shift 2
      ;;
    --repo)
      repo="$2"
      shift 2
      ;;
    --title)
      title="$2"
      shift 2
      ;;
    --report)
      report="$2"
      shift 2
      ;;
    --decision-request)
      decision_request="$2"
      shift 2
      ;;
    --manager-workdir)
      manager_workdir="$2"
      shift 2
      ;;
    --manager-thread-id)
      manager_thread_id="$2"
      shift 2
      ;;
    --watcher-session-name)
      watcher_session_name="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

[ -n "$game_workdir" ] || { echo "missing --game-workdir" >&2; exit 2; }
[ -n "$repo" ] || { echo "missing --repo" >&2; exit 2; }
[ -n "$title" ] || { echo "missing --title" >&2; exit 2; }
[ -n "$report" ] || { echo "missing --report" >&2; exit 2; }
[ -n "$decision_request" ] || { echo "missing --decision-request" >&2; exit 2; }
[ -x "$bridge_cmd" ] || { echo "missing bridge command: $bridge_cmd" >&2; exit 1; }

if [ -z "$watcher_session_name" ]; then
  slug=$(basename "$game_workdir" | tr -c '[:alnum:]_-' '_')
  watcher_session_name="claudecode_manager_watch_${slug}"
fi

exec "$bridge_cmd" open-stage \
  --repo "$repo" \
  --workdir "$game_workdir" \
  --resume-workdir "$manager_workdir" \
  --thread-id "$manager_thread_id" \
  --watcher-session-name "$watcher_session_name" \
  --title "$title" \
  --report "$report" \
  --decision-request "$decision_request"
