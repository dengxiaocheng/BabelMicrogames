#!/usr/bin/env sh

set -eu

workdir=""
session_id=""
model=""
allowed_tools=""
permission_flag="--dangerously-skip-permissions"
worker_id=""
task_title=""
task_summary=""
packet_file=""
send_packet="0"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/scripts/claudecode_issue_bridge.sh}"

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
    --allowed-tools)
      allowed_tools="$2"
      shift 2
      ;;
    --worker-id)
      worker_id="$2"
      shift 2
      ;;
    --task-title)
      task_title="$2"
      shift 2
      ;;
    --task-summary)
      task_summary="$2"
      shift 2
      ;;
    --packet-file)
      packet_file="$2"
      shift 2
      ;;
    --send-packet)
      send_packet="1"
      shift
      ;;
    --safe-permissions)
      permission_flag=""
      shift
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

if [ -n "$worker_id" ] && [ -z "$packet_file" ]; then
  packet_file=".codex-runtime/claudecode_workers/${worker_id}/packet.md"
fi

if [ -n "$worker_id" ]; then
  set -- "$bridge_cmd" worker-start --worker-id "$worker_id"
  if [ -n "$session_id" ]; then
    set -- "$@" --session-id "$session_id"
  fi
  if [ -n "$model" ]; then
    set -- "$@" --model "$model"
  fi
  if [ -n "$task_title" ]; then
    set -- "$@" --task-title "$task_title"
  fi
  if [ -n "$task_summary" ]; then
    set -- "$@" --task-summary "$task_summary"
  fi
  "$@"
fi

if [ "$send_packet" = "1" ] && [ -n "$packet_file" ] && [ -f "$packet_file" ]; then
  packet_content=$(cat "$packet_file")
  if [ -n "$packet_content" ]; then
    set -- claude
    if [ -n "$permission_flag" ]; then
      set -- "$@" "$permission_flag"
    fi
    if [ -n "$model" ]; then
      set -- "$@" --model "$model"
    fi
    if [ -n "$allowed_tools" ]; then
      set -- "$@" --allowedTools "$allowed_tools"
    fi
    if [ -n "$session_id" ]; then
      set -- "$@" -p "$packet_content" --resume "$session_id"
      "$@"
    else
      exec "$@" "$packet_content"
    fi
  fi
fi

if [ -n "$packet_file" ] && [ -f "$packet_file" ]; then
  echo "worker packet: $packet_file" >&2
fi

set -- claude
if [ -n "$permission_flag" ]; then
  set -- "$@" "$permission_flag"
fi
if [ -n "$model" ]; then
  set -- "$@" --model "$model"
fi
if [ -n "$allowed_tools" ]; then
  set -- "$@" --allowedTools "$allowed_tools"
fi
if [ -n "$session_id" ]; then
  set -- "$@" --resume "$session_id"
fi

exec "$@"
