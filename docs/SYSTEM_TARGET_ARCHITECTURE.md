# 系统目标架构

本文定义 Babel runtime 的长期目标架构。

它不是对当前 Python bridge 的翻译版，也不是对现有仓库脚手架的默认认可。

它描述的是未来实现应逐步收敛到的系统形态，即使这意味着替换当前仓库里的大量早期结构。

## 目标

构建一个能够做到以下事情的系统：

- 运行单人和多人 AI 驱动的 roleplay session
- 在重启和长时段运行后仍然保持连续性
- 超出单一 agent session 的上下文上限继续扩展
- 把 deterministic state 和 settlement 迁入代码
- 同时支持“以对话为主”的玩法和未来的实时 scene 展示
- 让 `online` 先承担 Godot 可视化交互的文字化替代验证，把场景节奏和玩家决策先压成文本 projection 与按钮命令
- 在 requirement 快速变化时仍不坍缩回 legacy debt
- 通过一条可恢复的统一 runtime lifecycle 执行以上所有行为，而不是堆积一组 per-mode mini-runtime

## 核心原则

系统必须分清：

- canonical state
- agent working memory
- presentation
- requirement management

它们彼此相关，但绝不是一回事。

系统也必须分清：

- execution authority
- mode behavior
- transport concern
- delivery concern

这些维度会频繁交互，但不应被压扁进同一个包边界里。

## 运行模型

系统应围绕三类一等记录运转：

- `runtime instance`
  一个长生命周期的 canonical state aggregate，可对应 session、room、consult thread 等运行单元

- `execution`
  一个已被接受的 inbound action 或 scheduled step，沿着可恢复的 stage machine 前进

- `artifact`
  任何非 canonical 的输出，例如 agent summary、可见文本、scene projection、operational memory file

每个被接受的输入，都应创建或复用一个 execution record。

每个 execution 应在明确 stage 之间流转，例如：

- accepted
- planned
- awaiting_artifacts
- settled
- projected
- committed
- delivered

recovery 必须恢复同一套 stage，而不是额外走一条 recovery-only path。

## 目标分层

### 1. Ingress Adapters

Ingress adapter 是 transport-specific shell。

它们拥有：

- signature verification
- webhook parsing
- request normalization
- transport-specific acknowledgment rule

它们输出 transport-independent 的 `InboundEnvelope`。

Ingress adapter 不拥有：

- runtime state
- gameplay rule
- agent policy
- delivery retry state

### 2. Runtime Kernel

Runtime kernel 是系统编排中心，也是唯一的在线 execution authority。

它拥有：

- execution acceptance
- idempotency
- lease
- runtime instance loading
- execution stage progression
- persistence ordering
- event journaling
- recovery entrypoint
- projection / delivery handoff
- mode module 与支撑子系统之间的协调

Runtime kernel 是 execution 的 canonical source of truth。

它不能被一个长驻 LLM 进程替代。

Kernel 应该知道如何编排工作，但不应包含 mode-specific gameplay rule。

### 3. Mode Router and Mode Modules

Mode routing 决定一个 execution 由哪个 module 负责。

Mode module 至少应包括：

- `free_chat`
- `project_consult`
- `solo_scene`
- `room_scene`

每个 mode module 拥有：

- 输入语法
- route-to-command mapping
- 何时需要 deterministic state 的策略
- 何时需要 agent task 的策略
- 面向用户的 projection policy

Mode module 不拥有：

- transport parsing
- global execution ordering
- durable retry semantics

这套设计替代了“`solo` 和 `multiplayer` 必须成为主编排中心”的假设。

它们应变成运行在同一 kernel 上的 mode behavior。

### 4. Deterministic State Core

`online` 的 Go 运行时不应再把“完整重写 Babel 稳定 deterministic core”当成默认方向。

更准确的目标是：

- Go 负责编排、requirement 验证、projection、delivery 和服务器生命周期
- Babel/C++ 负责稳定世界规则、确定性结算和可复用的核心状态推进
- Go 只在 Babel/C++ 宿主接口尚未准备好时，保留最小验证性 reducer 或过渡性脚手架

