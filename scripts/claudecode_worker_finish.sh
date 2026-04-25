#!/usr/bin/env sh

set -eu

workdir=""
comment=""
comment_file=""
annotate_comment="0"
worker_id=""
report_file=""

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
sh scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"

if [ -n "$worker_id" ] && [ -z "$comment_file" ] && [ -z "$report_file" ]; then
  report_file=".codex-runtime/claudecode_workers/${worker_id}/report.md"
fi
if [ -z "$comment_file" ] && [ -n "$report_file" ]; then
  comment_file="$report_file"
fi

if [ -n "$worker_id" ]; then
  set -- go run ./cmd/babel-issue-bridge worker-finish --worker-id "$worker_id"
else
  set -- go run ./cmd/babel-issue-bridge manager-handoff
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

exec "$@"
