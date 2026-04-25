#!/usr/bin/env sh

set -eu

if [ $# -ne 2 ]; then
  echo "usage: $0 s|m <new-thread-id>" >&2
  exit 2
fi

lane="$1"
new_thread_id="$2"

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
runtime_state_dir="$repo_root/.codex-runtime"
babel_repo_root="/home/openclaw/Babel"
babel_state_dir="$babel_repo_root/.codex-runtime"

read_thread_id() {
  sed -n 's/.*"thread_id": "\(.*\)".*/\1/p' "$1" | head -n 1
}

replace_exact() {
  old_value="$1"
  new_value="$2"
  file_path="$3"
  if [ -f "$file_path" ]; then
    perl -0pi -e "s/\\Q$old_value\\E/$new_value/g" "$file_path"
  fi
}

replace_var_line() {
  file_path="$1"
  var_name="$2"
  new_value="$3"
  perl -0pi -e "s/^$var_name=.*$/$var_name=\\\"$new_value\\\"/m" "$file_path"
}

case "$lane" in
  s)
    old_thread_id=$(read_thread_id "$runtime_state_dir/issue_bridge_state.json")
    [ -n "$old_thread_id" ] || { echo "failed to read current s thread id" >&2; exit 1; }
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/scripts/termux_rewrite_s_one_shot.sh"
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/.codex-runtime/issue_bridge_state.json"
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/.codex-runtime/thread_control.json"
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/scripts/termux_patch_existing_s_thread.sh"
    replace_var_line "$repo_root/scripts/termux_manual_remote.sh" "legacy_termux_s_thread_id" "$old_thread_id"
    replace_var_line "$repo_root/scripts/termux_manual_remote.sh" "current_termux_s_thread_id" "$new_thread_id"
    ;;
  m)
    old_thread_id=$(read_thread_id "$babel_state_dir/issue_bridge_state.json")
    [ -n "$old_thread_id" ] || { echo "failed to read current m thread id" >&2; exit 1; }
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/scripts/termux_rewrite_m_one_shot.sh"
    replace_exact "$old_thread_id" "$new_thread_id" "$repo_root/scripts/termux_reinstall_m_offline.sh"
    replace_exact "$old_thread_id" "$new_thread_id" "$babel_state_dir/issue_bridge_state.json"
    replace_exact "$old_thread_id" "$new_thread_id" "$babel_state_dir/thread_control.json"
    ;;
  *)
    echo "unknown lane: $lane" >&2
    exit 2
    ;;
esac

echo "switched $lane thread id: $old_thread_id -> $new_thread_id"