当前优先视为 Babel/C++ 所有权的稳定核心包括：

- `time_core` / `WorldClock`
- `population_core` / `NPCDatabase` / `PopulationManager`
- `economy_core` / `EconomySimulator`
- `construction_core` / `ConstructionManager`
- `social_core` / `ConsensusSystem`，以及后续清耦后的 `SocialGraph` / `FamilySystem`
- `narrative_core` / `NarrativeSystem` / `EventPoolManager`
- `settlement_core` / `SimulationManager` / `BatchSimulator`

它拥有：

- time progression
- map progression
- relationship progression
- inventory / resource update
- batch settlement
- world event application
- multiplayer turn resolution

原则：

- deterministic input 产生 deterministic output
- 不允许 LLM 决定 canonical state transition
- 不允许 frontend 拥有 simulation truth
- ruleset version 必须显式且可 reload

Deterministic core 应尽可能纯净、可 replay。

对于 `online` 仓库，这意味着：

- 不再把“补一个 Go 版 world rule”视为默认演进
- 更优先做 Babel/C++ 宿主适配、requirement 资产化和 Godot-facing projection 验证
- 只有在验证阶段必须临时模拟某段规则时，才允许保留 Go 侧过渡实现

### 5. Agent Supervisor

Agent supervisor 是 Go 运行时内部的一个子系统。

它管理：

- task route 到一个或多个 worker
- budget / timeout policy
- artifact collection
- tool authorization policy
- context loading / unloading
- retry 与 recovery
- runtime task 之间的隔离

Agent task 可以是：

- advisory
- interpretive
- narrative
- summarization-oriented
- consult / tool-driven

Agent 输出在被 mode module 或 deterministic reducer 明确接受之前，都只能是 artifact。

长期预期模式：

- 每个玩家可映射到一个 `Claude Code` worker
- 更高层的 `Codex` planner 可用于复杂综合、规划、requirement work
- Go runtime 始终是唯一 supervisor

换句话说：

- Go 编排 agent
- agent 不编排系统

### 6. Operational Memory Artifacts

Operational memory 不是 persistent truth。

它是 agent continuity 的派生缓存。

长生命周期的玩家上下文应被持久化为文件和可加载 artifact，例如：

- `primary_context.md`
- `session_manifest.json`
- `profile.json`
- `state.json`
- `inventory.json`
- `relationship_slice.json`
- `recent_summary.md`
- `long_term_facts.md`
- `open_threads.md`

原则：

- canonical state 存在于 runtime storage
- operational memory artifact 是派生且可重建的
- 单人模式应优先收束为“一个主文档 + 一个长期工作会话”，而不是许多分散小上下文
- 这个长期工作会话可以由 `Claude Code` 承担，但它只拥有 working memory，不拥有 canonical authority
- agent session 可以压缩、重启或被替换
- 这些文件必须允许 runtime 在 agent 重启或上下文压缩后重建可用上下文

Operational memory 应由 runtime 生成并版本化，而不是留在 worker 的私有本地状态里。

### 7. Projection Layer

Projection 把 canonical state 和已批准 artifact 变成用户可见或客户端可见输出。

Projection 输出可以包括：

- WeChat text frame
- operator summary
- structured scene payload
- replay / debug view

Projection 不拥有：

- transport retry
- canonical mutation
- mode resolution

### 8. Delivery Layer

Delivery 拥有 outbound job。

它应管理：

- transport-specific formatting
- retry
- idempotent send bookkeeping
- post-send status reporting

Delivery 必须可以被重跑，而不改变 canonical state。

### 9. Repository and Recovery

Repository 保存权威持久化模型：

- runtime instance
- runtime snapshot
- execution record
- event
- agent task
- artifact
- projection
- delivery job

Recovery 拥有：

- stale execution discovery
- lease expiry handling
- resumable stage selection
- replay / restart inspection

Repository 和 recovery 层都必须是 execution-centric，而不是 handler-centric。

### 10. Presentation Fronts

系统应在同一 kernel 之上支持多个 presentation front。

