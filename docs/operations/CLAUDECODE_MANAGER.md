# ClaudeCode 管理工作流

本文描述 ClaudeCode worker 与 Codex manager 的最小闭环。

## 固定边界

- Codex manager workdir：`/home/openclaw/claudecode-manager`
- manager 仓库：`dengxiaocheng/BabelMicrogames`
- 小游戏 workdir：`/home/openclaw/babel-microgames/<game>`
- 小游戏仓库：`dengxiaocheng/BabelMicrogame-*`
- issue bridge 真源：`/home/openclaw/babel-runtime`

manager 仓不实现 Go bridge。所有 bridge 调用默认走：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_issue_bridge.sh ...
```

该脚本转发到 `s`：

```text
/home/openclaw/babel-runtime/scripts/stage_issue_bridge.sh
```

## 正确恢复目录

游戏阶段 issue 的 state 写在游戏 workdir：

```text
/home/openclaw/babel-microgames/<game>/.codex-runtime/issue_bridge_state.json
```

但 issue state 里的 `resume_workdir` 必须是：

```text
/home/openclaw/claudecode-manager
```

这样 watcher 检测到 worker 关闭 issue 后，恢复的是 Codex manager，而不是游戏源码目录。

打开游戏阶段 issue：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_manager_open_game_stage.sh \
  --game-workdir /home/openclaw/babel-microgames/gongtou-dianming \
  --repo dengxiaocheng/BabelMicrogame-GongtouDianming \
  --title "工头点名 / 下一阶段" \
  --report "..." \
  --decision-request "..."
```

## Worker 队列

每个游戏自己的 worker 真源是：

```text
/home/openclaw/babel-microgames/<game>/.codex-runtime/claudecode_workers.json
```

manager 总表只是索引：

```text
/home/openclaw/claudecode-manager/.codex-runtime/microgame_manager_state.json
```

查看状态：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_manager_status.sh
```

## Worker 启动

无人值守启动一个 worker：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_worker_start_tmux.sh \
  --workdir /home/openclaw/babel-microgames/<game> \
  --worker-prefix <game>-
```

自动调度器：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_manager_autorun.sh \
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

每日 `14:00-18:00` 不启动 ClaudeCode worker；autorun 会遵守 quiet hours。

## Worker 回交

worker 完成后通过：

```bash
sh /home/openclaw/babel-runtime/scripts/claudecode_worker_finish.sh \
  --workdir /home/openclaw/babel-microgames/<game> \
  --worker-id <worker-id>
```

默认行为：

- 用 `manager-handoff` 把 report 评论到当前游戏 issue
- 关闭游戏 issue
- 调用 `/home/openclaw/babel-runtime/scripts/claudecode_manager_audit_issue.sh`
- 在 `BabelMicrogames` 打开并关闭一条 manager audit issue

audit state 单独写入：

```text
.codex-runtime/manager_audit_issue_state.json
```

不得写进主 state：

```text
.codex-runtime/issue_bridge_state.json
```

## Manager 审查

Codex manager 被 watcher 唤醒后，至少要检查：

- worker report 是否完整
- `git status --short`
- `git diff --check`
- 改动是否越过 packet write scope
- 测试命令是否执行
- 是否引入浏览器运行依赖或隐藏外部服务依赖
- 是否应该 `done / rework / blocked / cancelled`

只有通过审查的改动才提交和推送目标小游戏仓库。

## 阻塞规则

如果 Claude session 被占用或网络失败，不把它当作代码失败。

推荐状态：

- `blocked`：session 占用、网络故障、环境问题
- `rework`：代码实现不合格，需要同一 worker 修
- `cancelled`：任务方向废弃
- `done`：审查和测试都通过

manager 自动调度发现某个游戏有 `blocked` worker 时，不继续派发该游戏后续 worker。
