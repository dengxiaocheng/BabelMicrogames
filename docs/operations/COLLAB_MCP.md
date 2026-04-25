# 会话协作 MCP

本文描述当前 `online` 会话与 Babel / C++ 专用会话之间如何共享结构化协作上下文。

## 目标

这个 MCP 不是为了同步两条 Codex 会话的完整聊天历史。

它的目标是把真正需要跨会话稳定保存的协作信息显式化，包括：

- 当前 Go / C++ 边界契约
- 哪个会话正在活着
- 哪个会话认领了哪些 scope
- 最近发布了哪些 handoff
- 哪些 handoff 已经被接手
- 最近阶段进度和对应 commit
- 最近发布了哪些结构化产物，例如共享库路径

## 边界

这个 MCP 是节点级运维与协作状态，不是 canonical runtime state。

它不能替代：

- 运行时持久化
- 仓库 canonical docs
- Babel 侧 C++ 边界文档
- GitHub issue / commit / PR 这些正式交付物

它只解决一个问题：

两个会话如何在不共享隐式聊天上下文的前提下，仍然能严格协作。

因此，当前约定是：

- 会话当前状态优先看 `state.json`
- 过程动作优先看 `events.jsonl`
- Termux / watcher / manual-resume 这类固定入口负责自动写入关键 heartbeat
- 文档负责解释规则，不负责在每一轮对话里充当唯一事实源

## 默认状态目录

默认写入：

- `~/.codex-runtime/collab/state.json`
- `~/.codex-runtime/collab/events.jsonl`
- `~/.codex-runtime/collab/lock`

之所以不放在当前仓库的 `.codex-runtime/` 下，是因为这份状态需要同时被：

- `/home/openclaw/babel-runtime`
- `/home/openclaw/Babel`

两个仓库里的会话共享。

如果需要改路径，可以设置：

- `BABEL_COLLAB_STATE_DIR`

如果希望把 collaboration MCP 事件继续送进节点本地自动化，可以设置：

- `BABEL_COLLAB_EVENT_HOOK`

## 当前角色约定

当前默认角色是：

- `online`
  协调会话；负责 Go runtime、节点流程、边界契约和 handoff 组织

- `babel-cpp`
  Babel / C++ 专用实现会话；负责明确认领后的 C++ scope

这个角色分工不是靠聊天记忆维护，而应通过：

- `set_contract`
- `heartbeat`
- `claim_scope`
- `publish_handoff`
- `ack_handoff`

这些显式工具持续反映。

## Managed C++ Workflow

当前固定的逻辑 session ID 是：

- `online`
- `babel-cpp`

当前固定的默认角色是：

- `online-go-runtime`
- `babel-cpp-core`

推荐的 managed C++ workflow 是：

1. `online` 会话更新边界契约
2. `online` 会话发布 handoff 给 `babel-cpp`
3. `babel-cpp` 会话认领明确的 C++ scope
4. `babel-cpp` 会话确认 handoff 并开始实现
5. 在 scope 不重叠的前提下，双方并行推进各自的 lane，而不是默认等待对方停下
6. 双方持续通过 `report-progress` 更新阶段进展
7. Babel / C++ 会话产出可消费二进制时，用 `publish_artifact` 显式发布产物路径
8. `online` 会话可按 `@collab` 或显式读取 artifact 的方式消费这些产物
9. `online` 会话只在真实依赖点上根据 progress、artifact 和新的代码基线决定下一轮 handoff / pull

当前有两个自动接线点：

- `online` 仓库里的 `open-stage / close-active / watch / manual-resume`
  会自动刷新 `online` 的 heartbeat

- Babel 仓库自己的 `open-stage / close-active / watch / manual-resume`
  会自动刷新 `babel-cpp` 的 heartbeat

`m` 现在只是 Babel 固定线程的统一入口，不再承担唯一的状态写入职责。

这意味着：

