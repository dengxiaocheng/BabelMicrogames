#!/usr/bin/env sh

set -eu

manager_workdir="/home/openclaw/claudecode-manager"
dry_run="0"

while [ $# -gt 0 ]; do
  case "$1" in
    --manager-workdir)
      manager_workdir="$2"
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

[ -d "$manager_workdir" ] || {
  echo "missing manager workdir: $manager_workdir" >&2
  exit 1
}
command -v jq >/dev/null 2>&1 || {
  echo "missing jq" >&2
  exit 1
}

runtime="$manager_workdir/.codex-runtime"
registry="$runtime/claudecode_workers.json"
archive_dir="$runtime/archive/legacy-manager-state"
stamp=$(date -u +%Y%m%dT%H%M%SZ)

mkdir -p "$runtime"

legacy_registry="0"
total_workers="0"
bad_workers="0"
if [ -f "$registry" ]; then
  total_workers=$(jq -r '[.workers[]?] | length' "$registry")
  bad_workers=$(jq -r --arg manager_workdir "$manager_workdir" '
    [.workers[]? | select(((.repo_full_name // "") | startswith("dengxiaocheng/BabelMicrogame-") | not) or ((.workdir // "") == $manager_workdir))]
    | length
  ' "$registry")
  if [ "$total_workers" != "0" ]; then
    legacy_registry="1"
  fi
fi

legacy_dirs=""
for path in "$runtime/claudecode_workers" "$runtime/tmux"; do
  if [ -e "$path" ]; then
    legacy_dirs="$legacy_dirs $path"
  fi
done

if [ "$legacy_registry" = "0" ] && [ -z "$legacy_dirs" ]; then
  echo "legacy manager state clean: $manager_workdir"
  exit 0
fi

echo "legacy manager registry workers: total=$total_workers bad=$bad_workers"
if [ -n "$legacy_dirs" ]; then
  echo "legacy manager runtime dirs:$legacy_dirs"
fi

if [ "$dry_run" = "1" ]; then
  echo "dry-run: no files changed"
  exit 0
fi

mkdir -p "$archive_dir"

if [ -f "$registry" ] && [ "$legacy_registry" = "1" ]; then
  cp "$registry" "$archive_dir/claudecode_workers.$stamp.json"
  cat > "$registry" <<'EOF'
{
  "schema_version": 1,
  "workers": []
}
EOF
  echo "archived manager-level worker registry"
fi

for path in $legacy_dirs; do
  [ -e "$path" ] || continue
  base=$(basename "$path")
  target="$archive_dir/${base}.${stamp}"
  mv "$path" "$target"
  echo "archived $path -> $target"
done
