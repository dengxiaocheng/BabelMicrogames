# 需求变更记录

本文件只记录 `BabelMicrogames` manager 仓的变更。`s`、`m`、Babel runtime、Termux、collab、微信服务等历史记录不在本仓维护。

## 2026-04-25

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  manager 仓完成瘦身：删除复制自 runtime 的 `cmd/`、`internal/`、`go.mod`、`requirements/`、`.github/`、`.githooks/`、Termux/Windows 节点脚本和旧 runtime 文档。本仓只保留 ClaudeCode manager 业务脚本、微游戏工厂文档、计划文件和 manager 级状态说明。
- 入口：
  - `AGENTS.md`
  - `README.md`
  - `docs/INDEX.md`
  - `docs/OPERATIONS.md`
  - `docs/operations/README.md`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  `claudecode_manager_refresh_state.sh` 和 `claudecode_manager_status.sh` 现在把 manager 主 stage issue 与 manager audit issue 一起写入 `.codex-runtime/microgame_manager_state.json`，便于 `s` 只读一个总表。
- 入口：
  - `scripts/claudecode_manager_refresh_state.sh`
  - `scripts/claudecode_manager_status.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  manager audit issue 使用独立 `.codex-runtime/manager_audit_issue_state.json` 和 `.codex-runtime/manager_audit_issue.lock`，不再覆盖 manager 主 `.codex-runtime/issue_bridge_state.json`。
- 入口：
  - `scripts/claudecode_manager_audit_issue.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  ClaudeCode manager 的 issue bridge 默认复用 `s` 的 Go bridge。`scripts/claudecode_issue_bridge.sh` 是薄包装，转发到 `/home/openclaw/babel-runtime/scripts/stage_issue_bridge.sh`。
- 入口：
  - `scripts/claudecode_issue_bridge.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  `BabelMicrogames` 定位为 manager 资料仓，不保存具体小游戏源码。每个小游戏必须使用独立 `dengxiaocheng/BabelMicrogame-*` 仓库和独立 `/home/openclaw/babel-microgames/<game>` workdir。
- 入口：
  - `docs/operations/MICROGAME_FACTORY_FLOW.md`
  - `docs/operations/CLAUDECODE_MANAGER.md`
