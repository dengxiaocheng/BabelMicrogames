# Go 构建计划

## 方向变化

新系统仍然面向“未来可接 C++ 仿真核心”设计，但实现顺序已经明确变化：

1. 先把完整运行时在 Go 中做出来
2. 先把 settlement 和 simulation loop 在 Go 中做正确
3. 只把 C++ 当成后期优化目标

因此第一套可用系统应当是：

- Go gateway
- Go state model
- Go recovery
- Go settlement
- Go time model
- Go map model
- Go relationship model
- LLM render layer

只有当 Go 版本证明规则和性能画像之后，才进入 C++ 阶段。

## 构建原则

1. 先保证产品行为，再谈低层优化
2. 先在 Go 中固化 deterministic rules，再谈 kernel extraction
3. 先把接口稳定下来，再谈 C++ 迁移
4. 每个未来的 C++ 候选模块，都必须先有一个正确的 Go 实现

## Phase 顺序

### Phase 0. Foundation

交付：

- repo scaffold
- config loader
- storage layer
- event log
- health endpoint
- admin inspection commands
- recovery checkpoint framework

目标：

- 服务可启动
- config hot reload 可用
- checkpoint model 存在

### Phase 1. Solo Runtime in Go

交付：

- canonical solo state
- solo action intake
- deterministic Go settlement step
- time segment advancement
- LLM scene render
- render cache
- restart-safe solo recovery

目标：

- 单用户可以端到端完成 solo play
- 重启不会打断连续性

### Phase 2. Core World Modules in Go

将以下模块作为一等 Go 模块实现：

1. time module
2. map module
3. relationship module

#### Time module

负责：

- day / segment progression
- action windows
- per-segment modifiers
- scheduling

#### Map module

负责：

- zone graph
- travel cost
- worksite topology
- co-presence / adjacency
- local hazards / resources

#### Relationship module

负责：

- actor graph
- trust / fear / affinity / debt
- public / private visibility
- event-driven relationship delta

目标：

- solo runtime 不再依赖 ad hoc scene 假设
- 状态变化来自结构化 world modules

### Phase 3. Multiplayer Runtime in Go

交付：

- lobby model
- room state
- synchronized turn close
- shared + private visible state
- Go settlement for multi-actor segments
- multiplayer recovery

目标：

- room-based play 可以在 Go 中稳定运行

### Phase 4. Large-Scale Go Settlement

交付：

- Go 中的 batch entity stepping
- aggregation passes
- profiling instrumentation
- many-actor / many-segment stress tests

目标：

- 在动 C++ 之前先知道真正的热点在哪里

重要说明：

未来哪些东西适合抽成 C++，是在这个阶段识别，而不是更早。

### Phase 5. C++ Candidate Extraction

只在 profiling 之后进行。

可能的候选：

- large batch unit stepping
- dense resource settlement
- hazard propagation
- graph-heavy map / proximity calculations

前提：

- 候选功能已经有正确的 Go 版本
- Go / C++ 契约从已验证行为中抽取，而不是靠猜

### Phase 6. Hybrid Runtime

交付：

- Go 继续作为 canonical orchestrator
- 只把选定热点移到 Go adapter 背后
- Go 与 C++ 路径之间有 A/B validation
- deterministic parity checks 持续存在

## 模块优先级

应严格遵守以下顺序：

1. state / store / recovery
2. time module in Go
3. map module in Go
4. relationship module in Go
5. solo settlement in Go
6. multiplayer settlement in Go
7. profiling
8. C++ extraction

## 为什么必须 Go First

当前最大风险不是 CPU 成本，而是：

- 规则不清晰
- state ownership 不清晰
- recovery 语义脆弱

Go-first 先解决：

- canonical state ownership
- restart continuity
- correct deterministic rules
- module boundaries
- observability

然后 C++ 再去解决：

- throughput
- latency at scale
- dense simulation cost

## 建议优先构建的包

```text
internal/config
internal/store
internal/eventlog
internal/recovery
internal/timecore
internal/mapcore
internal/relcore
internal/solo
internal/render
internal/llm
internal/admin
```

之后再进入：

```text
internal/multiplayer
internal/settlement
internal/gateway
```
