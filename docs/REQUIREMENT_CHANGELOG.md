# 需求变更记录

这份文件记录 design / requirement 变化从进入仓库到被吸收的生命周期。

它的目的，是防止已确认的决策只留在聊天记录里，或者散落在不同文档中无法追踪。

## 用法

每一条进入 `docs/incoming/` 的重要 requirement / design 变化都应在这里登记。

每条记录至少应包含：

- 进入日期
- 变化内容
- 当前审查状态
- 更新了哪些 canonical docs
- 是否影响实现规划

## 状态值

- `incoming`
- `reviewed`
- `accepted`
- `rejected`
- `archived`
- `implemented`

## 范围值

- `runtime`
- `gateway`
- `control-plane`
- `llm`
- `store-recovery`
- `godot`
- `cplusplus`
- `requirement-system`
- `cross-cutting`

## 模板

## 2026-04-25

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  Codex manager 每次处理 ClaudeCode worker handoff 时，也要有自己的 issue 记录。新增 `scripts/claudecode_manager_audit_issue.sh`，通过 `s` Go bridge 在 `BabelMicrogames` 打开 manager 级 audit issue 并立即关闭；`scripts/claudecode_worker_finish.sh` 默认在 worker handoff 成功后调用该审计入口，除非显式设置 `CLAUDECODE_MANAGER_AUDIT_ISSUE=0`。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `README.md`
  - `docs/operations/CLAUDECODE_MANAGER.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/claudecode_manager_audit_issue.sh`
  - `scripts/claudecode_worker_finish.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  ClaudeCode manager 的 issue bridge 默认实现不再指向 `/home/openclaw/claudecode-manager/.codex-runtime/bin/babel-issue-bridge`。新增 `scripts/claudecode_issue_bridge.sh` 作为薄包装，统一转发到 `/home/openclaw/babel-runtime/scripts/stage_issue_bridge.sh`，因此 worker queue、stage issue、manager-handoff、watcher 事件和 `BABEL_ISSUE_BRIDGE_EVENT_HOOK` 都复用 `s` 的 Go bridge。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `README.md`
  - `docs/operations/CLAUDECODE_MANAGER.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/claudecode_issue_bridge.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  Codex manager 的调度状态从“占位仓本地 worker registry”收敛为“扫描每个 `BabelMicrogame-*` game workdir 的 manager 总表”。新增 `.codex-runtime/microgame_manager_state.json` 生成入口、状态摘要入口和历史污染状态归档入口；`autorun` 现在可以从 manager workdir 扫描 `/home/openclaw/babel-microgames/*`，但真正派发 ClaudeCode worker 时只进入目标游戏 workdir，避免再次污染 `s / m` 或 manager 占位仓。
  后续补强了两条失败治理规则：worker 队列优先按阶段顺序排序，避免因更新时间跳过 state/content/ui 顺序；ClaudeCode 固定 session 被占用时标 `blocked`，不再误判为代码 rework，也不会继续派发该游戏后续 worker。
- 更新的 canonical docs：
  - `README.md`
  - `AGENTS.md`
  - `docs/operations/CLAUDECODE_MANAGER.md`
  - `docs/operations/CODEX_MANAGER_INTELLIGENCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/claudecode_manager_refresh_state.sh`
  - `scripts/claudecode_manager_status.sh`
  - `scripts/claudecode_manager_clean_legacy_state.sh`
  - `scripts/claudecode_manager_autorun.sh`

- 状态：`implemented`
- 范围：`control-plane`
- 变化：
  `dengxiaocheng/BabelMicrogames` 正式定位为独立 Codex manager 的资料、流程和脚本仓库，不再作为任何具体小游戏源码仓。新增微游戏工厂端到端流程文档和 Codex manager 智能化路线文档，明确当前 manager 只是“能调度”，还没有做到 incoming 自动消化、全局状态集中、结构化验收、失败降级和跨游戏调度。当前树同步移除了历史残留的小游戏源码和 per-game plan，源码真源只保留在各自 `BabelMicrogame-*` 仓库。
