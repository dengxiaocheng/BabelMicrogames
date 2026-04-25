#!/data/data/com.termux/files/usr/bin/sh
rm -f "$PREFIX/bin/probe"
cat << 'EOF' > "$PREFIX/bin/probe"
#!/data/data/com.termux/files/usr/bin/sh

set -eu

# 这个启动器把 Termux 本地的 probe 入口写成通知栏 Exit 探针。
if [ -n "${PREFIX:-}" ]; then
  prefix_root="$PREFIX"
elif [ -d /data/data/com.termux/files/usr ]; then
  prefix_root=/data/data/com.termux/files/usr
else
  prefix_root=/tmp/termux-probe
fi
tmp_root="${TMPDIR:-$prefix_root/tmp}"
probe_root="${tmp_root%/}/codex-manual"
script_path="${PREFIX:-/data/data/com.termux/files/usr}/bin/probe"
command="${1:-run}"

mkdir -p "$probe_root"

usage() {
  cat <<EOF_USAGE
usage:
  $script_path run
  $script_path latest
  $script_path list
  $script_path clean
  $script_path help
EOF_USAGE
}

# latest/list/clean 只处理当前 probe_root 下的探针日志和探针进程。
latest_log_path() {
  find "$probe_root" -maxdepth 1 -type f -name 'termux-exit-probe-*.log' | sort | tail -n1
}

show_latest() {
  latest="$(latest_log_path)"
  if [ -z "$latest" ]; then
    echo "no probe log found in $probe_root"
    return 1
  fi
  echo "$latest"
  cat "$latest"
}

list_logs() {
  find "$probe_root" -maxdepth 1 -type f -name 'termux-exit-probe-*.log' | sort
}

clean_probe_processes() {
  ps -eo pid,args | while read -r pid args; do
    case "$args" in
      *"$script_path"*)
        if [ "$pid" != "$$" ]; then
          kill "$pid" 2>/dev/null || true
        fi
        ;;
    esac
  done
}

clean_probe_state() {
  clean_probe_processes
  rm -f "${probe_root%/}/termux-exit-probe-"*.log 2>/dev/null || true
  echo "cleaned $probe_root"
}

case "$command" in
  help|-h|--help)
    usage
    exit 0
    ;;
  latest)
    show_latest
    exit 0
    ;;
  list)
    list_logs
    exit 0
    ;;
  clean)
    clean_probe_state
    exit 0
    ;;
  run)
    shift || true
    ;;
  *)
    echo "unknown command: $command" >&2
    usage >&2
    exit 2
    ;;
esac

stamp="$(date +%Y%m%d-%H%M%S)"
log_path="${probe_root%/}/termux-exit-probe-${stamp}.log"

# run 模式会持续写 tick，直到当前脚本真正退出。
log_line() {
  printf '%s %s\n' "$(date '+%Y-%m-%dT%H:%M:%S%z')" "$*" >> "$log_path"
}

current_ppid() {
  ps -o ppid= -p "$$" 2>/dev/null | tr -d ' ' || true
}

current_tty() {
  ps -o tty= -p "$$" 2>/dev/null | tr -d ' ' || true
}

current_pgid() {
  ps -o pgid= -p "$$" 2>/dev/null | tr -d ' ' || true
}

on_signal() {
  signal_name="$1"
  log_line "signal=${signal_name} pid=$$ ppid=$(current_ppid) pgid=$(current_pgid) tty=$(current_tty)"
}

trap 'on_signal EXIT' EXIT
trap 'on_signal HUP' HUP
trap 'on_signal INT' INT
trap 'on_signal TERM' TERM

printf 'log=%s\n' "$log_path"
printf 'pid=%s ppid=%s pgid=%s tty=%s\n' "$$" "$(current_ppid)" "$(current_pgid)" "$(current_tty)"
printf 'latest: %s latest\n' "$script_path"
printf 'list: %s list\n' "$script_path"
printf 'clean: %s clean\n' "$script_path"

log_line "start pid=$$ ppid=$(current_ppid) pgid=$(current_pgid) tty=$(current_tty) prefix=${prefix_root} tmp=${tmp_root}"

count=0
while :; do
  count=$((count + 1))
  log_line "tick=${count} pid=$$ ppid=$(current_ppid) pgid=$(current_pgid) tty=$(current_tty)"
  sleep 1
done
EOF
chmod +x "$PREFIX/bin/probe"
hash -r
