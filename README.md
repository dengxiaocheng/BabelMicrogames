# Babel Microgames Manager

这是 Babel 微游戏工厂的独立 Codex manager 资料与脚本仓库。

GitHub 仓库：

- `dengxiaocheng/BabelMicrogames`

本仓库用途：

- 保存独立 Codex manager 的规则、流程文档和调度脚本
- 保存微游戏工厂的计划模板、worker packet 规则和验收策略
- 作为 `/home/openclaw/claudecode-manager` 的远端归档

本仓库不是小游戏源码仓库。

小游戏源码必须进入独立仓库：

- `dengxiaocheng/BabelMicrogame-*`

不要把 ClaudeCode worker issue 写入 `BabelOnline-GoCpp` 或 `Babel`。那两个仓库继续服务 `s/m` 长期会话和对应 watcher。`BabelMicrogames` 也不应该承载某个具体小游戏的源码；它只承载 manager 资料和必要的 manager 级协调记录。

本仓库从 Babel 新运行时仓库拆出，因此仍保留一部分 runtime / issue-bridge 工具能力；当前重点是让 ClaudeCode 在 5000 行以内的 Babel 微游戏切片上持续工作，并由 Codex manager 做排队、审查和收口。

它继承已经验证过的产品能力，但不继承旧实现债务。当前方向是：

- 以 Go 作为运行时和编排核心
- 把确定性状态与结算放在代码和持久化层里
- 把 LLM 限定为渲染、总结和受控解释层
- 用 staged execution、checkpoint、recovery 保证可重启连续性
- 只在 Go 版本验证正确且出现明确热点后，再考虑 C++ 提取

## 当前重点

当前仓库优先建设的是新运行时地基，而不是旧系统的平移：

- 统一的 kernel / mode / repository / recovery 执行链
- transport 与 runtime core 的明确边界
- deterministic settlement 与 projection / delivery 分层
- agent task / artifact 流水线
- file-backed operational memory
- 从一开始就可测试、可重放、可恢复

## 语言约束

仓库后续默认统一使用 `Go` 和 `C++`。

- `Go`
  当前 runtime、仓库级工具、校验器和运维自动化的默认实现语言

- `C++`
  仅在 Go 版本已经验证正确且出现明确性能或所有权边界需求后，再进入提取阶段

- `Python`
  不再作为新增实现语言；仓库里现存的 Python 只允许作为待迁移遗留，后续应逐步收编到 Go

## 文档入口

建议从这里开始阅读：

- [docs/INDEX.md](docs/INDEX.md)
- [docs/operations/MICROGAME_FACTORY_FLOW.md](docs/operations/MICROGAME_FACTORY_FLOW.md)
- [docs/operations/CODEX_MANAGER_INTELLIGENCE.md](docs/operations/CODEX_MANAGER_INTELLIGENCE.md)
- [docs/operations/CLAUDECODE_MANAGER.md](docs/operations/CLAUDECODE_MANAGER.md)
- [docs/architecture/README.md](docs/architecture/README.md)
- [docs/runtime/README.md](docs/runtime/README.md)
- [docs/governance/README.md](docs/governance/README.md)
- [docs/operations/README.md](docs/operations/README.md)

## 文档分区

- `docs/`
  当前 canonical 文档

- `docs/architecture/`
  架构模块入口

- `docs/runtime/`
  运行时模块入口

- `docs/governance/`
  文档治理模块入口

- `docs/operations/`
  运维模块入口

- `docs/planning/`
  较早期的规划输入与参考草案

- `plan/`
  当前会话按用户要求新建的活跃计划文件

- `docs/incoming/`
  新 requirement / design 投递的暂存区

## 计划与本地脚本归档

- 当前执行计划默认放在 `plan/`
- Termux 本地脚本继续归档在 `scripts/`
- Windows 本地运行脚本统一归档到 `scripts/windows/`
- 对应的运维说明统一收口到 `docs/operations/`

## 实用脚本

- `go run ./cmd/babel-issue-bridge`
  创建阶段 issue、默认 assign 并 `@mention` 当前 GitHub 用户、写入本地 watcher 状态和 terminal handoff 文案，并在 issue 关闭且带评论后恢复同一条 Codex 线程；只有在显式 waiting 状态下，当前终端回复才会消费活动 issue。内部会使用 `.codex-runtime/issue_bridge.lock` 串行化 watcher 与当前终端的关键操作，并把关键流程记录到 `.codex-runtime/issue_bridge_events.jsonl`。默认会从 `.codex-runtime/github-token.env` 读取 GitHub token。另一台实时开发节点推送后，也可以通过关闭当前 stage issue 并写入“先 pull 再继续”的 comment，把执行权交回服务器节点。