- 更新的 canonical docs：
  - `README.md`
  - `AGENTS.md`
  - `docs/INDEX.md`
  - `docs/OPERATIONS.md`
  - `docs/operations/README.md`
  - `docs/operations/CLAUDECODE_MANAGER.md`
  - `docs/operations/MICROGAME_FACTORY_FLOW.md`
  - `docs/operations/CODEX_MANAGER_INTELLIGENCE.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `/home/openclaw/claudecode-manager`
  - `dengxiaocheng/BabelMicrogames`

## 2026-04-23

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  `s / m` 现在不再只依赖 watchdog 和 SSH keepalive。当前新增显式的 manual lease：Termux 本地会周期性调用 `touch-manual-lease`，远端 `manual-resume` 会监视 `.codex-runtime/manual_leases/<entrypoint>.json`。如果 lease 缺失、session id 被替换、时间戳非法，或 TTL 内没有刷新，`manual-resume` 就会主动结束当前 `codex` 手动链。与此同时，新增统一的远端 helper `scripts/termux_manual_remote.sh`，把 `manual-resume` 和 `touch-lease` 的远端环境准备收敛到一处，避免 `s / m` 继续堆叠复杂引号。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/ops/issuebridge/bridge.go`
  - `internal/ops/issuebridge/command.go`
  - `scripts/termux_rewrite_s_one_shot.sh`
  - `scripts/termux_rewrite_m_one_shot.sh`
  - `scripts/termux_manual_remote.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  `s / m` 的 Termux 启动器不再只依赖 `exec ssh` 和远端 `ClientAlive*`。当前改成“前台 `ssh` + 本地 watchdog + 远端 keepalive 兜底”的组合：启动器完全基于 `sh`，不再要求 Termux 上有可直接执行的 `bash` 解释器路径；外层脚本壳继续持有 watchdog 所需的 FIFO，前台子壳先写入本地 `ssh` 的 pidfile，再 `exec ssh ...` 进入真正的前台 tty，避免后台 `ssh` 导致启动乱码和逐字显示；同时在 Termux 本地起一个轻量 watchdog，不只监视当前脚本壳持有的 FIFO，也监视本地 tty 是否已经挂断。一旦状态栏 `Exit` 让脚本壳消失，或脚本壳还活着但 tty 已先挂断，watchdog 都会主动结束本地 `ssh` 客户端。真正的 `ssh` 进程显式关闭这个 FIFO 写端，因此不会让 watchdog 因自身存活而误判。这样 `Exit` 的目标语义从“等待远端约 30 秒收尾”收紧成“正常路径下本地先杀 ssh，远端只做兜底”。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/termux_rewrite_s_one_shot.sh`
  - `scripts/termux_rewrite_m_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  Termux 侧 `s / m` 启动器现在用 `exec ssh ...` 结束脚本，而不是让 `ssh` 作为 shell 的普通子进程运行。这样状态栏 `Exit` 结束当前 Termux session 时，更容易直接把本地 `ssh` 客户端一起带走，再由服务器侧的 `ClientAlive*` 在有界时间内清理远端手动链。
- 更新的 canonical docs：
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/termux_rewrite_s_one_shot.sh`
  - `scripts/termux_rewrite_m_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  `cleanup-manual` 的同类清理逻辑已补上当前会话 `PGID` 排除，避免新的 `termux_s / termux_m` 在启动时先把自己所在的进程组误杀，出现 `Terminated` 后立刻断开的情况。当前语义收敛为：只清“旧的同类链”，不清“当前刚启动的这条链”。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/ops/issuebridge/bridge.go`
  - `internal/ops/issuebridge/command.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  节点手动会话现在正式进入“同类单槽位接管”模式。`manual-resume` 在真正接管前会先按 `entrypoint` 清掉同类旧手动链：新的 `termux_s` 会先清旧 `termux_s`，新的 `termux_m` 会先清旧 `termux_m`。同时新增显式排障入口 `cleanup-manual`，用于在服务器上按 `thread_id + entrypoint` 主动清理旧手动客户端。`m` 入口也改为统一复用 `babel-runtime` 仓库内的 `babel-issue-bridge` 二进制，避免 Babel 侧运维工具实现继续漂移。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/ops/issuebridge/bridge.go`
  - `internal/ops/issuebridge/command.go`
  - `scripts/termux_rewrite_m_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  方案一增强版已在当前节点落地：不再只依赖 `Termux Exit` 是否立刻让远端 pty 消失，而是给 `sshd` 增加显式 dead-client 探测。当前节点新增 `/etc/ssh/sshd_config.d/90-babel-runtime-session.conf`，启用 `TCPKeepAlive yes`、`ClientAliveInterval 15` 和 `ClientAliveCountMax 2`。这意味着状态栏 `Exit` 结束本地 Termux session 后，服务器通常会在约 `30` 秒内确认死连接并清理远端 `manual-resume -> codex resume` 手动链；文档里原先“本地退出即可瞬时清理”的表述也同步修正为“有界时间内清理”。
- 更新的 canonical docs：
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  当前服务器节点的运维基线已显式纳入 canonical docs。`docs/operations/NODE_RUNTIME.md` 现在不仅记录节点机器/系统基线、长期监听端口、`tmux/systemd/cron` 侧观测、repo-local `.codex-runtime/` 结构，以及 Codex 代理环境文件路径与本地代理端口用途；还补充了节点迁移清单、固定的节点清单命令、当前 live 环境变量快照，以及当前 token/proxy 文件的实际内容与权限。这样当前机器迁移所需的信息已经不再依赖聊天记录或现场再提取。
- 更新的 canonical docs：
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  `s` / `m` 的 Termux 重写脚本已改成在远端用独立进程组承载手动会话。重新执行 `scripts/termux_rewrite_s_one_shot.sh` 和 `scripts/termux_rewrite_m_one_shot.sh` 后，如果直接关闭 Termux 或 SSH 断开，远端对应的 `manual-resume -> codex resume` 整棵手动会话进程树会一起被杀掉，不再留下旧交互客户端残留；常驻 watcher 不受影响。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `scripts/termux_rewrite_s_one_shot.sh`
  - `scripts/termux_rewrite_m_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  为 Go/C++ 并行协作补充共享 `SceneHost` fixture。当前仓库新增 `FixturePair(solo_step|room_step)` 和 `go run ./cmd/babel-dev scene-host-fixture --mode ... --kind ...`，可以直接导出标准 request/response JSON，供 `babel-cpp` 对齐 `babel_sim_step` ABI，或供 Go 侧做 smoke/harness 验证。这样双会话并行开发时，不必只靠文档描述 payload 形状。
