#!/usr/bin/env sh

set -eu

# 这个脚本只负责打印当前 Termux 侧固定执行命令清单。
cat <<'EOF'
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_s_one_shot.sh | sh
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_m_one_shot.sh | sh
curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_probe_one_shot.sh | sh
sh /home/openclaw/babel-runtime/scripts/termux_patch_existing_s_thread.sh
sh /home/openclaw/babel-runtime/scripts/termux_reinstall_m_offline.sh
sh /home/openclaw/babel-runtime/scripts/termux_rewrite_s_one_shot.sh
sh /home/openclaw/babel-runtime/scripts/termux_rewrite_m_one_shot.sh
sh /home/openclaw/babel-runtime/scripts/termux_rewrite_probe_one_shot.sh
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh clean
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh run
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh latest
sh /home/openclaw/babel-runtime/scripts/termux_exit_probe.sh list
sh /home/openclaw/babel-runtime/scripts/termux_check_local_state.sh
probe clean
probe run
probe latest
probe list
EOF