- `go run ./cmd/babel-issue-bridge start-watcher`
  用 `tmux` 启动本地 issue watcher。前台只保留一行状态提示，详细流程写进 `issue_bridge_events.jsonl`。

- `go run ./cmd/babel-issue-bridge stop-watcher`
  停掉本地 issue watcher。前台只保留一行状态提示，详细流程写进 `issue_bridge_events.jsonl`。

- `go run ./cmd/babel-issue-bridge manual-resume --thread-id <thread>`
  供服务器本机或 Termux `s` 手动接管当前线程。进入交互前会先关闭当前等待中的阶段 issue、打断 watcher 拉起的自动客户端，并在退出后释放接管状态。恢复时会显式把当前仓库目录传给 `codex`，避免 thread 早期记录目录与当前目录不一致时弹出工作目录选择提示。

- `go run ./cmd/babel-issue-bridge manager-handoff --comment-file <path>`
  供非 Codex worker 在完成当前小任务后，把结果回交给 waiting 中的 Codex 管理线程。它会创建新评论并关闭当前 issue，但不会把这条评论标记成“当前终端已消费”，因此 watcher 仍然可以据此恢复管理线程。

- `go run ./cmd/babel-issue-bridge cleanup-manual --thread-id <thread> --entrypoint termux_s|termux_m`
  显式清理指定手动入口的旧会话链。`manual-resume` 现在在真正接管前也会自动执行同类清理，因此新的 `s` 会先清旧 `s`，新的 `m` 会先清旧 `m`。清理时还会排除当前手动会话所在的 `PGID`，避免“刚启动的新会话先把自己杀掉”。

- `go run ./cmd/babel-issue-bridge touch-manual-lease --thread-id <thread> --entrypoint termux_s|termux_m --session-id <id>`
  刷新当前手动会话的 lease。`s / m` 现在会在 Termux 本地周期性调用这条命令；远端 `manual-resume` 发现 lease 过期后，会主动结束当前手动链，不再单纯依赖 `Exit` 是否表现成某种固定的断线事件。

- `go run ./cmd/babel-dev install-ops-binaries`
  预编译当前仓库常用的节点运维二进制到 `.codex-runtime/bin/`。当前会安装 `babel-issue-bridge`，供 `s` 这类高频手动入口直接执行，避免每次都走 `go run`。

- `scripts/termux_rewrite_s_one_shot.sh`
  在 Termux 上一次性重写 `$PREFIX/bin/s`，让它 SSH 到远端节点、切到 `openclaw`、进入仓库目录，并通过 Go 版 `manual-resume` 子命令恢复指定的 Codex 会话。新生成的 `s` 会优先执行仓库内已编译的 `.codex-runtime/bin/babel-issue-bridge`；若缺失则只在首轮远端启动时 `go build` 一次。它完全基于 `sh` 运行：外层脚本持有 watchdog 所需的本地 FIFO，前台子壳先写入本地 `ssh` 的 pidfile，再 `exec ssh ...` 进入真正的前台 tty；同时本地会启动一个轻量 heartbeat，按固定间隔调用 `touch-manual-lease` 续租。一旦状态栏 `Exit` 让外层脚本壳消失，watchdog 会直接杀掉本地 `ssh` 客户端；即使这一步没发生，远端 `manual-resume` 也会在 lease 过期后主动结束 `manual-resume -> codex resume` 链。
  手机 Termux 直接安装：
  `curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_s_one_shot.sh | sh`

- `scripts/termux_patch_existing_s_thread.sh`
  当手机本地 `$PREFIX/bin/s` 已经存在、但里面仍然残留旧 thread id 时，用这个脚本直接在手机本地把旧值替换成当前仓库要求的新值。它不依赖手机联网，也不依赖服务器仓库路径。

- `scripts/termux_reinstall_m_offline.sh`
  当手机本地 `$PREFIX/bin/m` 已经存在、但你怀疑它还是旧 launcher、退出不干净或没有带上当前的 watchdog/lease 逻辑时，用这个脚本直接在手机本地重装一份当前仓库标准的 `m` 启动器。它是自包含的离线脚本，不依赖手机联网。

- `scripts/termux_rewrite_m_one_shot.sh`
  在 Termux 上一次性重写 `$PREFIX/bin/m`，让它 SSH 到远端节点、切到 `openclaw`、进入 `/home/openclaw/Babel`，并通过 `manual-resume` 恢复 Babel 的固定专用 Codex 会话。它与 `s` 的形态一致，只是使用另一条固定线程；当前统一复用 `babel-runtime` 仓库内预编译的 `.codex-runtime/bin/babel-issue-bridge` 作为运维入口，避免 Babel 侧工具实现继续漂移。它同样完全基于 `sh` 运行：外层脚本持有本地 watchdog/FIFO，前台子壳负责写 pidfile 后 `exec ssh ...`，同时本地 heartbeat 按固定间隔调用 `touch-manual-lease` 续租；因此即使 `Exit` 没有立刻清掉当前 `ssh`，远端 lease 也会过期并主动结束手动链。
  这个脚本只负责进入 Babel 专用会话，不改变当前 `online` 会话默认聚焦 `babel-runtime` 的约束。
  手机 Termux 直接安装：
  `curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_m_one_shot.sh | sh`

