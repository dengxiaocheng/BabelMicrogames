#!/data/data/com.termux/files/usr/bin/sh
rm -f "$PREFIX/bin/m"
cat << 'EOF' > "$PREFIX/bin/m"
#!/data/data/com.termux/files/usr/bin/sh
# 这个启动器负责把 Termux 的 m 入口重写成 Babel 固定线程的远端手动会话入口。
target_host=139.159.147.96
runtime_root=/home/openclaw/babel-runtime
remote_helper="$runtime_root/scripts/termux_manual_remote.sh"
session_id="$(date +%s).$$"
launcher_pid=$$
launcher_ppid="$(ps -o ppid= -p "$launcher_pid" | tr -d ' ' || true)"
launcher_tty="$(ps -o tty= -p "$launcher_pid" | tr -d ' ' || true)"
lease_interval=2
lease_ttl_seconds=8
lease_check_seconds=2
tmp_root="${TMPDIR:-$PREFIX/tmp}"
state_dir="${tmp_root%/}/codex-manual"
mkdir -p "$state_dir" || exit 1
fifo=$(mktemp -u "$state_dir/termux_m.XXXXXX.fifo") || exit 1
pidfile=$(mktemp "$state_dir/termux_m.XXXXXX.pid") || exit 1
reasonfile=$(mktemp "$state_dir/termux_m.XXXXXX.reason") || exit 1

# 这些本地临时文件只服务于当前一次 m 会话。
cleanup_local_state() {
  rm -f "$fifo" "$pidfile" "$reasonfile"
}

cleanup() {
  trap - EXIT HUP INT TERM
  if [ -n "${heartbeat_pid:-}" ]; then
    kill "$heartbeat_pid" 2>/dev/null || true
    wait "$heartbeat_pid" 2>/dev/null || true
  fi
  cleanup_local_state
}
trap cleanup EXIT HUP INT TERM

# watchdog 负责在本地前台会话消失后杀掉本地 ssh。
watchdog_cmd='
fifo="$1"
pidfile="$2"
reasonfile="$3"
(
  cat "$fifo" >/dev/null 2>&1 || true
  printf "%s\n" "fifo_closed" > "$reasonfile"
) &
fifo_watch_pid=$!
while :; do
  if [ -f "$reasonfile" ]; then
    break
  fi
  if ! stty -a <&4 >/dev/null 2>&1; then
    printf "%s\n" "tty_closed" > "$reasonfile"
    break
  fi
  sleep 1
done
kill "$fifo_watch_pid" 2>/dev/null || true
wait "$fifo_watch_pid" 2>/dev/null || true
pid=""
if [ -f "$pidfile" ]; then
  pid=$(cat "$pidfile" 2>/dev/null)
fi
rm -f "$fifo" "$pidfile" "$reasonfile"
if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
  kill -TERM "$pid" 2>/dev/null || true
  sleep 1
  kill -KILL "$pid" 2>/dev/null || true
fi
'

mkfifo "$fifo" || {
  cleanup_local_state
  exit 1
}

if command -v setsid >/dev/null 2>&1; then
  exec 4<&0
  setsid sh -c "$watchdog_cmd" sh "$fifo" "$pidfile" "$reasonfile" >/dev/null 2>&1 &
else
  exec 4<&0
  nohup sh -c "$watchdog_cmd" sh "$fifo" "$pidfile" "$reasonfile" >/dev/null 2>&1 &
fi
watchdog_pid=$!

# heartbeat 负责定期续租，让远端 manual-resume 能判断当前手动会话仍然活着。
heartbeat_loop() {
  while :; do
    if ! kill -0 "$launcher_pid" 2>/dev/null; then
      break
    fi
    current_launcher_ppid="$(ps -o ppid= -p "$launcher_pid" 2>/dev/null | tr -d ' ' || true)"
    if [ -z "$current_launcher_ppid" ] || [ "$current_launcher_ppid" = "1" ] || [ "$current_launcher_ppid" != "$launcher_ppid" ]; then
      break
    fi
    current_launcher_tty="$(ps -o tty= -p "$launcher_pid" 2>/dev/null | tr -d ' ' || true)"
    if [ -z "$current_launcher_tty" ] || [ "$current_launcher_tty" = "?" ] || [ "$current_launcher_tty" != "$launcher_tty" ]; then
      break
    fi
    if ! stty -a <&4 >/dev/null 2>&1; then
      break
    fi
    if [ -f "$pidfile" ]; then
      ssh_pid="$(cat "$pidfile" 2>/dev/null || true)"
      if [ -n "$ssh_pid" ]; then
        if ! kill -0 "$ssh_pid" 2>/dev/null; then
          break
        fi
        ssh_tty="$(ps -o tty= -p "$ssh_pid" 2>/dev/null | tr -d ' ' || true)"
        if [ -z "$ssh_tty" ] || [ "$ssh_tty" = "?" ]; then
          break
        fi
      fi
    fi
    ssh -o StrictHostKeyChecking=no root@"$target_host" \
      "su - openclaw -c 'sh $remote_helper --action touch-lease --workdir /home/openclaw/Babel --bridge-root /home/openclaw/babel-runtime --thread-id 019da0d8-5750-7543-9d76-2feb50289399 --entrypoint termux_m --session-id $session_id'" \
      >/dev/null 2>&1 || true
    sleep "$lease_interval" || true
  done
}
heartbeat_loop &
heartbeat_pid=$!

# 最后进入真正的前台 ssh，并把 pid 记录给 watchdog/heartbeat 使用。
exec 3>"$fifo"
sh -c '
pidfile="$1"
shift
printf "%s\n" "$$" > "$pidfile"
exec ssh -o StrictHostKeyChecking=no -tt root@"$1" "su - openclaw -c '\''sh $2 --action manual-resume --workdir /home/openclaw/Babel --bridge-root /home/openclaw/babel-runtime --thread-id 019da0d8-5750-7543-9d76-2feb50289399 --entrypoint termux_m --session-id $3 --lease-ttl-seconds $4 --lease-check-seconds $5'\''"
' sh "$pidfile" "$target_host" "$remote_helper" "$session_id" "$lease_ttl_seconds" "$lease_check_seconds" 3>&-
status=$?
exec 3>&-
exec 4<&-
kill "$heartbeat_pid" 2>/dev/null || true
wait "$heartbeat_pid" 2>/dev/null || true
wait "$watchdog_pid" 2>/dev/null || true
cleanup_local_state
exit "$status"
EOF
chmod +x "$PREFIX/bin/m"
hash -r
