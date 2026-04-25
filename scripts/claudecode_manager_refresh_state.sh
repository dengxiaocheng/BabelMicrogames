#!/usr/bin/env sh

set -eu

manager_workdir="/home/openclaw/claudecode-manager"
game_root="/home/openclaw/babel-microgames"
state_file=""
tmux_socket="claudecode_manager"
quiet="0"

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
    --state-file)
      state_file="$2"
      shift 2
      ;;
    --tmux-socket)
      tmux_socket="$2"
      shift 2
      ;;
    --quiet)
      quiet="1"
      shift
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

[ -d "$manager_workdir" ] || {
  echo "missing manager workdir: $manager_workdir" >&2
  exit 1
}
command -v jq >/dev/null 2>&1 || {
  echo "missing jq" >&2
  exit 1
}

if [ -z "$state_file" ]; then
  state_file="$manager_workdir/.codex-runtime/microgame_manager_state.json"
fi

mkdir -p "$(dirname "$state_file")"
games_file=$(mktemp "${TMPDIR:-/tmp}/microgame-manager-games.XXXXXX")
game_file=$(mktemp "${TMPDIR:-/tmp}/microgame-manager-game.XXXXXX")
sessions_file=$(mktemp "${TMPDIR:-/tmp}/microgame-manager-sessions.XXXXXX")
tmp_state=$(mktemp "${TMPDIR:-/tmp}/microgame-manager-state.XXXXXX")
trap 'rm -f "$games_file" "$game_file" "$sessions_file" "$tmp_state"' EXIT HUP INT TERM

printf '[]\n' > "$games_file"

repo_name_for() {
  git -C "$1" remote get-url origin 2>/dev/null |
    sed -e 's#^git@github.com:##' \
        -e 's#^https://github.com/##' \
        -e 's#^http://github.com/##' \
        -e 's#\.git$##'
}

