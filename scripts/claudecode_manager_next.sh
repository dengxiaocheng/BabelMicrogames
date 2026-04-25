#!/usr/bin/env sh

set -eu

workdir=""
session_id=""
model=""
send_packet="0"
print_only="0"
all_actionable="0"
max_running="1"
allow_same_lane="0"
worker_prefix=""
run_once="0"
timeout_seconds="1800"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --model)
      model="$2"
      shift 2
      ;;
    --send-packet)
      send_packet="1"
      shift
      ;;
    --print-only)
      print_only="1"
      shift
      ;;
    --all-actionable)
      all_actionable="1"
      shift
      ;;
    --max-running)
      max_running="$2"
      shift 2
      ;;
    --allow-same-lane)
      allow_same_lane="1"
      shift
      ;;
    --worker-prefix)
      worker_prefix="$2"
      shift 2
      ;;
    --run-once)
      run_once="1"
      shift
      ;;
    --timeout-seconds)
      timeout_seconds="$2"
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

set -- "$bridge_cmd" worker-next --shell
if [ "$all_actionable" = "1" ]; then
  set -- "$@" --all-actionable
fi
set -- "$@" --max-running "$max_running"
if [ -n "$worker_prefix" ]; then
  set -- "$@" --worker-prefix "$worker_prefix"
fi
if [ "$allow_same_lane" = "1" ]; then
  set -- "$@" --allow-same-lane
fi

worker_env=$("$@")
eval "$worker_env"

echo "next worker: $WORKER_ID [$STATUS]" >&2
if [ -n "${LANE:-}" ]; then
  echo "lane: $LANE" >&2
fi
if [ -n "${TASK_TITLE:-}" ]; then
  echo "task: $TASK_TITLE" >&2
fi
if [ -n "${PACKET_FILE:-}" ]; then
  echo "packet: $PACKET_FILE" >&2
fi

if [ "$print_only" = "1" ]; then
  printf '%s\n' "$worker_env"
  exit 0
fi

if [ -z "$session_id" ] && [ -n "${SESSION_ID:-}" ]; then
  session_id="$SESSION_ID"
fi
if [ -z "$model" ] && [ -n "${MODEL:-}" ]; then
  model="$MODEL"
fi
if [ "$send_packet" = "0" ] && [ -z "$session_id" ] && [ -n "${PACKET_FILE:-}" ] && [ -f "$PACKET_FILE" ]; then
  send_packet="1"
fi

if [ "$run_once" = "1" ]; then
  set -- sh /home/openclaw/claudecode-manager/scripts/claudecode_worker_run_once.sh --workdir "$workdir" --worker-id "$WORKER_ID" --timeout-seconds "$timeout_seconds"
else
  set -- sh /home/openclaw/claudecode-manager/scripts/claudecode_worker_resume.sh --workdir "$workdir" --worker-id "$WORKER_ID"
fi
if [ -n "$session_id" ]; then
  set -- "$@" --session-id "$session_id"
fi
if [ -n "$model" ]; then
  set -- "$@" --model "$model"
fi
if [ "$run_once" = "0" ] && [ "$send_packet" = "1" ]; then
  set -- "$@" --send-packet
fi

exec "$@"