- `scripts/termux_rewrite_probe_one_shot.sh`
  在 Termux 上一次性重写 `$PREFIX/bin/probe`。安装完成后，手机本地可直接执行 `probe clean`、`probe run`、`probe latest`、`probe list`，不用先把仓库同步到手机。
  如果手机本地没有仓库路径，可直接通过 GitHub raw 安装：
  `curl -fsSL https://raw.githubusercontent.com/dengxiaocheng/BabelOnline-GoCpp/main/scripts/termux_rewrite_probe_one_shot.sh | sh`

- `scripts/termux_commands.sh`
  直接打印当前 Termux 侧固定执行命令清单，只列原始脚本命令，不增加包装脚本层。

当前这些 Termux shell 脚本只保留文字解释型注释，不再把可执行命令写进注释里。

- `scripts/termux_manual_remote.sh`
  统一承载 `s / m` 远端动作的 helper。当前两类动作都走它：`manual-resume` 和 `touch-lease`。这样 `s / m` 不必在生成脚本里重复堆叠复杂的远端引号和环境准备逻辑。

- `scripts/termux_check_local_state.sh [host]`
  在 Termux 本地排查这套 `s / m` 启动器留下的状态。它会输出三块信息：当前连向目标节点的本地 `ssh` 客户端、`$TMPDIR/codex-manual/` 下的临时文件，以及当前 `$PREFIX/bin/s` / `$PREFIX/bin/m` 的实际文件状态。默认主机是 `139.159.147.96`。

- `scripts/claudecode_worker_resume.sh`
  在目标工作目录中进入或恢复一个 ClaudeCode worker 会话。它适合执行已经拆好的小任务，不负责项目级管理。

- `scripts/claudecode_worker_packet.sh`
  为指定 worker 生成固定的 `packet.md / report.md`，并把路径写回 `.codex-runtime/claudecode_workers.json`。它还支持 `task-level / max-files / max-delta-lines / read-scope / write-scope / test-command`，让任务拆分标准直接进 packet，而不是停留在口头约束。

- `scripts/claudecode_manager_next.sh`
  让 Codex manager 直接挑出当前最该派发的 worker，并可一键进入对应的 ClaudeCode worker 会话。默认带 `max-running=1` 和同 `lane` 互斥；如果没有现成 session id，但 worker packet 已经存在，也可以直接拉起新的 Claude 会话。

- `scripts/claudecode_manager_start_watcher.sh`
  启动 ClaudeCode 专用 manager watcher。它只常驻轻量 Go watcher；当 worker 关闭 issue 并留下 report 评论后，才用 `codex exec resume` 唤醒一次 Codex manager，处理完成后退出。

- `scripts/claudecode_manager_stop_watcher.sh`
  停止 ClaudeCode 专用 manager watcher，不影响 `s / m` 手动入口。

- `scripts/claudecode_worker_run_once.sh`
  以无人值守方式执行一个 ClaudeCode worker packet。它使用 `claude -p` 跳过交互式 trust prompt，完成后按 worker report 回交给 Codex manager，适合微游戏工厂批量派发。

- `scripts/microgame_factory_create.sh`
  用一条 Babel 创意线直接生成微游戏工厂产物：`creative card`、`mini GDD`、标准 `TASKS.md`，并顺手注册一组 foundation/state/content/ui/integration/qa 的 worker packet。

- `scripts/microgame_factory_start.sh`
  用预制 Babel 创意线一键开始微游戏任务链。它会先生成对应的工厂产物和 worker packets，再以无人值守 `run-once` 方式立即派出当前第一个 Claude worker，并只从当前项目自己的 worker 前缀里挑任务。当前内置预设有：`peigei`、`qidao`、`dianming`。

- `scripts/claudecode_worker_finish.sh`
  让 ClaudeCode worker 通过 `manager-handoff` 把结果回交给 waiting 中的 Codex 管理线程。

- `go run ./cmd/babel-issue-bridge worker-register|worker-packet|worker-next|worker-start|worker-finish|worker-queue|worker-set-status`
  当前最小版的 Claude worker registry / queue。用来登记 worker、生成固定 task packet、写入 `lane`、按并发和 lane 约束自动挑选下一个可派发 worker、查看当前队列、在 worker 完成后把结果挂回 Codex manager，并在 manager 审查后回写最终状态。