计划中的 front 包括：

- WeChat 对话界面
- admin / control 界面
- 未来的 web tool
- 未来的 Godot 实时 scene client

Presentation front 不拥有规则。

它们消费或提交的是：

- normalized user intent
- projection frame
- structured scene data

### 11. Requirement Registry and Asset System

系统必须包含专门的 requirement registry。

这个层存在的原因，是 gameplay requirement 仍会持续演化。

它应管理：

- 已验证 mechanic
- 实验中 mechanic
- 被否决的 idea
- ruleset version
- prompt pack version
- content constraint
- release-ready gameplay asset

这不是一个记笔记的 sidecar。

它应逐步变成把探索过的 gameplay idea 转成 versioned、可测试、可加载 runtime asset 的正式来源，供 mode module 和 deterministic core 使用。

## 长期 Agent 模型

成熟形态下，agent model 预期如下：

- Go runtime 控制 orchestration
- 每个玩家对应的 `Claude Code` worker 持有丰富的 task-specific context
- 可选的高层 `Codex` planner 提供综合和监督
- file-backed operational memory 把连续性扩展到单次上下文窗口之外

这样可以避免“一台中央 agent 直接背负所有玩家和房间上下文”的失败模式。

## 与 Godot 的关系

Godot 最终应扮演实时展示与交互客户端。

它应消费：

- structured scene state
- actor state
- map state
- event state
- 适合动画系统消费的 action output

它不应独立拥有 gameplay truth。

Runtime kernel 和 canonical repository 始终保持权威。

这样允许我们：

- 先用文本方式快速迭代
- 再在后续接入更丰富的实时 scene 展示

Godot payload 应通过 projection 导出，而不是在 handler 里 ad hoc 组装。

## 与 C++ 的关系

C++ 是后期优化目标，不是最初的系统中心。

未来更可能提取到 C++ 的部分包括：

- 密集多 actor simulation
- 大规模 time-segment settlement
- 重型 map / path / resource 计算
- 大规模 relationship propagation

边界必须保持狭窄且 deterministic。

Go 继续拥有：

- kernel orchestration
- persistence
- recovery
- transport integration
- agent coordination

## 硬规则

系统必须坚持以下规则：

1. kernel 加 canonical repository 才是唯一 execution truth。
2. LLM agent 可以 assist、summarize、narrate，但不能直接改 canonical state。
3. operational memory artifact 是可重建上下文，不是 simulation truth。
4. Godot 是 client，不是 authority。
5. recovery path 和 normal path 必须复用同一套 state transition logic。
6. requirement management 必须产出 versioned asset，而不是只产出文档。
7. delivery retry 绝不能变成 hidden state transition path。
8. mode behavior 应能被替换，而不改变 kernel ownership boundary。

## 迁移含义

Legacy Python capability 作为产品行为被继承。

Legacy implementation pattern 不作为架构继承。

当前仓库里的早期实现也适用同一条规则。

也就是说：

- 继承 feature
- 干净地重建每一层
- 不把运维 hack 当设计默认值
- 不把当前包名误当永久边界

## 近期构建顺序

接下来的主要实现阶段大致应为：

1. 冻结 execution-centric 的 kernel / repository 契约
2. 实现 kernel stage machine、replay、recovery
3. 重建 free chat / consult / solo scene / room scene 的 mode routing 和 mode module
4. 把 projection 和 delivery 作为独立、可恢复层接入
5. 增加 agent task supervision 和 operational memory artifact
6. 增加 requirement registry 和 asset loading
7. 增加 Godot-facing projection
8. 最后才为 C++ 提取做 profiling

## 成功条件

如果项目满足以下条件，就算成功：

- 产品 feature 保持易于演化
- runtime state 清晰且可调试
- 重启后仍然可恢复
- agent 可被替换而不丢世界
- 系统既能支撑 text-first front，也能支撑未来 real-time front

如果出现以下情况，就算失败：

- agent 变成 hidden state authority
- presentation layer 分叉 simulation truth
- recovery 和 normal execution 分离
- requirement knowledge 只存在于聊天记录里
