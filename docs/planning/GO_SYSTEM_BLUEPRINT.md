# Babel 新运行时蓝图

## 目标

为 Babel 从零构建一套新的 runtime。

这不是 Python-to-Go 的直接移植。当前 Python bridge 只能作为产品行为参考，而不是架构模板。

新系统应当：

- 支持 solo roleplay 与 multiplayer roleplay
- 在重启后以尽可能小的用户可见中断保持连续性
- 把确定性状态和结算移出 LLM
- 让 Go 成为编排与服务层
- 在未来为高密度 world update / settlement 预留 C++ 仿真核心边界
- 只让 LLM 负责 narration、option wording 与受控解释

## 核心原则

1. Greenfield design
   不继承当前代码布局和内存模式。

2. Deterministic state first
   真实游戏状态在代码和存储里，不在模型上下文里。

3. LLM as presentation layer
   模型可以描述、建议、风格化，但不能成为 progression、inventory、time advancement 或 settlement 的真值来源。

4. Restart-safe by default
   每个活动 session 都必须能从持久化状态与短执行日志中恢复。

5. Fast hot change path
   prompts、balance values、rulesets、schedules、feature flags 应能从配置文件热更新，而不是靠完整重启。

6. Simulation kernel isolation
   高频、多单位更新若需要高性能，应被约束在狭窄而稳定的仿真契约里。

## 高层架构

```text
WeChat / future clients
        |
        v
API Gateway / Session Router (Go)
        |
        +--> Runtime Coordinator / Kernel (Go)
        |        |
        |        +--> Solo Mode / Scene Runtime
        |        +--> Multiplayer / Room Runtime
        |        +--> Recovery Supervisor
        |        +--> Prompt / Policy Loader
        |
        +--> LLM Orchestrator (Go)
        |        |
        |        +--> narration request builder
        |        +--> tool routing
        |        +--> response validator
        |
        +--> Simulation Adapter (Go <-> C++)
        |        |
        |        +--> timeslot stepping
        |        +--> batch entity updates
        |        +--> deterministic settlement
        |
        +--> Persistence Layer (Go)
                 |
                 +--> PostgreSQL or SQLite for state
                 +--> append-only event log
                 +--> blob / json snapshots
```

## 服务拆分

### 1. Gateway

职责：

- 接收用户消息和按钮动作
- 验证 channel event
- 归一化成内部命令
- 立即返回 quick ack
- 异步触发实际执行

说明：

- WeChat-specific 逻辑只应留在这里
- 这一层不应理解 story logic

### 2. Runtime Coordinator / Kernel

职责：

- 定位用户 session 或 room
- 路由到 solo 或 multiplayer runtime
- 获取每个 runtime 的并发锁
- 保证 duplicate delivery 时的 idempotent replay
- 在执行前先持久化 input command

### 3. Solo Engine / Mode

职责：

- 维护 solo session state
- 将玩家 action 应用到确定性游戏状态
- 如有需要，触发 simulation step
- 请求 LLM 生成 scene text 和 candidate options
- 保存最后一个 render frame 以支持快速恢复

### 4. Multiplayer Engine / Room Mode

职责：

- lobby、ready state、join policy、room membership
- stage progression
- per-player hidden / private state
- shared scene state
- synchronized turn close and settlement
- in-progress room 的 restart recovery

### 5. LLM Orchestrator

职责：

- 按 task type 选择 model / policy
- 从状态中构建紧凑、结构化 prompts
- 校验输出 schema
- 拒绝不合格输出并重试
- 支持 prompt pack 热更新

重要规则：

- LLM 绝不能直接写 canonical state

### 6. Simulation Adapter

职责：

- 将 Go 状态转换为 compact simulation input
- 调用 C++ 核心
- 解析确定性输出
- 把 delta 应回 canonical state

### 7. Recovery Supervisor

职责：

- 启动时扫描活跃 session / room
- 恢复未完成动作
- 重新排队被卡住的 scene
- 重建临时 worker
- 在 health endpoint 中暴露恢复状态

## Go 应负责什么

Go 应拥有：

- transport adapters
- API / gateway
- session management
- room management
- action validation
- turn orchestration
- prompt assembly
- LLM calls
- persistence
- config reload
- observability
- recovery logic
- admin tools

Go 不应拥有：

- 如果已经成为性能瓶颈的高密度 per-entity micro-simulation loops
- 更适合 C++ 的大规模 batch settlement math

## C++ 仿真核心

### 目的

为以下场景提供确定性高吞吐核心：

- many units 在 many timeslots 上推进
- resource production / consumption
- work assignment 与 fatigue accumulation
- injury、risk、congestion、morale、environment effects
- event trigger evaluation
- end-of-timeslot 与 end-of-day settlement

### 范围

C++ 核心只负责计算。

它不应知道 WeChat、prompt text 或用户 transport。

### 建议接口

使用窄边界：

```text
Go canonical state
  -> normalized simulation snapshot
  -> C++ step(request)
  -> deterministic result delta
  -> Go applies delta and persists
```

### C++ 输入形状

- world clock
- environment snapshot
- room / session seed
- entity arrays
- work assignments
- inventory / resource tables
- active modifiers
- scripted event flags

### C++ 输出形状

- entity deltas
- environment deltas
- resource deltas
- triggered events
- next seed state
- debug counters

## LLM 边界

LLM 只能负责：

- scene narration
- option wording
- controlled summarization
- bounded interpretation

LLM 不能负责：

- canonical inventory change
- time advancement
- hidden-state ownership
- authoritative settlement

## 持久化要求

持久化层至少需要支持：

- canonical state
- append-only event log
- execution / checkpoint records
- render frames
- delivery records

恢复不应依赖聊天上下文，而应依赖 runtime-controlled persistence。

## 可观测性

系统至少应暴露：

- health status
- active runtime / execution counts
- stale checkpoint counts
- recovery sweep summaries
- per-stage latency
- delivery backlog

## 构建顺序

推荐顺序：

1. state / persistence / recovery
2. solo path
3. world modules
4. multiplayer path
5. profiling
6. C++ extraction

## 反模式

不要这样做：

- 让 agent 成为隐藏状态 authority
- 让 transport 逻辑混进 deterministic core
- 让 recovery 走另一套和正常执行不同的语义
- 让已验证 gameplay 只存在于聊天记录
- 为了“以后可能性能更好”过早上 C++