- 更新的 canonical docs：
  - `README.md`
  - `docs/INTERFACES.md`
  - `docs/TESTING.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/corehost/fixtures.go`
  - `cmd/babel-dev/main.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  Go 侧新增 `verify-scene-host-library`，可以直接用 `solo_step / room_step` 标准 fixture 对 Babel/C++ 共享库执行 ABI 兼容性验证。这样 `online` 与 `babel-cpp` 并行开发时，Babel 侧一旦产出最小 `.so`，当前仓库就能立刻做 smoke 校验，而不是继续靠人工比对 request/response JSON。
- 更新的 canonical docs：
  - `README.md`
  - `docs/INTERFACES.md`
  - `docs/TESTING.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/corehost/verify.go`
  - `cmd/babel-dev/main.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  Go 侧现在还具备一条最小 shared-library smoke harness。测试会临时编出 fixture-based `.so`，再用真实 `dlopen + babel_sim_step` 跑通 `verify-scene-host-library`。这样后续 `babel-cpp` 产出最小共享库时，`online` 不只是“理论上能验证”，而是已经有一条与真实加载路径一致的 smoke template。
- 更新的 canonical docs：
  - `docs/TESTING.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/corehost/sharedlib_smoke_test.go`

- 状态：`implemented`
- 范围：`runtime`
- 变化：
  `wechatapp` 现在把 shared-library 集成状态暴露成运行态显式信息，而不是只藏在启动逻辑里。若配置了 `BABEL_SCENE_CORE_LIBRARY`，服务启动前会先跑标准 fixture 验证；`/healthz` 会显式返回当前 `scene_host_mode / scene_host_verified / scene_host_library / scene_host_contract`，便于 `online` 与 `babel-cpp` 联调时快速判断当前是否真的跑在共享库上。
- 更新的 canonical docs：
  - `README.md`
  - `docs/INTERFACES.md`
  - `docs/TESTING.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/app/wechatapp/app.go`
  - `internal/app/wechatapp/app_test.go`
  - `internal/corehost/verify.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  collaboration MCP 现在支持结构化 artifact 发布。Babel / C++ 会话可通过 `publish-artifact --kind scene_host_library --path ...` 发布共享库产物路径；`babel-wechatd` 则支持 `BABEL_SCENE_CORE_LIBRARY=@collab`，直接从最新 `scene_host_library` artifact 解析共享库路径。这让两个会话的协作从“共享 handoff 和文档”进一步收束成“共享实际可消费的产物引用”。
- 更新的 canonical docs：
  - `README.md`
  - `docs/INTERFACES.md`
  - `docs/operations/COLLAB_MCP.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/ops/collabmcp/store.go`
  - `internal/ops/collabmcp/command.go`
  - `internal/ops/collabmcp/server.go`
  - `cmd/babel-wechatd/main.go`

- 状态：`implemented`
- 范围：`runtime`
- 变化：
  `BABEL_SCENE_CORE_LIBRARY=@collab` 的运行态语义进一步收紧：若 collaboration MCP 中没有可消费的 `scene_host_library` artifact，`babel-wechatd` 启动必须直接失败，不能再静默回退到 Go fallback。与此同时，`/healthz` 现在显式返回 `scene_host_source`，让联调时可以直接看出当前共享库来自本地默认 fallback、显式路径，还是 collab artifact。
- 更新的 canonical docs：
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `docs/INTERFACES.md`
  - `docs/TESTING.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-wechatd/main.go`
  - `cmd/babel-wechatd/main_test.go`
  - `internal/app/wechatapp/app.go`
  - `internal/app/wechatapp/app_test.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  Go/C++ 双会话的默认协作语义已明确改为“共享边界后并行开发”，而不是“一个等一个”。在 `set_contract + claim_scope + ack_handoff` 已完成且 scope 不重叠时，`online` 应继续推进自己的 Go lane，`babel-cpp` 应继续推进自己的 C++ lane；handoff 只表示依赖声明和接手边界，不表示发送方自动停工。只有在 ABI、payload shape、requirement refs 或新 commit 消费点上，才需要显式进入 `pull / handoff` 同步。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `docs/operations/COLLAB_MCP.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`runtime`
- 变化：
  `SceneHost` 进一步从“纯 Go 接口”推进到“可切 shared-library 的 Go/C++ 宿主边界”。`internal/corehost` 现在除了本地 fallback host，还新增了 shared-library scene host skeleton：Go 侧会把 `RuntimeRecord`、resolved `RuntimeRequirements`、当前 scene state 和 action 打包成 JSON request，通过 `babel_sim_step / babel_sim_free` 窄 `C ABI` 调用未来 Babel/C++ 导出的 deterministic scene core。`wechatapp` 和 `babel-wechatd` 也已增加 `SceneCoreLibraryPath` / `BABEL_SCENE_CORE_LIBRARY` 配置位，后续只要给出共享库路径，就能把 `solo_scene / room_scene` 切到同一条 ABI 上。
- 更新的 canonical docs：
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/INTERFACES.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/corehost/sharedlib.go`
  - `internal/corehost/dlopen_linux_cgo.go`
  - `internal/app/wechatapp/app.go`
  - `cmd/babel-wechatd/main.go`

- 状态：`implemented`
- 范围：`runtime`
- 变化：
  微信 `solo_scene / room_scene` 现在不再只是“把 snapshot 写成主文档”，而是把 requirement bundle 引用和 deterministic 房间摘要一起接进运行链。`wechatapp` 已给 kernel 注入 filesystem requirement registry，scene runtime 在创建时会绑定 `bootstrap.ruleset / bootstrap.prompt_pack / bootstrap.gameplay_asset`，并把这些引用反映到 canonical state 与 `primary_context.md / session_manifest.json`；联机状态文本也新增了已提交玩家、待提交玩家和上轮结算摘要，避免房间进度只剩模糊的 AI 描述。
