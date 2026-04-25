#!/data/data/com.termux/files/usr/bin/sh

set -eu

# 这个脚本用于在 Termux 本地快速查看 ssh 残留、临时文件和 s/m 启动器状态。
prefix_root="${PREFIX:-/data/data/com.termux/files/usr}"
tmp_root="${TMPDIR:-$prefix_root/tmp}"
manual_root="${tmp_root%/}/codex-manual"
target_host="${1:-139.159.147.96}"

print_section() {
  printf '\n=== %s ===\n' "$1"
}

print_section "ssh"
if command -v pgrep >/dev/null 2>&1; then
  pgrep -af "ssh .*${target_host}" || true
else
  ps -ef | grep "[s]sh .*${target_host}" || true
fi

print_section "codex-manual temp"
if [ -d "$manual_root" ]; then
  find "$manual_root" -maxdepth 1 -type f | sort || true
else
  printf '(missing) %s\n' "$manual_root"
fi

print_section "prefix bin"
for name in s m; do
  path="${prefix_root%/}/bin/$name"
  if [ -e "$path" ]; then
    ls -l "$path"
  else
    printf '(missing) %s\n' "$path"
  fi
done

print_section "env"
printf 'PREFIX=%s\n' "$prefix_root"
printf 'TMPDIR=%s\n' "$tmp_root"
printf 'HOST=%s\n' "$target_host"
