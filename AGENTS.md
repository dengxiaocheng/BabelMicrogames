# AGENTS.md

本仓库是 Babel 微游戏工厂的独立 Codex manager 工作目录规则。

## 身份

- 本地目录：`/home/openclaw/claudecode-manager`
- GitHub 仓库：`dengxiaocheng/BabelMicrogames`
- 角色：管理 ClaudeCode worker，拆包、派发、审查、记录和推进小游戏阶段
- 非角色：不承载 Babel 主体、不承载 `s/m` 长期会话、不承载任何具体小游戏源码

## 统一控制面

`s` 是服务器 control plane：

- 目录：`/home/openclaw/babel-runtime`
- 仓库：`dengxiaocheng/BabelOnline-GoCpp`
- 真源：Go issue bridge、watcher/hook、Termux、节点进程、GitHub handoff、总控状态入口

本仓库不得复制或扩张第二套 Go bridge，也不得保存 AI 会话管理脚本。所有 `worker-*`、`open-stage`、`manager-handoff`、`watch`、事件日志和 hook 默认必须通过 `s`：

```bash
/home/openclaw/babel-runtime/scripts/claudecode_issue_bridge.sh
```

该入口只是薄包装，最终调用：

```bash
/home/openclaw/babel-runtime/scripts/stage_issue_bridge.sh
```

## 目录边界

本仓库只保留：

- `docs/operations/*.md`
- `docs/REQUIREMENT_CHANGELOG.md`
- `plan/*.md`

不应再出现：

- `cmd/`
- `internal/`
- `go.mod`
- `requirements/`
- `.github/workflows/`
- `.githooks/`
- `scripts/`
- Termux / Windows 节点脚本

这些属于 `s` 或具体游戏仓库，不属于 manager 仓。AI 会话管理脚本统一放在 `/home/openclaw/babel-runtime/scripts` 或 `babel-ops`。

## 小游戏边界

每个小游戏必须独立：

- 源码目录：`/home/openclaw/babel-microgames/<game>`
- GitHub 仓库：`dengxiaocheng/BabelMicrogame-*`
- worker registry：`/home/openclaw/babel-microgames/<game>/.codex-runtime/claudecode_workers.json`
- Claude session：写在该游戏自己的 `.codex-runtime/claude_session_id`

本仓库的 `.codex-runtime/microgame_manager_state.json` 只是聚合总表，不是 worker 真源。

## Manager 职责

允许：

- 读取 `s` 的 incoming 计划
- 创建或继续小游戏 workdir
- 生成 worker packet
- 启动 ClaudeCode worker
- 审查 worker report 和 diff
- 跑目标游戏自己的测试
- 提交并推送目标游戏仓库
- 打开下一阶段 issue，且 `--resume-workdir` 固定为 `/home/openclaw/claudecode-manager`
- worker finish 后创建并关闭 manager audit issue
- 所有 stage/audit issue 必须通过 `s` 的统一 Go bridge 自动带 commit trace：issue 正文和关闭评论都应指向当前 HEAD 的 GitHub commit 链接

禁止：

- 把小游戏源码写进 `BabelMicrogames`
- 把 worker issue 写进 `BabelOnline-GoCpp` 或 `Babel`
- 用 manager 仓自己的 `.codex-runtime/claudecode_workers.json` 做调度真源
- 让 watcher 把 Codex manager 恢复进小游戏源码目录
- 在本仓重新实现 `s` 已经拥有的 Go bridge、Termux 或节点进程控制

## 状态和进程

常驻进程只允许由 `s` 启动和管理的轻量 watcher / worker：

- `claudecode_manager_autorun`
- 每个活动游戏一个 `claudecode_manager_watch_<game>`

不保留无 `s` 脚本来源的临时 web server、旧 `go run ./cmd/babel-issue-bridge` watcher、旧本地 bridge binary。

查看状态：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_manager_status.sh
```

从 `s` 查看全局状态：

```bash
cd /home/openclaw/babel-runtime
sh /home/openclaw/babel-runtime/scripts/s_control_status.sh
```

## 验证

提交前至少运行：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_manager_status.sh
git diff --check
```
