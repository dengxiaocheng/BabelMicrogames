#!/usr/bin/env sh

set -eu

manager_workdir="/home/openclaw/claudecode-manager"
game_root="/home/openclaw/babel-microgames"
tmux_socket="claudecode_manager"

while [ $# -gt 0 ]; do
  case "$1" in
    --manager-workdir)
      manager_workdir="$2"
      shift 2
      ;;
    --game-root)
      game_root="$2"
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

state_file="$manager_workdir/.codex-runtime/microgame_manager_state.json"

sh "$manager_workdir/scripts/claudecode_manager_refresh_state.sh" \
  --manager-workdir "$manager_workdir" \
  --game-root "$game_root" \
  --tmux-socket "$tmux_socket" \
  --quiet

jq -r '
  "generated: \(.generated_at_utc)",
  "summary: games=\(.summary.games_total) dirty=\(.summary.games_dirty) dispatchable=\(.summary.games_dispatchable) review=\(.summary.games_waiting_review) queued=\(.summary.workers_queued) running=\(.summary.workers_running) rework=\(.summary.workers_rework) done=\(.summary.workers_done) tmux_workers=\(.summary.tmux_worker_sessions)",
  "tmux_workers: " + ((.tmux.worker_sessions // []) | join(", ")),
  (.games[] | "\(.slug): \(.git.state) \(.repo) stage=\(.current_stage // "")/\(.current_stage_status // "") action=\(.next_recommended_action) session=\(.claude_session_id)")
' "$state_file"
