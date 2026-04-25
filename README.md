# Babel Microgames Manager

这是 Babel 微游戏工厂的独立 Codex manager 仓库。

- 本地目录：`/home/openclaw/claudecode-manager`
- 远端仓库：`dengxiaocheng/BabelMicrogames`
- 控制面真源：`/home/openclaw/babel-runtime`
- 小游戏源码根目录：`/home/openclaw/babel-microgames`

本仓库只保存 manager 规则、脚本、流程文档和 manager 级协调记录；具体小游戏源码必须进入独立的 `dengxiaocheng/BabelMicrogame-*` 仓库。

## 边界

本仓库不是：

- Babel 主项目
- `s` 的 Go runtime
- `m` 的 C++/Godot 工作目录
- 某个小游戏源码仓
- 第二套 issue bridge 实现

`s` 统一拥有 Go issue bridge、watcher/hook、Termux、节点进程和 GitHub handoff。manager 只通过薄包装调用 `s`：

```bash
scripts/claudecode_issue_bridge.sh
```

## 主要文档

- [docs/INDEX.md](docs/INDEX.md)
- [docs/operations/MICROGAME_FACTORY_FLOW.md](docs/operations/MICROGAME_FACTORY_FLOW.md)
- [docs/operations/CLAUDECODE_MANAGER.md](docs/operations/CLAUDECODE_MANAGER.md)
- [docs/operations/CODEX_MANAGER_INTELLIGENCE.md](docs/operations/CODEX_MANAGER_INTELLIGENCE.md)
- [docs/REQUIREMENT_CHANGELOG.md](docs/REQUIREMENT_CHANGELOG.md)

`s` 的节点、Termux、issue bridge、collab、Windows 协作等文档不在本仓维护，去 `/home/openclaw/babel-runtime/docs/operations/` 看。

## 常用入口

查看 manager 总表：

```bash
sh scripts/claudecode_manager_status.sh
```

启动自动调度：

```bash
sh scripts/claudecode_manager_autorun.sh \
  --workdir /home/openclaw/claudecode-manager \
  --manager-workdir /home/openclaw/claudecode-manager \
  --game-root /home/openclaw/babel-microgames \
  --tmux-socket claudecode_manager \
  --poll-seconds 30 \
  --quiet-start 1400 \
  --quiet-end 1800 \
  --max-running 1 \
  --timeout-seconds 1800 \
  --daemon
```

启动某个游戏 watcher：

```bash
sh scripts/claudecode_manager_start_watcher.sh \
  --workdir /home/openclaw/babel-microgames/gongtou-dianming \
  --session-name claudecode_manager_watch_gongtou_dianming \
  --resume-prefix claudecode_manager_resume_gongtou_dianming \
  --poll-seconds 30 \
  --tmux-socket claudecode_manager
```

打开游戏阶段 issue：

```bash
sh scripts/claudecode_manager_open_game_stage.sh \
  --game-workdir /home/openclaw/babel-microgames/<game> \
  --repo dengxiaocheng/BabelMicrogame-<Name> \
  --title "<stage>" \
  --report "..." \
  --decision-request "..."
```

worker 完成回交：

```bash
sh scripts/claudecode_worker_finish.sh --workdir /home/openclaw/babel-microgames/<game> --worker-id <worker-id>
```

该入口默认会创建并关闭一条 manager audit issue，状态写入：

```text
.codex-runtime/manager_audit_issue_state.json
```

## 状态文件

- `.codex-runtime/microgame_manager_state.json`
  manager 聚合总表，供 `s` 读取。
- `.codex-runtime/manager_audit_issue_state.json`
  最近一条 manager audit issue。
- `.codex-runtime/issue_bridge_events.jsonl`
  通过 `s` bridge 产生的结构化事件。

每个游戏自己的 worker 真源仍在：

```text
/home/openclaw/babel-microgames/<game>/.codex-runtime/claudecode_workers.json
```

## 验证

```bash
find scripts -maxdepth 1 -type f -name '*.sh' -print0 | xargs -0 -n1 sh -n
sh scripts/claudecode_manager_status.sh
git diff --check
```