- `go run ./cmd/babel-wechatd`
  启动微信测试号服务。除 `BABEL_WECHAT_TOKEN / BABEL_WECHAT_MEMORY_ROOT / BABEL_WECHAT_MAX_RESUME_STEPS` 外，现还支持 `BABEL_SCENE_CORE_LIBRARY`（或兼容别名 `BABEL_CORE_SHARED_LIBRARY`），用于把 `solo_scene / room_scene` 从本地 Go fallback host 切到 Babel/C++ 导出的 shared-library scene host。服务启动时会先对 `.so` 跑一轮标准 fixture 验证，`/healthz` 也会回报当前 `scene_host_mode / source / verified / library / contract`。如果把 `BABEL_SCENE_CORE_LIBRARY` 设为 `@collab`，服务会从 collaboration MCP 的最新 `scene_host_library` artifact 自动解析共享库路径；若未找到 artifact，则启动直接失败，不再静默回退到 Go fallback。

- `go run ./cmd/babel-issue-bridge events --tail 20`
  查看最近的 issue bridge 结构化操作日志。

- `BABEL_ISSUE_BRIDGE_EVENT_HOOK`
  可选的节点本地 hook 命令。每条 issue bridge 事件都会把单条 JSON 通过 stdin 发给它，适合继续接到本地通知或别的运维自动化。

- `go run ./cmd/babel-collab-mcp`
  启动节点级协作 MCP，用来给当前 `online` 会话和 Babel / C++ 专用会话共享结构化协作上下文。它默认把状态写到 `~/.codex-runtime/collab/`，同步的是边界契约、session heartbeat、scope 认领、handoff 和记要，而不是整段聊天历史。

- `go run ./cmd/babel-collab-mcp publish-artifact --session-id babel-cpp --repo Babel --kind scene_host_library --path /abs/path/libbabel_scene_core.so`
  发布结构化产物信息。当前约定下，Babel / C++ 会话可用它发布编好的共享库路径，`online` 侧再通过同一份状态消费。

- `go run ./cmd/babel-collab-mcp snapshot`
  直接查看当前节点级协作状态快照，便于人工排查哪个会话持有了哪些 scope、还有哪些 pending handoff。

- `go run ./cmd/babel-collab-mcp events --tail 20`
  查看最近的协作 MCP 结构化事件日志。

- `go run ./cmd/babel-collab-mcp heartbeat|claim-scope|publish-handoff|ack-handoff ...`
  直接在节点 shell 或 Termux 里管理 `online` / `babel-cpp` 两条会话的结构化协作状态，不需要先准备独立 MCP client。

- `BABEL_COLLAB_EVENT_HOOK`
  可选的节点本地 hook 命令。每条 collaboration MCP 事件都会把单条 JSON 通过 stdin 发给它，适合继续接到本地通知、service 或其他自动化。

- `go run ./cmd/babel-dev check-docs-consistency`
  检查 canonical 文档模块入口和相对链接是否一致，避免文档收束后出现断链。

- `go run ./cmd/babel-dev check-docs-sync-guard`
  根据当前 `git diff` 推导触发的同步矩阵项；如果没有任何对应 canonical 文档一起变化，就直接失败。

- `go run ./cmd/babel-dev check-requirement-assets`
  校验 `requirements/` 下的 registry、schema、asset 路径和跨资产引用，避免 requirement-management foundation 漂移成一堆散落 JSON。

- `go run ./cmd/babel-dev scene-host-fixture --mode solo_step --kind request`
  输出 `SceneHost` 的标准 request/response fixture，供 `babel-cpp` 对齐 `babel_sim_step` ABI，或供当前仓库做 smoke/harness 验证。

- `go run ./cmd/babel-dev verify-scene-host-library --library /path/to/libbabel_scene_core.so --mode all`
  用标准 `solo_step / room_step` fixture 校验共享库 ABI 是否与当前 Go host 契约一致。

- `go run ./cmd/babel-dev render-guard-report .ci-artifacts/guard-status.jsonl`
  把 CI guard 的 jsonl 状态渲染成 markdown 报告，供 step summary 和 PR sticky comment 复用。

- `go run ./cmd/babel-dev install-hooks`
  把 `.githooks/pre-commit` 安装到当前仓库本地配置，在提交前自动运行文档一致性、同步守卫和 requirement asset 校验。

- `.github/workflows/docs-sync-guard.yml`
  在 PR 和 `main` push 上运行同一套文档守卫，防止只在本地 hook 里生效；失败时会上传 `.ci-artifacts/` 日志，并在 PR 上更新一条 sticky comment 报告。

更多运行和运维约定见：

- [docs/OPERATIONS.md](docs/OPERATIONS.md)