pages_url_for() {
  repo="$1"
  case "$repo" in
    dengxiaocheng/*)
      name=${repo#dengxiaocheng/}
      printf 'https://dengxiaocheng.github.io/%s/' "$name"
      ;;
    *)
      printf ''
      ;;
  esac
}

if [ -d "$game_root" ]; then
  for game in "$game_root"/*; do
    [ -d "$game/.git" ] || continue

    slug=$(basename "$game")
    repo=$(repo_name_for "$game" || true)
    branch=$(git -C "$game" branch --show-current 2>/dev/null || true)
    head_commit=$(git -C "$game" rev-parse --short HEAD 2>/dev/null || true)
    dirty_count=$(git -C "$game" status --short 2>/dev/null | wc -l | tr -d ' ')
    if [ "$dirty_count" = "0" ]; then
      git_state="clean"
    else
      git_state="dirty"
    fi
    pages_url=$(pages_url_for "$repo")

    claude_session_id=""
    if [ -f "$game/.codex-runtime/claude_session_id" ]; then
      claude_session_id=$(tr -d '[:space:]' < "$game/.codex-runtime/claude_session_id")
    fi

    registry="$game/.codex-runtime/claudecode_workers.json"
    has_registry="false"
    worker_counts='{}'
    workers_total="0"
    current_stage=""
    current_stage_status=""
    blocked_reason=""
    next_action="initialize_worker_registry"

    if [ -f "$registry" ]; then
      has_registry="true"
      worker_counts=$(jq -c '[.workers[]?.status] | group_by(.) | map({(.[0]): length}) | add // {}' "$registry")
      workers_total=$(jq -r '[.workers[]?] | length' "$registry")
      current_stage=$(jq -r '
        def rank:
          if .status == "handoff_queued" then 0
          elif .status == "rework" then 1
          elif .status == "blocked" then 2
          elif .status == "running" then 3
          elif .status == "queued" then 4
          else 9
          end;
        [.workers[]? | select(.status == "handoff_queued" or .status == "rework" or .status == "blocked" or .status == "running" or .status == "queued")]
        | sort_by(rank)
        | .[0].worker_id // ""
      ' "$registry")
      current_stage_status=$(jq -r --arg id "$current_stage" '.workers[]? | select(.worker_id == $id) | .status // ""' "$registry")
      blocked_reason=$(jq -r '
        [.workers[]? | select(.status == "blocked" or .status == "rework" or .status == "failed") | .last_note // empty]
        | .[0] // ""
      ' "$registry")

      handoff_count=$(jq -r '[.workers[]? | select(.status == "handoff_queued")] | length' "$registry")
      blocked_count=$(jq -r '[.workers[]? | select(.status == "blocked")] | length' "$registry")
      rework_count=$(jq -r '[.workers[]? | select(.status == "rework")] | length' "$registry")
      running_count=$(jq -r '[.workers[]? | select(.status == "running")] | length' "$registry")
      queued_count=$(jq -r '[.workers[]? | select(.status == "queued")] | length' "$registry")
      if [ "$dirty_count" != "0" ]; then
        next_action="clean_worktree_before_dispatch"
      elif [ "$handoff_count" != "0" ]; then
        next_action="review_worker_handoff"
      elif [ "$blocked_count" != "0" ]; then
        next_action="resolve_blocked_worker"
      elif [ "$rework_count" != "0" ]; then
        next_action="dispatch_rework_worker"
      elif [ "$running_count" != "0" ]; then
        next_action="wait_running_worker"
      elif [ "$queued_count" != "0" ]; then
        next_action="dispatch_next_worker"
      else
        next_action="idle_or_seed_next_game"
      fi
    fi

    jq -n \
      --arg slug "$slug" \
      --arg workdir "$game" \
      --arg repo "$repo" \
      --arg branch "$branch" \
      --arg git_state "$git_state" \
      --argjson dirty_count "$dirty_count" \
      --arg head_commit "$head_commit" \
      --arg pages_url "$pages_url" \
      --arg claude_session_id "$claude_session_id" \
      --argjson has_registry "$has_registry" \
      --argjson worker_counts "$worker_counts" \
      --argjson workers_total "$workers_total" \
      --arg current_stage "$current_stage" \
      --arg current_stage_status "$current_stage_status" \
      --arg blocked_reason "$blocked_reason" \
      --arg next_action "$next_action" \
      '{
        slug: $slug,
        workdir: $workdir,
        repo: $repo,
        branch: $branch,
        git: {
          state: $git_state,
          dirty_count: $dirty_count,
          head_commit: $head_commit
        },
        pages_url: $pages_url,
        claude_session_id: $claude_session_id,
        has_worker_registry: $has_registry,
        worker_counts: $worker_counts,
        workers_total: $workers_total,
        current_stage: $current_stage,
        current_stage_status: $current_stage_status,
        blocked_reason: $blocked_reason,
        next_recommended_action: $next_action
      }' > "$game_file"
    jq --slurpfile game "$game_file" '. + [$game[0]]' "$games_file" > "$games_file.next"
    mv "$games_file.next" "$games_file"
  done
fi

if tmux -L "$tmux_socket" list-sessions -F '#S' >/dev/null 2>&1; then
  tmux -L "$tmux_socket" list-sessions -F '#S' |
    grep '^claudecode_worker_' |
    jq -R . |
    jq -s . > "$sessions_file"
else
  printf '[]\n' > "$sessions_file"
fi

generated_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)

jq -n \
  --arg generated_at_utc "$generated_at_utc" \
  --arg manager_workdir "$manager_workdir" \
  --arg game_root "$game_root" \
  --arg tmux_socket "$tmux_socket" \
  --slurpfile games "$games_file" \
  --slurpfile worker_sessions "$sessions_file" \
  '{
    schema_version: 1,
    generated_at_utc: $generated_at_utc,
    manager_workdir: $manager_workdir,
    game_root: $game_root,
    tmux_socket: $tmux_socket,
    summary: {
      games_total: ($games[0] | length),
      games_dirty: ($games[0] | map(select(.git.state == "dirty")) | length),
      games_dispatchable: ($games[0] | map(select(.next_recommended_action == "dispatch_next_worker" or .next_recommended_action == "dispatch_rework_worker")) | length),
      games_waiting_review: ($games[0] | map(select(.next_recommended_action == "review_worker_handoff")) | length),
      workers_total: ($games[0] | map(.workers_total) | add // 0),
      workers_queued: ($games[0] | map(.worker_counts.queued // 0) | add // 0),
      workers_running: ($games[0] | map(.worker_counts.running // 0) | add // 0),
      workers_rework: ($games[0] | map(.worker_counts.rework // 0) | add // 0),
      workers_blocked: ($games[0] | map(.worker_counts.blocked // 0) | add // 0),
      workers_handoff_queued: ($games[0] | map(.worker_counts.handoff_queued // 0) | add // 0),
      workers_done: ($games[0] | map(.worker_counts.done // 0) | add // 0),
      tmux_worker_sessions: ($worker_sessions[0] | length)
    },
    tmux: {
      worker_sessions: $worker_sessions[0]
    },
    games: $games[0]
  }' > "$tmp_state"

mv "$tmp_state" "$state_file"

if [ "$quiet" != "1" ]; then
  jq -r '
    "manager_state: " + .generated_at_utc,
    "summary: games=\(.summary.games_total) dispatchable=\(.summary.games_dispatchable) review=\(.summary.games_waiting_review) queued=\(.summary.workers_queued) running=\(.summary.workers_running) blocked=\(.summary.workers_blocked) done=\(.summary.workers_done) tmux_workers=\(.summary.tmux_worker_sessions)",
    (.games[] | "game: \(.slug) repo=\(.repo) git=\(.git.state) stage=\(.current_stage // ""):\(.current_stage_status // "") action=\(.next_recommended_action)")
  ' "$state_file"
fi
