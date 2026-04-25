# ClaudeCode 管理工作流

本文描述一个最小可用的管理流。

端到端流程先读：

- [MICROGAME_FACTORY_FLOW.md](./MICROGAME_FACTORY_FLOW.md)

manager 智能化路线先读：

- [CODEX_MANAGER_INTELLIGENCE.md](./CODEX_MANAGER_INTELLIGENCE.md)

- `ClaudeCode` 负责执行被拆好的 worker 任务
- `Codex` 负责管理、验收、排队和续跑
- GitHub issue 继续作为 handoff 边界
- worker registry 和 queue 由当前仓库自己的 `issue bridge` 维护

## 独立仓库边界

`dengxiaocheng/BabelMicrogames` 是 manager 资料仓，不是具体小游戏源码仓。

ClaudeCode 做微游戏的流水线必须使用一游戏一仓库：

- `dengxiaocheng/BabelMicrogame-*`

它不能再复用：

- `dengxiaocheng/BabelOnline-GoCpp`
- `dengxiaocheng/Babel`

原因很简单：`s`、`m` 自己的长期会话也依赖各自仓库的 stage issue。ClaudeCode worker 如果复用这些仓库，会污染长期会话的 waiting issue 和 watcher 状态。

本仓库脚本会在关键入口执行：

```bash
sh scripts/claudecode_manager_repo_guard.sh
```

只要 `origin` 不是 `dengxiaocheng/BabelMicrogame-*` 前缀，manager/worker handoff 会直接拒绝运行。

## Codex manager 独立工作目录

Codex manager 固定只在这个目录恢复和执行：

```bash
/home/openclaw/claudecode-manager
```

小游戏源码目录只给 ClaudeCode worker 和测试/提交使用，例如：

```bash
/home/openclaw/babel-microgames/gongtou-dianming
```

正确关系是：

- game workdir 保存游戏代码、worker registry、worker packet、worker report
- manager workdir 保存调度脚本、bridge 二进制、管理规则
- game issue watcher 可以从 game workdir 读取 `.codex-runtime/issue_bridge_state.json`
- 但 issue state 里的 `workdir` 必须是 `/home/openclaw/claudecode-manager`
- 因此 watcher 恢复 Codex 时回到管理目录，而不是游戏源码目录

打开游戏阶段 issue 必须使用：

```bash
sh scripts/claudecode_manager_open_game_stage.sh \
  --game-workdir /home/openclaw/babel-microgames/gongtou-dianming \
  --repo dengxiaocheng/BabelMicrogame-GongtouDianming \
  --title "工头点名 / 下一阶段" \
  --report "..." \
  --decision-request "..."
```

该脚本会把 issue state 写在游戏 workdir，同时把 Codex resume 目录固定为 manager workdir。

## 推荐运行形态

推荐保持：

- 常驻进程只跑轻量 Go watcher
- Codex manager 不常驻工作
- ClaudeCode worker 完成后关闭 issue 并留言
- watcher 检测到关闭 issue 后，用 `codex exec resume` 唤醒 Codex manager 一次
- Codex manager 处理队列、审查 report、派下一包或重新进入 waiting，然后退出

启动专用 watcher：

```bash
sh scripts/claudecode_manager_start_watcher.sh
```

启动持续调度器：

```bash
sh scripts/claudecode_manager_autorun.sh --daemon
```

它只在以下条件同时满足时拉起下一个 ClaudeCode worker：

- 当前不在 14:00-18:00 quiet hours
- `claudecode_manager` tmux socket 下没有 `claudecode_worker_*` 会话
- worker queue 中存在 `queued` 或 `rework` 任务

如果希望队列清空后不再进入人工决策点，而是按固定 Codex 策略继续做下一条微游戏线，使用：

```bash
sh scripts/claudecode_manager_autorun.sh --daemon --auto-seed-microgames
```

固定策略：

- 队列有任务：继续派发下一个 worker
- 队列为空：从内置 backlog 自动种下一条微游戏线
- 当前默认 backlog：`dianming`、`shuiyuan`、`huijiang`、`yinji`、`bingpeng`
- 没有 backlog 时才保持空转等待

