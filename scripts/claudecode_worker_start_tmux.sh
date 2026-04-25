#!/usr/bin/env sh

set -eu

workdir=""
session_name=""
worker_prefix=""
timeout_seconds="1800"
max_running="1"
allow_same_lane="0"
model=""
tmux_socket="claudecode_manager"
bridge_cmd="${BRIDGE_CMD:-/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge}"
session_id=""

while [ $# -gt 0 ]; do
  case "$1" in
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --session-name)
      session_name="$2"
      shift 2
      ;;
    --worker-prefix)
      worker_prefix="$2"
      shift 2
      ;;
    --timeout-seconds)
      timeout_seconds="$2"
      shift 2
      ;;
    --max-running)
      max_running="$2"
      shift 2
      ;;
    --allow-same-lane)
      allow_same_lane="1"
      shift
      ;;
    --model)
      model="$2"
      shift 2
      ;;
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --tmux-socket)
      tmux_socket="$2"
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
command -v tmux >/dev/null 2>&1 || {
  echo "missing tmux" >&2
  exit 1
}

quote() {
  printf "'"
  printf "%s" "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

if [ -z "$session_name" ]; then
  session_suffix=$(printf "%s" "${worker_prefix:-next}" | tr -c '[:alnum:]_-' '_' | sed 's/_*$//')
  if [ -z "$session_suffix" ]; then
    session_suffix="next"
  fi
  session_name="claudecode_worker_${session_suffix}"
fi

if [ -n "$tmux_socket" ]; then
  tmux_has_session() { tmux -L "$tmux_socket" has-session -t "$session_name"; }
  tmux_new_session() { tmux -L "$tmux_socket" new-session -d -s "$session_name" "$run_file"; }
else
  tmux_has_session() { tmux has-session -t "$session_name"; }
  tmux_new_session() { tmux new-session -d -s "$session_name" "$run_file"; }
fi

if tmux_has_session 2>/dev/null; then
  echo "tmux session already exists: $session_name" >&2
  exit 1
fi

mkdir -p .codex-runtime/tmux
run_file=$(mktemp ".codex-runtime/tmux/${session_name}.XXXXXX.sh")

{
  echo '#!/usr/bin/env sh'
  echo 'set -eu'
  printf 'cd %s\n' "$(quote "$workdir")"
  printf 'BRIDGE_CMD=%s exec sh /home/openclaw/claudecode-manager/scripts/claudecode_manager_next.sh --workdir %s --run-once --timeout-seconds %s --max-running %s' \
    "$(quote "$bridge_cmd")" \
    "$(quote "$workdir")" "$(quote "$timeout_seconds")" "$(quote "$max_running")"
  if [ -n "$worker_prefix" ]; then
    printf ' --worker-prefix %s' "$(quote "$worker_prefix")"
  fi
  if [ "$allow_same_lane" = "1" ]; then
    printf ' --allow-same-lane'
  fi
  if [ -n "$model" ]; then
    printf ' --model %s' "$(quote "$model")"
  fi
  if [ -n "$session_id" ]; then
    printf ' --session-id %s' "$(quote "$session_id")"
  fi
  printf '\n'
} > "$run_file"

chmod +x "$run_file"
tmux_new_session
echo "started tmux session: $session_name"
