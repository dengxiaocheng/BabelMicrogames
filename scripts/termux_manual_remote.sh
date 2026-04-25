#!/usr/bin/env sh

set -eu

# 这个脚本是 s/m 在远端执行的统一 helper，只负责 manual-resume 和 lease 续租。
action=""
workdir=""
bridge_root=""
thread_id=""
entrypoint=""
session_id=""
lease_ttl_seconds=""
lease_check_seconds=""

legacy_termux_s_thread_id="019db3d5-3e00-7fd0-b723-0b48a2977b35"
current_termux_s_thread_id="019dbcf2-d632-7770-9cbc-ae1f963eb2e0"

while [ $# -gt 0 ]; do
  case "$1" in
    --action)
      action="$2"
      shift 2
      ;;
    --workdir)
      workdir="$2"
      shift 2
      ;;
    --bridge-root)
      bridge_root="$2"
      shift 2
      ;;
    --thread-id)
      thread_id="$2"
      shift 2
      ;;
    --entrypoint)
      entrypoint="$2"
      shift 2
      ;;
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --lease-ttl-seconds)
      lease_ttl_seconds="$2"
      shift 2
      ;;
    --lease-check-seconds)
      lease_check_seconds="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

if [ -z "$action" ] || [ -z "$workdir" ] || [ -z "$bridge_root" ] || [ -z "$thread_id" ] || [ -z "$entrypoint" ]; then
  echo "missing required args" >&2
  exit 2
fi

if [ "$entrypoint" = "termux_s" ] && [ "$thread_id" = "$legacy_termux_s_thread_id" ]; then
  thread_id="$current_termux_s_thread_id"
fi

export PATH=/home/openclaw/.nvm/versions/node/v22.22.1/bin:$PATH
export TERM=xterm-256color

PROXY_ENV=/home/openclaw/.config/babel-runtime/codex-proxy.env
if [ -f "$PROXY_ENV" ]; then
  . "$PROXY_ENV"
fi

cd "$workdir" || exit 1
BRIDGE_BIN="$bridge_root/.codex-runtime/bin/babel-issue-bridge"

# 如果运维二进制缺失或过期，就在远端现场重编译。
bridge_needs_rebuild() {
  if [ ! -x "$BRIDGE_BIN" ]; then
    return 0
  fi
  if find \
    "$bridge_root/cmd/babel-issue-bridge" \
    "$bridge_root/internal/ops/issuebridge" \
    -type f -newer "$BRIDGE_BIN" 2>/dev/null | grep -q .; then
    return 0
  fi
  return 1
}

if bridge_needs_rebuild; then
  mkdir -p "$bridge_root/.codex-runtime/bin" || exit 1
  (cd "$bridge_root" && go build -o "$BRIDGE_BIN" ./cmd/babel-issue-bridge) || exit 1
fi

case "$action" in
  manual-resume)
    if [ -z "$session_id" ]; then
      echo "missing --session-id for manual-resume" >&2
      exit 2
    fi
    # manual-resume 进入工作态后，会把整条手动链放进同一清理语义里。
    exec sh -lc "
pgid=\$(ps -o pgid= \$\$ | tr -d ' ')
cleanup() {
  trap - EXIT HUP INT TERM
  kill -TERM -- -\"\$pgid\" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM
\"$BRIDGE_BIN\" manual-resume --thread-id \"$thread_id\" --entrypoint \"$entrypoint\" --lease-session-id \"$session_id\" ${lease_ttl_seconds:+--lease-ttl-seconds \"$lease_ttl_seconds\"} ${lease_check_seconds:+--lease-check-seconds \"$lease_check_seconds\"}
"
    ;;
  touch-lease)
    if [ -z "$session_id" ]; then
      echo "missing --session-id for touch-lease" >&2
      exit 2
    fi
    # touch-lease 只刷新当前手动会话的 lease，不接管线程。
    exec "$BRIDGE_BIN" touch-manual-lease --thread-id "$thread_id" --entrypoint "$entrypoint" --session-id "$session_id"
    ;;
  *)
    echo "unknown action: $action" >&2
    exit 2
    ;;
esac
