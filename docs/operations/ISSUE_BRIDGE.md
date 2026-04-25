# 阶段 Issue Bridge

本文描述 stage issue、watcher、manual takeover、结构化操作日志和 hook 的运转方式。

## 目标

如果希望在 Codex 当前终端会话停下后，改用 GitHub issue 驱动下一步继续执行，最小链路是：

1. 当前阶段完成后先 `commit + push`
2. 创建阶段 issue
3. 启动 watcher
4. 用户在该 issue 下评论下一步指令，并关闭 issue
5. watcher 检测到“关闭 + 新评论”后，自动继续同一条 Codex 线程

这个链路继续的是同一个 Codex `thread_id`，但不是原来的同一个前端窗口。

## 本地状态文件

issue bridge 使用的节点本地文件包括：

- `.codex-runtime/issue_bridge_state.json`
- `.codex-runtime/thread_control.json`
- `.codex-runtime/issue_bridge.lock`
- `.codex-runtime/issue_bridge_events.jsonl`

它们都属于节点本地运维状态，不是项目 canonical state。

## 结构化操作日志与 Hook

为了降低前台噪音，issue bridge 的关键流程事件会写入：

- `.codex-runtime/issue_bridge_events.jsonl`

需要排查时，用：

```bash
go run ./cmd/babel-issue-bridge events --tail 20
```

如果节点本地希望把这些事件继续送到别的运维边界，可以设置：

- `BABEL_ISSUE_BRIDGE_EVENT_HOOK`

它应是一个本地 shell command。每当事件被追加时，脚本会把单条 JSON 通过标准输入发给这个 hook。

## 认证与通知边界

本地脚本访问 GitHub REST API 时，优先读取：

- `BABEL_GITHUB_TOKEN`
- `GITHUB_TOKEN`

仓库默认的本地 token 文件路径是：

- `.codex-runtime/github-token.env`

阶段 issue 是否推送到 GitHub Mobile，属于用户账号和移动端 App 的通知设置。

## 创建阶段 Issue

示例：

```bash
go run ./cmd/babel-issue-bridge open-stage \
  --title "阶段汇报：kernel watcher automation" \
  --report-file /tmp/stage_report.md \
  --decision-request "请在本 issue 直接评论下一步，并在评论后关闭 issue。" \
  --thread-id "$CODEX_THREAD_ID"
```

## 当前终端直接续跑时的关闭规则

如果用户没有通过 issue 评论，而是直接回到当前活动终端继续下达指令，那么：

- 只有当前 state 处于 `handoff_status=waiting` 时，这条终端回复才算 handoff 回复
- 当前活动终端应直接关闭当前阶段 issue
- 默认直接使用当前终端收到的用户原话作为 comment
- comment 要在本地 state 中标记为已消费，避免 watcher 再次误触发

如果用户是在执行过程中主动插话、打断或补充信息，而系统还没有进入 waiting 状态，那么：

- 这条消息不应消费当前阶段 issue
- 也不应触发 handoff 流程

## Handoff 体验

阶段结束后，不应只把“下一步决策请求”写进 GitHub issue。

当前约定是：

- issue body 里保留完整阶段报告和决策请求
- 终端结尾也同步给出同一条决策请求
- 终端结尾带上当前阶段 issue 链接

## Manager Handoff

除了“当前终端直接消费 handoff”之外，还存在一条额外语义：

- 当前终端不是最终消费者
- 当前 issue 仍然要被关闭
- 但 watcher 之后仍然应该恢复固定管理线程

这个场景适用于：

- `ClaudeCode worker -> Codex manager`
- 其他非 watcher worker -> Codex 管理线程

这时不能用：

```bash
go run ./cmd/babel-issue-bridge close-active ...
```

因为 `close-active` 的默认语义是：

- 当前活动终端已经接手
- watcher 不必再恢复新的客户端

这里必须使用：

```bash
go run ./cmd/babel-issue-bridge manager-handoff --comment-file <path>
```

它会：

- 创建一条新评论
- 关闭当前 waiting issue
- 保留这条评论为“未消费”
- 让 watcher 之后仍然能据此恢复固定管理线程

## 跨节点 Handoff

stage issue 不只用于“手机不在当前终端”这一个场景，也可以作为服务器节点和另一台实时开发节点之间的 handoff 通道。

推荐语义是：