停止专用 watcher：

```bash
sh scripts/claudecode_manager_stop_watcher.sh
```

这个 watcher 使用独立 tmux session：

- `claudecode_manager_watch`

自动恢复客户端使用独立前缀：

- `claudecode_manager_resume`

它不复用 `termux_s`、`termux_m`，也不应该让当前人工聊天会话长期充当 manager。

## 目标

不要让 `ClaudeCode` 直接承担长期项目管理。

推荐分工是：

- `ClaudeCode`
  只做明确边界内的小任务

- `Codex`
  负责：
  - 开阶段 issue
  - 组织 worker brief
  - 等待 worker 结果
  - 接收 handoff
  - 审查和关闭当前小阶段

## 最小流程

### 1. Codex 管理线程先进入 waiting

由管理线程使用：

```bash
go run ./cmd/babel-issue-bridge open-stage ...
```

打开当前 worker 阶段 issue，并让 watcher 进入等待。

这一步的 thread id 仍然是 `Codex` 管理线程自己的 thread id，不是 Claude 的 session id。

### 2. 把 worker 放进 registry / queue

当前最小版已经提供：

```bash
go run ./cmd/babel-issue-bridge worker-register --worker-id <id> --task-title <title>
go run ./cmd/babel-issue-bridge worker-packet --worker-id <id> --lane <lane> --task-level <S|M|L> --task-title <title> --task-summary <summary>
go run ./cmd/babel-issue-bridge worker-next
go run ./cmd/babel-issue-bridge worker-queue
go run ./cmd/babel-issue-bridge worker-set-status --worker-id <id> --status done
```

worker queue 的目标不是自动调度一切，而是先让 `Codex manager` 能看到：

- 哪些任务还在 `queued`
- 哪些 worker 正在 `running`
- 哪些已经 `handoff_queued`

推荐现在直接补一层固定 packet：

```bash
sh scripts/claudecode_worker_packet.sh \
  --worker-id task-001 \
  --lane ui \
  --task-level M \
  --task-title "配给日 UI" \
  --task-summary "补发粮界面的首屏交互" \
  --goal "把按钮态和结算入口补齐" \
  --write-scope "src/ui/" \
  --read-scope "docs/" \
  --test-command "go test ./internal/ops/issuebridge" \
  --acceptance "按钮状态正确" \
  --acceptance "结算入口可达" \
  --constraint "不要顺手改别的场景" \
  --deliverable "更新代码并写 report"
```

它会固定生成：

- `.codex-runtime/claudecode_workers/<worker-id>/packet.md`
- `.codex-runtime/claudecode_workers/<worker-id>/report.md`

这样 manager 不用再手拼长提示，worker 完成后也有固定 report 落点。

现在 packet 里除了 `lane`，还支持固定任务尺度：

- `task-level`
- `max-files`
- `max-delta-lines`
- `read-scope`
- `write-scope`
- `test-command`

推荐默认标准：

- `S`
  - `max-files = 1`
  - `max-delta-lines = 200`
- `M`
  - `max-files = 3`
  - `max-delta-lines = 500`
- `L`
  - `max-files = 5`
  - `max-delta-lines = 800`

也就是说：

- `S` 适合单文件小修
- `M` 适合单模块闭环
- `L` 只适合接近上限的小型跨文件任务，再大就应该继续拆

如果你传了 `task-level` 但没显式传 `max-files / max-delta-lines`，系统会自动套默认预算。

另外现在 manager 不用再自己翻 queue，可以直接取“下一个该派发的 worker”：

```bash
go run ./cmd/babel-issue-bridge worker-next
```

它默认会同时带两层约束：

- `rework`
- `queued`
- `max-running = 1`
- `lane` 同类互斥

也就是说：

- 不会把 `running / handoff_queued` 当成新的派发对象
- 只要已经有一个 worker 在跑，默认就不再续派第二个
- 如果当前有 `ui` lane 在跑，默认也不会再派新的 `ui` lane

如果你要放宽它，可以显式：

```bash
go run ./cmd/babel-issue-bridge worker-next --max-running 2 --allow-same-lane
```

如果当前仓库里同时挂着多条项目线，推荐再加：

