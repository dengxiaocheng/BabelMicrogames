#!/usr/bin/env sh

set -eu

manager_workdir="${CLAUDECODE_MANAGER_WORKDIR:-/home/openclaw/claudecode-manager}"
repo="${CLAUDECODE_MANAGER_REPO:-dengxiaocheng/BabelMicrogames}"
manager_thread_id="${CLAUDECODE_MANAGER_THREAD_ID:-019dbdc4-8d17-7600-8de1-cdd9d510fa97}"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/scripts/claudecode_issue_bridge.sh}"
state_file="${CLAUDECODE_MANAGER_AUDIT_STATE_FILE:-.codex-runtime/manager_audit_issue_state.json}"
lock_file="${CLAUDECODE_MANAGER_AUDIT_LOCK_FILE:-.codex-runtime/manager_audit_issue.lock}"
title=""
report=""
report_file=""
comment=""
comment_file=""
decision_request="Codex manager audit issue. This issue is opened and closed to record the manager-side handoff."
dry_run="0"

while [ $# -gt 0 ]; do
  case "$1" in
    --manager-workdir)
      manager_workdir="$2"
      shift 2
      ;;
    --repo)
      repo="$2"
      shift 2
      ;;
    --manager-thread-id)
      manager_thread_id="$2"
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
    --report-file)
      report_file="$2"
      shift 2
      ;;
    --comment)
      comment="$2"
      shift 2
      ;;
    --comment-file)
      comment_file="$2"
      shift 2
      ;;
    --decision-request)
      decision_request="$2"
      shift 2
      ;;
    --dry-run)
      dry_run="1"
      shift
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

[ -n "$title" ] || { echo "missing --title" >&2; exit 2; }
if [ -z "$report" ] && [ -z "$report_file" ]; then
  echo "missing --report or --report-file" >&2
  exit 2
fi

cd "$manager_workdir"
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$manager_workdir" --expected-repo "$repo"
[ -x "$bridge_cmd" ] || { echo "missing bridge command: $bridge_cmd" >&2; exit 1; }

set -- "$bridge_cmd" open-stage \
  --repo "$repo" \
  --workdir "$manager_workdir" \
  --thread-id "$manager_thread_id" \
  --watcher-session-name claudecode_manager_audit \
  --state-file "$state_file" \
  --lock-file "$lock_file" \
  --title "$title" \
  --decision-request "$decision_request"
if [ -n "$report_file" ]; then
  set -- "$@" --report-file "$report_file"
else
  set -- "$@" --report "$report"
fi
if [ "$dry_run" = "1" ]; then
  set -- "$@" --dry-run
fi

"$@"

if [ "$dry_run" = "1" ]; then
  exit 0
fi

set -- "$bridge_cmd" close-active --force --state-file "$state_file" --lock-file "$lock_file"
if [ -n "$comment_file" ]; then
  set -- "$@" --comment-file "$comment_file"
elif [ -n "$comment" ]; then
  set -- "$@" --comment "$comment"
else
  set -- "$@" --comment "Codex manager audit recorded and closed."
fi

exec "$@"