- 更新的 canonical docs：
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/INTERFACES.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/app/wechatapp/app.go`
  - `internal/corehost/`
  - `internal/gateway/wechat/runtime_bridge.go`
  - `internal/gateway/wechat/scene_memory.go`
  - `internal/settlement/simple.go`
  - `internal/projection/simple.go`

- 状态：`implemented`
- 范围：`gateway`
- 变化：
  微信 `solo_scene / room_scene` 现在会把当前 runtime snapshot 同步派生到 file-backed operational memory。`RuntimeBridge` 在返回 transport-facing 回复后，会额外写出 `primary_context.md`、`session_manifest.json`、`scene_state.json` 和 `recent_summary.md`；`solo_scene` 会固定暴露最近动作、最近回复和当前玩家状态，`room_scene` 会固定暴露房间阶段、玩家列表、轮次和最近公共事件。这样后续长会话 worker 可以读取一份稳定的主文档继续工作，但 canonical state 仍然只来自 runtime repository，不由文档反向成为真相源。
- 更新的 canonical docs：
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/gateway/wechat/runtime_bridge.go`
  - `internal/gateway/wechat/scene_memory.go`
  - `internal/app/wechatapp/app.go`
  - `internal/app/wechatapp/app_test.go`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  协作 MCP 从“只有 MCP server”扩展成“server + 节点 CLI + 自动 heartbeat 接线”。`cmd/babel-collab-mcp` 现在支持直接在 shell / Termux 中执行 `heartbeat`、`claim-scope`、`release-scope`、`report-progress`、`publish-handoff`、`ack-handoff`，并新增 `BABEL_COLLAB_EVENT_HOOK` 供节点本地自动化消费事件。同时，`online` 仓库的 `open-stage / close-active / watch / manual-resume` 会自动刷新 `online` 的 heartbeat；`babel-cpp` 的 heartbeat 也不再只依赖 `m` 入口，而是由 Babel 仓库自己的 `issue bridge / manual-resume / watch` 生命周期直接刷新。这样当前 `online` 会话已经可以稳定协调 Babel / C++ 专用会话，而不需要依赖隐式聊天上下文来判断哪条会话正握着执行权；流程识别也优先来自结构化 state、events、hook 和固定入口，而不是要求 assistant 在每轮对话里重新阅读整套流程文档。
- 更新的 canonical docs：
  - `README.md`
  - `docs/OPERATIONS.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/COLLAB_MCP.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `internal/ops/collabmcp/command.go`
  - `internal/ops/issuebridge/collab.go`
  - `internal/ops/issuebridge/command.go`
  - `scripts/termux_rewrite_m_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  新增节点级协作 MCP，用来让当前 `online` 会话和 Babel / C++ 专用会话共享结构化协作上下文，而不是假设两个会话能同步隐式聊天上下文。新命令 `go run ./cmd/babel-collab-mcp` 默认把协作状态写到 `~/.codex-runtime/collab/`，覆盖边界契约、session heartbeat、scope 认领、handoff、ack 和记要；同时新增 `snapshot` 和 `events` 入口供人工排查。仓库规则也同步明确：当前 `online` 会话可以协调 Babel / C++ 专用会话，但跨会话同步必须通过显式状态、handoff 和文档边界，而不能靠聊天记忆。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `README.md`
  - `docs/INDEX.md`
  - `docs/OPERATIONS.md`
  - `docs/operations/README.md`
  - `docs/operations/COLLAB_MCP.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-collab-mcp/main.go`
  - `internal/ops/collabmcp/`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  Babel 专用的 Termux 入口 `m` 改为由当前 online 仓库统一托管，并与 `s` 采用同一形态。新增 `scripts/termux_rewrite_m_one_shot.sh`，它会在 Termux 上重写 `$PREFIX/bin/m`，SSH 到服务器、切到 `openclaw`、进入 `/home/openclaw/Babel`，并通过 `manual-resume` 恢复 Babel 的固定专用 Codex 线程。这样 `s` 与 `m` 都由同一节点工具仓库管理：`s` 负责恢复当前 runtime 线程，`m` 负责恢复 Babel 的固定线程。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  把跨节点 handoff 的标准短指令固定为 `拉取`。当用户在实时开发节点推送后，只要在当前 stage issue 顶层评论 `拉取` 并关闭 issue，服务器节点就应按默认同步路径处理：检查当前分支与 upstream，执行 fast-forward 同步，读取新增 commit 和关键 diff，再继续当前任务。同步完成后，后续仍按原来的小阶段流程重新创建下一条 stage issue。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  明确双节点开发 handoff 规则。服务器节点继续承担当前 Codex 线程的等待点与 watcher；另一台实时开发节点，例如用户自己的 Windows 电脑，可以直接复用同一套代码仓开发和推送，但不需要在本地重复 issue watcher 仪式。推送后若要把执行权交回服务器节点，必须复用当前 stage issue，在顶层 comment 中明确写出 pull / sync 指令后再关闭 issue；恢复后的服务器线程需要先同步工作区，再继续执行。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `README.md`
  - `docs/OPERATIONS.md`
  - `docs/operations/README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  `install-hooks` 也已迁入 Go。`cmd/babel-dev` 新增 `install-hooks` 子命令，负责把 `.githooks/pre-commit` 安装为当前仓库的 `core.hooksPath`；原 `scripts/install_local_hooks.sh` 已删除。至此，仓库侧运维/守卫入口已经不再依赖 shell 脚本，只剩 Git hook 文件本身和 Termux 环境安装脚本属于 shell 边界。
- 更新的 canonical docs：
  - `README.md`
  - `AGENTS.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-dev/main.go`
  - `.githooks/pre-commit`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  继续收束 shell 运维入口。`start-watcher`、`stop-watcher` 和 `manual-resume` 现在已经成为 `cmd/babel-issue-bridge` 的一等子命令，对应的 shell 流程脚本已删除；README、运维文档和 Termux `s` 重写脚本均改为直接调用 Go 子命令。这样当前仓库里只剩环境安装类 shell 文件，不再让 shell 持有 issue bridge 的实际流程。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-issue-bridge/main.go`
  - `internal/ops/issuebridge/command.go`
  - `scripts/termux_rewrite_s_one_shot.sh`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  issue bridge、watcher 和 manual takeover 运维链路已从 Python 全量收编到 Go。新增 `cmd/babel-issue-bridge` 与 `internal/ops/issuebridge`，覆盖阶段 issue 创建、watcher 轮询恢复、终端 handoff 消费、manual claim/release、结构化事件日志和 GitHub API 访问。与此同时，仓库里的最后一份 Python 运维脚本及其测试已删除，`docs-sync-guard` CI 也移除了 Python 依赖。