```bash
go run ./cmd/babel-issue-bridge worker-next --worker-prefix peigei-ri-
```

这样 manager 只会在指定项目前缀里挑 worker，不会被别的历史队列干扰。

### 3. 启动 ClaudeCode worker

使用：

```bash
sh scripts/claudecode_worker_resume.sh --workdir <repo> --worker-id <id> --session-id <claude-session-id>
```

它会先把对应 worker 标记成 `running`，再进入 `ClaudeCode` 会话。

如果该 worker 已经生成了 packet，脚本会默认识别：

- `.codex-runtime/claudecode_workers/<worker-id>/packet.md`

如需先把 packet 注入已有 Claude session，再进入交互，可显式加：

```bash
sh scripts/claudecode_worker_resume.sh --worker-id <id> --session-id <claude-session-id> --send-packet
```

如果要让 manager 直接挑下一个并启动 Claude worker，可直接用：

```bash
sh scripts/claudecode_manager_next.sh --session-id <claude-session-id> --send-packet
```

如果你没有现成的 Claude session id，也可以直接：

```bash
sh scripts/claudecode_manager_next.sh
```

这时脚本会在 packet 存在时自动把 packet 作为初始 prompt，直接起一个新的 Claude 会话，不再要求你先手工准备 session。

如果要让 worker 无人值守跑完并自动回交，推荐使用：

```bash
sh scripts/claudecode_manager_next.sh --run-once
```

`--run-once` 会调用 `scripts/claudecode_worker_run_once.sh`，用 `claude -p` 执行 packet，避免交互式 trust prompt 卡住不可接回的 TTY。微游戏工厂的一键启动默认走这个模式。

如果当前终端不应该承载 worker 进程，应把无人值守 worker 放进独立 tmux 会话：

```bash
sh scripts/claudecode_worker_start_tmux.sh --worker-prefix peigei-ri-
```

该脚本仍然走 `claudecode_manager_next.sh --run-once`，但 worker 进程挂在独立 tmux 会话里，不依赖当前 `s/m` 连接存活。

`claudecode_manager_start_watcher.sh` 和 `claudecode_worker_start_tmux.sh` 默认使用 `tmux -L claudecode_manager`，使 manager watcher、worker、自动恢复出的 manager Codex 客户端与默认 tmux server 隔离。

服务器每日 `14:00-18:00` 不运行 ClaudeCode 工作。

该规则由 `scripts/claudecode_quiet_hours_guard.sh` 执行：

```bash
sh scripts/claudecode_quiet_hours_guard.sh
```

静默时段内脚本会停止 `claudecode_manager_watch`、`claudecode_worker_*` 和 `claudecode_manager_resume_*` 会话，并把仍标记为 `running` 的 worker 改回 `rework`。静默时段外脚本只恢复 manager watcher，不主动派发 worker。

服务器侧 QA 不依赖真实浏览器或 Playwright。浏览器入口只做 Node/模块级验证、静态导入验证和 CLI/engine 逻辑验证；如需真实浏览器交互测试，应在有浏览器环境的机器上执行。

这个脚本默认也是保守模式：

- `--max-running 1`
- 同 lane 互斥

如果你要放宽：

```bash
sh scripts/claudecode_manager_next.sh --session-id <claude-session-id> --send-packet --max-running 2 --allow-same-lane
```

如果你只想看当前会派给谁，不想真正启动，就加：

```bash
sh scripts/claudecode_manager_next.sh --print-only
```

如果你要只派某个项目自己的 worker：

```bash
sh scripts/claudecode_manager_next.sh --worker-prefix peigei-ri- --print-only
```

如果要只派某个项目自己的 worker，并无人值守跑完：

```bash
sh scripts/claudecode_manager_next.sh --worker-prefix peigei-ri- --run-once
```

### 4. ClaudeCode worker 完成后回交给 Codex manager

使用：

```bash
sh scripts/claudecode_worker_finish.sh --workdir <repo> --worker-id <id> --comment-file <path>
```

如果没有额外指定 `--comment-file`，脚本会默认使用：

- `.codex-runtime/claudecode_workers/<worker-id>/report.md`

