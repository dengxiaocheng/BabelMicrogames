#!/usr/bin/env sh

set -eu

workdir=""
worker_id=""
model=""
allowed_tools=""
timeout_seconds="1800"
auto_finish="1"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"
session_id=""

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --worker-id)
      worker_id="$2"
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
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --timeout-seconds)
      timeout_seconds="$2"
      shift 2
      ;;
    --no-auto-finish)
      auto_finish="0"
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
if [ -z "$worker_id" ]; then
  echo "missing required arg: --worker-id" >&2
  exit 2
fi

cd "$workdir"
sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_repo_guard.sh --workdir "$workdir"
[ -x "$bridge_cmd" ] || {
  echo "missing bridge command: $bridge_cmd" >&2
  exit 1
}

if [ -z "$session_id" ]; then
  session_id=$(sh /home/openclaw/claudecode-manager/scripts/claudecode_game_session.sh --workdir "$workdir")
fi

packet_file=".codex-runtime/claudecode_workers/${worker_id}/packet.md"
report_file=".codex-runtime/claudecode_workers/${worker_id}/report.md"
output_file=".codex-runtime/claudecode_workers/${worker_id}/claude-output.log"

if [ ! -f "$packet_file" ]; then
  echo "missing packet: $packet_file" >&2
  exit 1
fi

"$bridge_cmd" worker-start --worker-id "$worker_id" >/dev/null

prompt=$(mktemp "${TMPDIR:-/tmp}/claudecode-worker.XXXXXX")
trap 'rm -f "$prompt"' EXIT HUP INT TERM

cat "$packet_file" > "$prompt"
cat >> "$prompt" <<'EOF'

## Run Mode

You are running unattended in ClaudeCode worker mode.

- Complete only this packet.
- Modify files only inside the declared write scope.
- Keep within the declared budget.
- Fill the report file listed in the packet.
- Do not wait for user confirmation.
- When finished, run the finish command listed in the packet.
EOF

set -- claude --dangerously-skip-permissions --session-id "$session_id" -p "$(cat "$prompt")"
if [ -n "$model" ]; then
  set -- "$@" --model "$model"
fi
if [ -n "$allowed_tools" ]; then
  set -- "$@" --allowedTools "$allowed_tools"
fi
if command -v timeout >/dev/null 2>&1; then
  set -- timeout "$timeout_seconds" "$@"
fi

if "$@" >"$output_file" 2>&1; then
  claude_status=0
else
  claude_status=$?
fi

cat "$output_file"

if [ "$claude_status" -ne 0 ]; then
  "$bridge_cmd" worker-set-status --worker-id "$worker_id" --status rework --note "claudecode run-once failed: $claude_status" >/dev/null
  exit "$claude_status"
fi

worker_status=$(jq -r --arg id "$worker_id" '.workers[] | select(.worker_id==$id) | .status' .codex-runtime/claudecode_workers.json 2>/dev/null || true)
if [ "$auto_finish" = "1" ] && [ "$worker_status" != "handoff_queued" ] && [ "$worker_status" != "done" ]; then
  if [ ! -f "$report_file" ] || grep -q '待填写' "$report_file"; then
    {
      echo "# Worker Report: $worker_id"
      echo
      echo "## Summary"
      sed -n '1,120p' "$output_file"
      echo
      echo "## Budget Check"
      echo "- Actual Changed Files: $(git diff --name-only | wc -l | tr -d ' ')"
      echo "- Stayed Within Budget: 待 manager 审查"
      echo
      echo "## Files Changed"
      git diff --name-only | sed 's/^/- /'
      echo
      echo "## Tests"
      echo "- ClaudeCode run-once exited successfully."
      echo
      echo "## Scope Deviations"
      echo "- 待 manager 审查"
      echo
      echo "## Risks / Follow-ups"
      echo "- 待 manager 审查"
    } > "$report_file"
  fi
  BRIDGE_CMD="$bridge_cmd" sh /home/openclaw/claudecode-manager/scripts/claudecode_worker_finish.sh --workdir "$workdir" --worker-id "$worker_id"
fi