- 更新的 canonical docs：
  - `README.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/governance/DOC_MANIFEST.json`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-issue-bridge/main.go`
  - `internal/ops/issuebridge/`
  - `scripts/start_codex_issue_watcher.sh`
  - `scripts/stop_codex_issue_watcher.sh`
  - `scripts/codex_manual_resume.sh`
  - `.github/workflows/docs-sync-guard.yml`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  仓库级 guard tooling 开始从 Python 收编到 Go。新增 `cmd/babel-dev` 和 `internal/devtools/quality`，把文档一致性检查、diff-based sync guard、requirement asset 校验和 guard report 渲染统一到 Go 命令入口；本地 `.githooks/pre-commit`、GitHub Actions `docs-sync-guard`、README 和治理文档均改为调用 Go 入口。与此同时，仓库规则进一步明确：后续新增实现默认只允许使用 `Go` 和 `C++`，不再新增 Python；现存 Python 仅允许作为待迁移遗留。
- 更新的 canonical docs：
  - `README.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- 实现入口：
  - `cmd/babel-dev/main.go`
  - `internal/devtools/quality/`
  - `.githooks/pre-commit`
  - `.github/workflows/docs-sync-guard.yml`

- 状态：`implemented`
- 范围：`cross-cutting`
- 变化：
  增加基于 GitHub issue 的阶段续跑机制。小阶段在 `commit + push` 后可创建阶段 issue，本机 watcher 在检测到“用户评论并关闭 issue”后，对同一条 Codex `thread_id` 执行 `codex resume`。
  阶段 issue 默认应 assign 并 `@mention` 用户；GitHub token 的本地默认路径固定为 `.codex-runtime/github-token.env`。
  同时增加线程接管控制文件 `.codex-runtime/thread_control.json`：服务器平时只保留 watcher 等待；Termux `s` 会先人工抢占线程，并自动中断 watcher 拉起的自动客户端。
  如果用户已经在当前活动终端直接回复，则当前终端应直接关闭当前阶段 issue；默认使用用户原话作为关闭 comment，并把这条 comment 记为已消费，避免 watcher 再次误触发新客户端。
  如果用户是通过 Termux `s` 或服务器本机手动接管线程，则手动接管脚本会立即关闭当前等待中的阶段 issue，不再保留一个悬空的 GitHub 等待点。
  阶段结束后的“下一步决策请求”不应只存在于 issue 正文；同源的 terminal handoff 文案也应写入 state，并在聊天末尾同步提示。
  另外增加 `.codex-runtime/issue_bridge.lock`，用来串行化当前终端关闭 issue 与 watcher 恢复线程的关键窗口，降低两边并发误触发。
  终端回复是否消费活动 issue，不再依赖聊天语义猜测；只有当 state 明确处于 waiting 状态时，才允许当前终端回复关闭 issue。执行过程中被读取的普通输入不应误触发该流程。所有这类流程变化都必须同步写入运维文档。
  为了降低 Termux 前台噪音，继续把运维流程动作迁移到 `.codex-runtime/issue_bridge_events.jsonl`。watcher 启停、手动接管、stage issue 开闭、自动恢复等动作应优先写入结构化操作日志，而不是在前台脚本输出多行说明。
  issue bridge 事件还可通过 `BABEL_ISSUE_BRIDGE_EVENT_HOOK` 继续送入节点本地 hook，降低把流程说明打印到对话里的需求。
  同时明确 `docs/incoming/` 的所有权限制：只有用户可以向该目录新增文件；助手不能主动新增，也不能在仅仅识别到新 incoming 文档时就自行处理。只有用户显式要求处理时，助手才可以把它融合进现有文档，并在完成后归档原文件。
  为了减少上下文污染，文档同步流程改为按需读取：先通过 `docs/INDEX.md` 选择最小读取集合，只在信息不足或边界冲突时再扩展，不再把整套文档默认读入每一轮对话。