它内部会执行：

```bash
go run ./cmd/babel-issue-bridge worker-finish --worker-id <id> --comment-file <path>
```

语义是：

- 先把 worker 标记成 `handoff_queued`
- 创建一条新的 issue comment
- 关闭当前 waiting issue
- 不把这条评论标记成“当前终端已经消费”
- 让 watcher 仍然可以用这条评论恢复 `Codex` 管理线程

### 5. Codex manager 自动恢复

watcher 检测到：

- issue 已关闭
- 有新的未消费评论

之后会自动恢复 `Codex` 管理线程。

恢复后的 `Codex` 应只做管理职责：

- 读取 Claude worker 回交内容
- 查看 `worker-queue`
- 审查结果
- 决定继续拆下一任务、修补还是归档
- 再把对应 worker 标成 `done / rework / cancelled`

专用 manager watcher 使用 `--resume-mode exec`，所以它恢复的是一次性 `codex exec resume`，不是长期交互式 TUI。一次处理结束后，Codex 客户端退出；只有 Go watcher 继续等待下一条 issue 关闭事件。

## 为什么要用 manager-handoff

普通的：

```bash
go run ./cmd/babel-issue-bridge close-active ...
```

语义是：

- 当前活动终端已经接手
- watcher 不需要再恢复新客户端

这不适合 `ClaudeCode -> Codex manager` 交接。

所以这里必须使用：

```bash
go run ./cmd/babel-issue-bridge manager-handoff ...
```

它的语义是：

- 这条 issue 已经可以关闭
- 但恢复动作仍然应该交给 watcher 去拉起 `Codex` 管理线程

## 边界

这套最小流程当前只解决：

- `ClaudeCode worker` 如何把结果交回 `Codex manager`
- `Codex manager` 如何通过现有 watcher 自动恢复

它还没有解决：

- 自动发现新建 worker issue
- 自动批量调度多个 Claude worker
- 自动做 worker 队列和并发上限

这些属于下一阶段。

## 当前命令与脚本

- `scripts/claudecode_worker_resume.sh`
  进入 ClaudeCode worker 会话

- `scripts/claudecode_worker_packet.sh`
  生成固定 packet / report 文件，并把路径写回 worker registry；也支持直接写入 `task-level / budget / scope / test-command`

- `scripts/claudecode_worker_finish.sh`
  把 worker 结果通过 `manager-handoff` 交回 Codex manager

- `scripts/claudecode_manager_next.sh`
  让 manager 直接挑出当前最该派发的 worker，并可立即启动 Claude worker

- `scripts/claudecode_worker_start_tmux.sh`
  在独立 tmux 会话中启动一个无人值守 Claude worker，避免占用当前手动连接

- `scripts/microgame_factory_start.sh`
  用内置 Babel 创意预设一键开始一条微游戏任务链。它会先生成 plan 和 worker packets，再按当前项目前缀立即派出第一个 worker。

- `go run ./cmd/babel-issue-bridge worker-register`
  登记新 worker

- `go run ./cmd/babel-issue-bridge worker-packet`
  生成或刷新 worker 的标准任务包和 report 模板；可顺手写入 `lane / task-level / budget / scope / test-command`

- `go run ./cmd/babel-issue-bridge worker-next`
  取当前最该派发的 worker；默认只从 `rework / queued` 中选，并带 `max-running=1 + lane 互斥`。可额外用 `--worker-prefix` 只在某条项目线内选择

- `go run ./cmd/babel-issue-bridge worker-start`
  标记 worker 开始执行

- `go run ./cmd/babel-issue-bridge worker-finish`
  标记 worker 回交并执行 `manager-handoff`

- `go run ./cmd/babel-issue-bridge worker-queue`
  查看当前 queue

- `go run ./cmd/babel-issue-bridge worker-set-status`
  让 manager 在审查后把 worker 标成 `done / rework / cancelled`

## 下一阶段建议

后续如果要继续做，可以再加：

1. Claude worker 专用 issue label / topic
2. manager watcher 对 worker issue 的自动发现
3. manager 自动批量续派多个 Claude worker
4. manager 对失败 worker 的自动降级和暂停