- 服务器节点是当前 Codex 线程的等待节点
- Windows 等实时开发节点是用户主动操控的实现节点
- 实时开发节点本地开发时，不需要把自己的编辑循环也套进 issue watcher 流程
- 但当它把代码推到远端、希望服务器节点继续当前线程时，应复用当前打开的 stage issue 来交还执行权

推荐操作顺序：

1. 服务器节点完成当前小阶段，`commit + push` 后创建 stage issue，并进入 waiting
2. 用户转到实时开发节点继续开发、提交、推送
3. 推送完成后，用户在当前 stage issue 顶层评论“先 pull / sync，再继续”的明确指令
4. 用户关闭该 issue
5. watcher 恢复同一条 Codex 线程，并把这条 comment 作为下一步输入

这里保留一个标准短指令：

- `拉取`
  含义固定为“服务器线程先同步当前工作区到当前跟踪分支，读取刚拉下来的代码变动，再继续当前任务”

要注意：

- issue comment 只是把指令送回服务器节点，不会自动更新服务器工作区
- 恢复后的服务器线程必须先 `git fetch` / `git pull` / `git switch` 到 comment 指定的状态，再继续改动
- 如果 comment 只有 `拉取`，就按默认同步路径处理：检查当前分支与 upstream，执行 fast-forward 同步，读取新增 commit / diff，然后继续
- 如果 comment 没写清楚分支、远端或 sync 方式，服务器线程不应擅自假设
- 不要让服务器节点和实时开发节点在同一分支上并发做未协调的实现工作

## Watcher 生命周期

前台调试：

```bash
go run ./cmd/babel-issue-bridge watch
```

后台常驻：

```bash
go run ./cmd/babel-issue-bridge start-watcher
```

停止：

```bash
go run ./cmd/babel-issue-bridge stop-watcher
```

这两个脚本现在默认只输出一行状态；更详细的启动、跳过、停止记录会写进 `issue_bridge_events.jsonl`。

## Watcher 触发条件

watcher 只会在以下条件同时满足时恢复线程：

- 当前活动 issue 已关闭
- 该 issue 有新的非空顶层评论
- 该评论尚未被上一次 resume 消费
- 当前线程没有被人工接管
- 当前没有 watcher 先前拉起且仍存活的自动恢复客户端

## 竞态与锁

为了降低 race，当前实现使用：

- `.codex-runtime/issue_bridge.lock`

当前终端在执行 `close-active` 时会持有这把锁；watcher 在真正准备拉起新的 `codex resume` 前，也会先持有同一把锁并重读最新 state。

## 手动接管优先级

手动 `s` 唤醒优先于 watcher 自动恢复。

当前做法是：

1. `s` 在服务器上执行 `go run ./cmd/babel-issue-bridge manual-resume`
2. 这个脚本先把线程标记为 `manual`
3. 自动关闭当前仍在等待中的阶段 issue
4. 如果 watcher 已经拉起自动客户端，就先中断那个 `tmux` 会话
5. 然后再进入用户自己的交互式 `codex resume`
6. 退出后再把线程控制状态释放回 `idle`

当前还新增了一条硬约束：

- `manual-resume` 在真正 `claim-manual` 前，会先按 `entrypoint` 清掉同类旧手动链
- 也就是新的 `termux_s` 会先清旧 `termux_s`，新的 `termux_m` 会先清旧 `termux_m`
- 这样即使上一次 `Termux Exit` 没让远端立刻感知断线，也不会无限积累旧的同类手动客户端

可显式执行的排障命令是：

```bash
go run ./cmd/babel-issue-bridge cleanup-manual --thread-id <thread> --entrypoint termux_s
go run ./cmd/babel-issue-bridge cleanup-manual --thread-id <thread> --entrypoint termux_m
go run ./cmd/babel-issue-bridge touch-manual-lease --thread-id <thread> --entrypoint termux_s --session-id <id>
```

为避免“新的手动会话一启动就把自己清掉”，`cleanup-manual` 现在还会显式排除当前手动会话所在的 `PGID`。因此它清的是“旧的同类链”，不是“当前刚启动的这条链”。

为降低手动入口的冷启动开销，当前标准 `s / m` 启动器会优先直接执行仓库内预编译的：

- `.codex-runtime/bin/babel-issue-bridge`

如果该二进制不存在，才会在首轮启动时执行一次 `go build`。因此高频使用前，可先在对应仓库执行：

```bash
go run ./cmd/babel-dev install-ops-binaries
```

另外，`manual-resume` 会显式把当前工作目录传给 `codex`。这样即使某条线程最早是在别的目录建立，也不会因为“session 记录目录”和“当前目录”不一致而弹出目录选择提示。