- 更新的 canonical docs：
  - `AGENTS.md`
  - `docs/OPERATIONS.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `README.md`
- 实现入口：
  - `cmd/babel-issue-bridge/main.go`
  - `internal/ops/issuebridge/`
  - `scripts/codex_manual_resume.sh`
  - `scripts/start_codex_issue_watcher.sh`
  - `scripts/stop_codex_issue_watcher.sh`

## 2026-04-22 - diff-based doc sync guard added

- status: `implemented`
- scope: `cross-cutting`
- source: 继续把同步矩阵做成更硬的实现前守卫
- incoming_doc: n/a
- summary:
  在 `DOC_MANIFEST.json` 中新增 `implementation_guard`，把路径触发规则也纳入机器可读定义；新增 `scripts/check_docs_sync_guard.py`，可根据当前 `git diff` 推导触发的同步矩阵项，并在没有任何对应 canonical 文档变更时直接失败；同时增加 `.githooks/pre-commit` 和 `scripts/install_local_hooks.sh`，把这层守卫前移到本地提交前。
- canonical_docs_updated:
  - `AGENTS.md`
  - `README.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 如后续需要更细粒度的守卫，可继续把路径规则拆到子系统级，而不是只按当前大类触发
- notes:
  - 这层守卫仍然是最小门：要求“至少有一份相关 canonical 文档随变更一起修改”，而不是强制每次全量重写整组文档

## 2026-04-22 - subsystem-level doc guard and CI enforcement

- status: `implemented`
- scope: `cross-cutting`
- source: 继续把同步矩阵做成更细粒度的子系统级守卫，并接入 CI
- incoming_doc: n/a
- summary:
  把 diff-based 文档同步守卫从粗粒度类别收紧到 `mode`、`gateway`、`kernel`、`repository/recovery`、`agent/projection/delivery`、`testkit`、兜底 runtime 结构等子系统级触发项；同时新增 `.github/workflows/docs-sync-guard.yml`，让 GitHub PR / `main` push 也执行 manifest 校验、脚本单测和 diff-based 文档守卫。
- canonical_docs_updated:
  - `README.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 如果未来 requirement-management foundation 形成独立代码目录，可再为它补一组专门的 trigger 和 CI 范围规则
- notes:
  - CI 侧通过 `--merge-base-with` / `--base --head` 计算比较范围，避免把错误的 diff 基线带进文档守卫

## 2026-04-22 - requirement foundation assets and stronger CI gate

- status: `implemented`
- scope: `requirement-system`
- source: 继续把同步矩阵扩展到 requirement-management foundation 的目录/资产级守卫，并把 CI 做成更强的 PR gate
- incoming_doc: n/a
- summary:
  建立仓库内 `requirements/` 基础目录、registry、schema 和 bootstrap asset；新增 `scripts/check_requirement_assets.py` 与对应测试，校验 registry、asset path 和跨资产引用；同步把 docs sync guard 扩展到 requirement registry foundation / requirement asset content 两组子系统级类别；GitHub Actions 增加路径过滤、并发控制、失败日志 artifact 和 step summary，本地 `.githooks/pre-commit` 也会串起 requirement asset 校验。
- canonical_docs_updated:
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/INTERFACES.md`
  - `docs/ROADMAP.md`
  - `docs/STORAGE_SCHEMA.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/TESTING.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 下一步应把 repo 内 versioned requirement asset 真正接到 runtime loader / resolver，而不是只停在文件存在和引用校验
- notes:
  - CI 现在只在相关路径变化时运行 docs-sync-guard，并会在失败时上传 `.ci-artifacts` 供排查

## 2026-04-22 - runtime requirement loader and PR guard feedback

- status: `implemented`
- scope: `cross-cutting`
- source: 把 requirement-management foundation 接成 runtime loader / resolver，并继续强化 CI 反馈
- incoming_doc: n/a
- summary:
  新增 `internal/requirementregistry` 的 filesystem-backed loader / resolver，能从仓库内 `requirements/` 目录解析 ruleset、prompt pack、gameplay asset、content constraint 和 experiment trace；kernel 在执行前会解析 runtime 上声明的 requirement bundle，并显式传入 `ModeExecutionInput.Requirements`。同时新增 `scripts/render_guard_report.py`，让 CI 将 guard 结果渲染成 markdown，并在 PR 上更新 sticky comment。
- canonical_docs_updated:
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/INTERFACES.md`
  - `docs/ROADMAP.md`
  - `docs/STORAGE_SCHEMA.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/TESTING.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/README.md`
  - `docs/REQUIREMENT_SYNC_CHECKLIST.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 下一步应把 resolved requirement bundle 真正用于 mode / deterministic core 的行为分支，而不是只停在解析与传递
- notes:
  - 当前 PR 流程下，guard 的最新状态会通过 sticky comment 回写到 PR，而不只存在于 Actions 页面

## 2026-04-22 - canonical docs modularized for on-demand reading

- status: `implemented`
- scope: `cross-cutting`
- source: 文档体系收束
- incoming_doc: n/a
- summary:
  canonical 文档从“根目录平铺入口”收束为专业模块入口。新增 `docs/architecture/`、`docs/runtime/`、`docs/governance/`、`docs/operations/` 四个模块 README，并把文档治理和运维重文档拆成模块子文档；`INDEX.md`、`DESIGN_SYNC_WORKFLOW.md`、`OPERATIONS.md` 退化为总入口页。默认读取路径改为“先模块、后最小集合、再按需扩展”，以降低上下文污染。
- canonical_docs_updated:
  - `docs/INDEX.md`
  - `docs/DESIGN_SYNC_WORKFLOW.md`
  - `docs/OPERATIONS.md`
  - `docs/governance/DOC_SYSTEM.md`
  - `docs/governance/INCOMING_WORKFLOW.md`
  - `docs/governance/README.md`
  - `docs/operations/NODE_RUNTIME.md`
  - `docs/operations/ISSUE_BRIDGE.md`
  - `docs/operations/README.md`
  - `docs/architecture/README.md`
  - `docs/runtime/README.md`
  - `README.md`
- implementation_followup:
  - 如后续需要更硬的一致性约束，可继续增加文档同步守卫或按主题读取 manifest
- notes:
  - 当前已增加 `scripts/check_docs_consistency.py`，用于检查模块入口和相对链接是否断裂
  - 当前已增加 `docs/governance/DOC_MANIFEST.json`，用于承载模块集合、topic 路由和同步矩阵的机器可读定义

```md
## YYYY-MM-DD - <short title>