- `online` 会话可以像协调器一样管理 Babel / C++ 会话
- Babel / C++ 会话的活跃态不会再只存在于聊天历史里
- 但 scope 认领和 handoff 仍然必须显式声明，不能靠自动猜

## 工具集合

当前 `go run ./cmd/babel-collab-mcp` 暴露的是 MCP server，主要工具有：

- `set_contract`
  更新当前 Go / C++ 边界契约和必读引用

- `read_state`
  读取当前结构化协作状态

- `heartbeat`
  更新某个会话的 repo、role、status、thread、note

- `claim_scope`
  认领目录、模块或子系统

- `release_scope`
  释放之前认领的 scope

- `report_progress`
  记录阶段进度、关键变更路径和 commit

- `publish_artifact`
  发布结构化产物，例如 `scene_host_library -> /abs/path/libbabel_scene_core.so`

- `publish_handoff`
  发布显式 handoff

- `ack_handoff`
  确认某条 handoff 已被接手

另外还提供两个资源：

- `collab://state/current`
- `collab://contract/current`

适合只读拉取当前快照。

同时，命令行也支持直接调用同名管理动作：

```bash
go run ./cmd/babel-collab-mcp heartbeat --session-id online --repo babel-runtime --role online-go-runtime --status active
go run ./cmd/babel-collab-mcp claim-scope --session-id babel-cpp --repo Babel --scope src/
go run ./cmd/babel-collab-mcp publish-artifact --session-id babel-cpp --repo Babel --kind scene_host_library --path /abs/path/libbabel_scene_core.so
go run ./cmd/babel-collab-mcp publish-handoff --from-session-id online --to-session-id babel-cpp --repo Babel --title "实现 deterministic core" --summary "请接手 settlement core 的重构。"
go run ./cmd/babel-collab-mcp ack-handoff --session-id babel-cpp --handoff-id <id>
```

## 推荐协作流

一个最小稳定流程应是：

1. `online` 会话先用 `set_contract` 固定当前边界
2. Babel / C++ 会话用 `heartbeat` 标记自己已在线
3. Babel / C++ 会话用 `claim_scope` 认领目标 C++ 模块
4. `online` 会话用 `publish_handoff` 把本轮实现目标交给它
5. Babel / C++ 会话用 `ack_handoff` 确认接手
6. `online` 会话继续推进自己已认领的 Go scope，不因 handoff 自动阻塞
7. Babel / C++ 会话继续推进自己已认领的 C++ scope，不等待 `online` 空转
8. 实现过程中双方持续 `report_progress`
9. 可消费二进制一旦产出，应通过 `publish_artifact` 发布产物路径
10. 只有出现真实依赖时，才通过 `publish_handoff` / `拉取` / commit-based sync 交接
11. 实现完成后释放 scope 或发布下一条 handoff

## 并行原则

固定原则是：

- `set_contract + claim_scope + ack_handoff` 完成后，默认进入并行开发
- handoff 是依赖声明，不是自动 stop-the-world 信号
- 只要 scope 不重叠，`online` 和 `babel-cpp` 就应各自沿自己的 lane 前进
- 只有在 ABI、payload shape、requirement refs 或新 commit 消费点上，才需要显式同步

## 和 issue bridge 的关系

issue bridge 解决的是：

- 当前终端不在场时，怎么恢复线程
- 怎么把 GitHub issue comment 变成下一条指令

collaboration MCP 解决的是：

- 当前 `online` 会话和 Babel / C++ 会话如何共享显式上下文

两者不是同一层：

- issue bridge 管“唤醒和等待”
- collaboration MCP 管“边界、认领、handoff 和进度”

## 调试入口

人工查看当前状态：

```bash
go run ./cmd/babel-collab-mcp snapshot
```

查看最近事件：

```bash
go run ./cmd/babel-collab-mcp events --tail 20
```

如果 Babel 专用会话要复用这个节点级 MCP，只需要让它指向同一个状态目录，不需要复制任何仓库内状态文件。
