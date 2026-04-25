#!/usr/bin/env sh

set -eu

workdir=""
comment=""
comment_file=""
annotate_comment="0"
worker_id=""
report_file=""
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/scripts/claudecode_issue_bridge.sh}"

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
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
    --worker-id)
      worker_id="$2"
      shift 2
      ;;
    --report-file)
      report_file="$2"
      shift 2
      ;;
    --annotate-comment)
      annotate_comment="1"
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

if [ -n "$worker_id" ] && [ -z "$comment_file" ] && [ -z "$report_file" ]; then
  report_file=".codex-runtime/claudecode_workers/${worker_id}/report.md"
fi
if [ -z "$comment_file" ] && [ -n "$report_file" ]; then
  comment_file="$report_file"
fi

if [ -n "$worker_id" ]; then
  set -- "$bridge_cmd" worker-finish --worker-id "$worker_id"
else
  set -- "$bridge_cmd" manager-handoff
fi
if [ -n "$comment" ]; then
  set -- "$@" --comment "$comment"
fi
if [ -n "$comment_file" ]; then
  set -- "$@" --comment-file "$comment_file"
fi
if [ "$annotate_comment" = "1" ]; then
  set -- "$@" --annotate-comment
fi

if "$@"; then
  finish_status=0
else
  finish_status=$?
fi

abs_path() {
  case "$1" in
    /*) printf "%s\n" "$1" ;;
    *) printf "%s/%s\n" "$workdir" "$1" ;;
  esac
}

if [ "$finish_status" -eq 0 ] && [ -n "$worker_id" ] && [ "${CLAUDECODE_MANAGER_AUDIT_ISSUE:-1}" = "1" ]; then
  audit_report_file=""
  if [ -n "$comment_file" ]; then
    audit_report_file=$(abs_path "$comment_file")
  elif [ -n "$report_file" ]; then
    audit_report_file=$(abs_path "$report_file")
  fi
  if [ -n "$audit_report_file" ] && [ -f "$audit_report_file" ]; then
    BRIDGE_CMD="$bridge_cmd" sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_audit_issue.sh \
      --title "Codex manager audit: $worker_id" \
      --report-file "$audit_report_file" \
      --comment "Codex manager audit recorded worker handoff: $worker_id" || true
  else
    BRIDGE_CMD="$bridge_cmd" sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_audit_issue.sh \
      --title "Codex manager audit: $worker_id" \
      --report "Worker handoff recorded: $worker_id" \
      --comment "Codex manager audit recorded worker handoff: $worker_id" || true
  fi
fi

exit "$finish_status"