- status:
- scope:
- source:
- incoming_doc:
- summary:
- canonical_docs_updated:
  - ...
- implementation_followup:
  - ...
- notes:
```

## 记录

## 2026-04-22 - runtime redesign reset

- status: accepted
- scope: cross-cutting
- source: 对当前实现评审后提出的直接重设计请求
- incoming_doc: `docs/incoming/2026-04-22-runtime-redesign-reset.md`
- summary: 围绕 unified runtime kernel、mode module、显式 agent task supervision、projection / delivery 分离和 execution-centric persistence 重置目标架构。
- canonical_docs_updated:
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/SYSTEM_TARGET_ARCHITECTURE.md`
  - `docs/ROADMAP.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/INTERFACES.md`
  - `docs/STORAGE_SCHEMA.md`
  - `docs/TESTING.md`
- implementation_followup:
  - 把当前包拆分当成可替换脚手架，而不是固定事实
  - 实现统一 execution kernel 和 repository contract
  - 在新 kernel 之上重建 product mode，而不是继续扩展旧 thin runtime split
- notes:
  - 这是一条架构级变化，明确丢弃所有会阻碍重建的现有实现包袱

## 2026-04-22 - design sync workflow established

- status: accepted
- scope: cross-cutting
- source: 仓库流程设计
- incoming_doc: n/a
- summary: 确立 `docs/incoming/` 作为设计文档落点，并定义默认的 fetch / diff / triage / canonicalize 流程。
- canonical_docs_updated:
  - `docs/DESIGN_SYNC_WORKFLOW.md`
  - `docs/INDEX.md`
- implementation_followup:
  - 后续所有 incoming design drop 都应使用这套流程
- notes:
  - 这条记录确立了仓库级 intake process

## 2026-04-22 - canonical docs switch to Chinese-first prose

- status: implemented
- scope: cross-cutting
- source: 针对文档语言与可读性的直接反馈
- incoming_doc: n/a
- summary: 将 canonical docs 调整为中文主导 prose，英文继续保留在代码标识符、协议关键字、路径和命令示例中；同时补强核心执行链注释要求。
- canonical_docs_updated:
  - `docs/INDEX.md`
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/SYSTEM_TARGET_ARCHITECTURE.md`
  - `docs/ROADMAP.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/INTERFACES.md`
  - `docs/STORAGE_SCHEMA.md`
  - `docs/TESTING.md`
- implementation_followup:
  - 后续 canonical docs 默认写中文
  - 核心执行链补足边界注释和阶段注释
- notes:
  - 代码标识符保持英文，不影响 Go 层命名一致性

## 2026-04-23 - wechat test account no longer routes admin commands

- status: implemented
- scope: transport / ops
- source: 直接移除微信测试号管理员菜单入口
- incoming_doc: n/a
- summary: `internal/gateway/wechat` 不再识别或处理管理员 slash command；测试号入口只保留用户向的 `free_chat` 和 `project_consult` 路由，未知 slash command 统一落到 unsupported reply。
- canonical_docs_updated:
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 微信 transport 不再依赖 `controlplane.AdminResponder`
  - 管理员能力保留在 control plane / operator surface，不再复用测试号聊天入口
- notes:
  - 这是一次边界收束，不影响 `controlplane` 包自身的 operator-facing 能力

## 2026-04-23 - online defers stable deterministic core ownership to Babel/C++

- status: implemented
- scope: architecture
- source: 明确要求 `online` 聚焦 Godot 替代验证与服务器编排，不再默认重写 Babel/C++ 稳定核心
- incoming_doc: n/a
- summary: 收束 `online` 与 Babel/C++ 的边界：Go 运行时优先承担编排、requirement 验证、projection 与 delivery；`time_core`、`population_core`、`economy_core`、`construction_core`、`social_core`、`narrative_core`、`settlement_core` 的长期稳定所有权留给 Babel/C++。
- canonical_docs_updated:
  - `docs/ARCHITECTURE.md`
  - `docs/SYSTEM_TARGET_ARCHITECTURE.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - `online` 继续优先建设 Go host orchestration、projection、delivery 和 Godot-facing validation flow
  - 后续需要通过 host adapter / requirement bundle 明确复用 Babel/C++，而不是继续扩写 Go 版稳定 world rule
- notes:
  - 这条收束不否认 Go 侧存在过渡性 deterministic 验证脚手架，但它不再被视为长期目标

## 2026-04-23 - new Go wechat service absorbs player menu mode switching

- status: implemented
- scope: transport / product
- source: 旧微信桥接归档，测试号改测新的 Go 服务
- incoming_doc: n/a
- summary: 新增 `cmd/babel-wechatd` 与 `internal/app/wechatapp`，为 Go 版 `wechat` handler 补上健康检查、旧菜单键兼容、用户向模式切换和会话内 route 记忆；旧管理员菜单键在新服务中统一返回 retired reply。
- canonical_docs_updated:
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 用新的 Go 服务替换旧的 Python 微信桥接进行测试
  - 后续再决定是否为新版服务显式下发简化后的玩家菜单