如果是通过 Termux `s` / `m` 进入手动会话，当前标准启动器还会在远端把这组手动客户端放进独立进程组。因此：

- 直接关闭 Termux
- 或 SSH 链接异常断开

都应把对应的 `manual-resume -> codex resume` 进程组一起杀掉，而不是在节点上留下陈旧交互客户端。这个行为只针对手动会话；watcher 仍应继续常驻。

当前标准启动器还要求同时满足：

- SSH 侧必须保留交互 stdin，避免 `codex` 报 `stdin is not a terminal`
- 远端必须显式提供可用的终端类型，避免 `codex` 因 `TERM=dumb` 直接退出
- Termux 状态栏 `Exit` 后，当前本地 `ssh` 客户端应尽快结束，而不是只靠远端 `sshd` 超时发现
- 快速退出时又必须整组清理远端手动客户端

因此当前实现使用“强制 TTY + 本地 watchdog + 远端前台进程组运行 + 断开时 kill 当前进程组”的组合，而不是再用会吞掉交互 stdin 的 here-doc，或会丢掉控制终端的 `setsid` 形态。

本地 watchdog 的职责是：

- 在 Termux 本地额外起一个轻量监视进程
- 同时监视当前脚本壳持有的 FIFO 写端是否还存在，以及本地前台 tty 是否已经挂断
- 一旦状态栏 `Exit` 先结束本地脚本壳，就立即向本地 `ssh` 客户端发送 `TERM/KILL`
- 如果脚本壳还活着，但 Termux 已经先把前台 tty 挂断，也会直接向本地 `ssh` 客户端发送 `TERM/KILL`

为了避免 watchdog 自己把 FIFO 一直占着，真正的 `ssh` 客户端会显式关闭该写端；这样 FIFO 的生命周期只跟脚本壳绑定，不跟 `ssh` 绑定。

当前实现还新增了一层更硬的退出语义：

- `s / m` 在本地会周期性调用 `touch-manual-lease`
- `manual-resume` 启动后会监视对应 `entrypoint` 的 lease 文件
- 如果 lease 缺失、session id 被替换、时间戳非法，或超过 TTL 没有刷新，远端就会主动结束当前手动链

同时，Termux 侧生成脚本现在要求：

- 使用 `sh` 作为 shebang
- 外层脚本壳继续持有 watchdog 所需的 FIFO
- 前台子壳先写入本地 `ssh` 的 pidfile，再 `exec ssh ...`
- 本地 heartbeat 在每轮续租前都要显式检查原始启动器 PID、原始父进程、原始 tty，以及当前前台 `ssh` pidfile/tty；任一失效时，heartbeat 必须立刻停止，而不是继续孤儿续租

这样外层脚本还能继续持有 watchdog 所需的本地状态，而真正的交互显示仍由前台 `ssh` 承担，不再出现“启动时打印杂乱字符、进入 `codex` 后一行只有一个字”的后台 tty 副作用，也不再依赖 Termux 上必须存在某个固定的 `bash` 解释器路径。

另外，真正的“断线即清理”责任现在不再只押在外层 shell trap 上。`manual-resume` 本身也会：

- 把 `codex resume` 放进独立进程组
- 捕获 `SIGHUP / SIGTERM / Ctrl-C`
- 在这类退出路径上把整组 `codex` 手动子进程一起结束

这样即使 SSH 链路或 Termux 先断，清理责任也仍然落在 Go 入口，而不是完全依赖启动脚本能否及时跑完 shell trap。再配合同类单槽位接管和 lease 监视，当前节点的手动会话语义已经变成：

- 正常 `Exit` 时优先由本地 watchdog 结束 `ssh`
- watchdog 没跑到时，远端 lease 到期后也会主动结束手动链
- watchdog 没跑到时，再由 SSH keepalive 在有界时间内清理
- 即使没及时清理，下次同类入口也会先接管并清旧

## 可见性与并发约束

`codex resume` 继续的是同一条会话线程，但它可能出现在新的客户端窗口或新的 `tmux` pane 中。

因此：

- 同一条线程的历史会延续
- 旧的、已经停下的前端窗口不会自动刷新这些新消息
- 同一个 `thread_id` 不应同时由多个活跃客户端并发驱动
- 另一台开发节点的代码推送也不应被视为服务器工作区已经自动同步；必须经过显式 handoff comment 和实际 pull / sync