- notes:
  - 该服务当前只吸收 `free_chat` / `project_consult` 的用户向菜单切换，不继续承载旧管理员菜单能力

## 2026-04-23 - wechat Go service now exposes minimal solo roleplay

- status: implemented
- scope: transport / product
- source: 按“玩家主链优先”开始逐项迁移测试号菜单
- incoming_doc: n/a
- summary: 新的 Go `wechat` 服务已接通 `MODE_GAME -> solo_scene` 的最小链路；点击“角色扮演”后，后续自由文本会进入 `solo_scene`，并返回可读的玩家向投影文本。测试号菜单现为 `角色扮演 / 项目咨询 / 自由聊天`。
- canonical_docs_updated:
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 下一步继续补 `决 / 想`
  - 再补 `祈祷` 与 `反馈`
- notes:
  - 当前 `solo_scene` 仍是最小验证版，不等于旧 Python 桥接完整剧情体验

## 2026-04-23 - wechat Go service now exposes player menu command groups

- status: implemented
- scope: transport / product
- source: 继续按玩家主链迁移测试号菜单，并将 Go 侧定位明确为 Godot 替代验证入口
- incoming_doc: n/a
- summary: 微信新服务现已吸收 `决 / 想 / 模式` 三组玩家菜单，并把 `角色扮演` 明确收束为 `单人角色`。`决` 按钮会下发快捷行动，`想` 按钮会下发内心活动命令，其中“祈祷”已并入“眺望远方”。`模式` 组现承载 `单人角色 / 项目咨询 / 自由聊天 / 反馈 / 联机`；`反馈` 会落盘到 file-backed operational log，`联机` 会进入共享 `room_scene` 验证链路。
- canonical_docs_updated:
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/SUBSYSTEM_BOUNDARIES.md`
  - `docs/SYSTEM_TARGET_ARCHITECTURE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 继续把 `room_scene` 从最小共享房间验证推进到更完整的 lobby / round 体验
  - 后续把稳定联机核心逐步对齐到 Babel/C++ 宿主接口，而不是在 Go 中长期固化
- notes:
  - Go 侧当前联机实现仅用于流程与需求快速验证，不构成 Babel/C++ 稳定多人核心的长期替代

## 2026-04-23 - wechat multiplayer validation flow now requires two active players

- status: implemented
- scope: transport / product
- source: 微信测试号联机验证中发现共享房间会累积旧成员，导致回合要求人数异常
- incoming_doc: n/a
- summary: Go 侧 `room_scene` 验证链已修正为“至少两名活跃玩家才开局”；单人点击 `联机` 或提前发送行动时只会看到等待提示，不会误提交回合。共享房间同时增加了超时成员自动清理，避免旧测试账号长期残留并卡住联机回合。
- canonical_docs_updated:
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 继续把最小联机房间推进到更完整的 lobby / round / scene summary 体验
  - 后续再把房间级 Claude Code / artifact memory 接到 agent supervisor，而不是让 agent 取代 canonical state
- notes:
  - 当前“活跃玩家”判定仍然是 Go 验证期策略，不代表 Babel/C++ 长期联机核心最终接口

## 2026-04-23 - wechat multiplayer validation flow now exposes room stage text

- status: implemented
- scope: transport / product
- source: 继续把微信联机从纯计数器验证推进到可读的房间体验，同时明确 AI 不独占机制约束
- incoming_doc: n/a
- summary: 微信 `联机` 状态文本现在会直接显示大厅 / 回合阶段、玩家列表、当前轮次和最近房间事件，不再只是 `人数 / 提交数`。这些文本全部由 Go 侧确定性状态生成，用于验证多人体验节奏；AI 不承担完整机制限制，长期数值与系统约束仍要逐步收回到 Go / Babel-C++。
- canonical_docs_updated:
  - `docs/ARCHITECTURE.md`
  - `docs/FEATURE_INHERITANCE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 继续补更完整的 round resolution summary 和房间级 artifact memory
  - 后续再把房间级 scene summary 更明确地对齐到 Babel/C++ 宿主接口
- notes:
  - 当前房间文本仍是验证期产物，重点是让多人节奏和规则边界更快暴露，而不是作为最终联机 UX

## 2026-04-23 - single-player memory converges toward one primary document and one working session

- status: implemented
- scope: architecture / agent
- source: 明确要求单人模式与全局模式不要把所有关键操作交给 AI，但应支持“一个主文档 + 一个主会话”的长期工作记忆
- incoming_doc: n/a
- summary: `agent` operational memory 现在除了派生小文件，还会固定产出 `primary_context.md` 和 `session_manifest.json`，作为后续 `Claude Code` 长会话的主上下文入口。与此同时，架构文档明确：这个主会话只拥有 working memory，不拥有 canonical state authority；关键操作和规则推进仍然要逐步落回 `Go host + Babel/C++ core`。
- canonical_docs_updated:
  - `docs/ARCHITECTURE.md`
  - `docs/SYSTEM_TARGET_ARCHITECTURE.md`
  - `docs/REQUIREMENT_CHANGELOG.md`
- implementation_followup:
  - 后续把单人模式的实际 Claude Code 长会话接到 `session_manifest.json` 这一层
  - 房间级和全局级模式也沿用“主文档 + 主会话 + runtime authority”的同一结构
- notes:
  - 当前仍只是 working memory 结构收束，不代表 AI 已取得单人或全局模式的操作 authority
