# Go 模块设计

## 范围

本文定义在任何 C++ 提取之前必须先存在的 Go 原生模块。

这些模块不是临时替身，而是第一版正确实现。未来如果要做 C++，也只是把已经证明是热点的局部能力替换到稳定接口后面。

## 优先模块

1. time module
2. map module
3. relationship module
4. settlement pipeline

## 1. Time Module

建议包名：

```text
internal/timecore
```

### 职责

- 定义时间片
- 验证每个时间片允许哪些 action
- 推进 world clock
- 应用每个 segment 的默认 modifier
- 向 settlement / render 层暴露 schedule metadata

### 核心类型

```yaml
WorldClock:
  day_index: int
  segment: enum(dawn, morning, noon, afternoon, dusk, night)
  absolute_tick: int

SegmentRule:
  segment: string
  allowed_action_tags: [string]
  default_modifiers: [Modifier]
  next_segment: string
```

### 对外接口

```text
CurrentSegment(clock) -> SegmentRule
ValidateAction(segment, action) -> bool
Advance(clock) -> nextClock
ApplySegmentDefaults(state, segment) -> stateDelta
```

### 说明

- 它不应知道 transport 或 prompts
- 它应保持纯函数、确定性

## 2. Map Module

建议包名：

```text
internal/mapcore
```

### 职责

- 定义 zone 与 subzone
- adjacency 与 travel cost
- worksite topology
- co-location 与 line-of-effect 规则
- local hazards 与 resources

### 核心类型

```yaml
Zone:
  zone_id: string
  zone_type: string
  tags: [string]
  neighbors: [Edge]
  hazard_profile: HazardProfile
  resource_profile: ResourceProfile

Edge:
  target_zone_id: string
  traversal_cost: int
  traversal_tags: [string]
```

### 对外接口

```text
IsAdjacent(a, b) -> bool
TravelCost(a, b) -> int
VisiblePeers(zone, entities) -> [entity]
ZoneModifiers(zone) -> [Modifier]
```

### 说明

- multiplayer 的 shared-scene consistency 应依赖 mapcore，而不是 prompt trick
- 未来若做 C++ 提取，通常也只是优化 graph / proximity 计算，而不是让规则定义脱离 Go

## 3. Relationship Module

建议包名：

```text
internal/relcore
```

### 职责

- actor-to-actor relationship graph
- trust / affinity / fear / debt
- public / private visibility
- event-driven relationship update

### 核心类型

```yaml
RelationshipEdge:
  source_actor_id: string
  target_actor_id: string
  trust: int
  affinity: int
  fear: int
  debt: int
  tags: [string]

RelationshipDelta:
  source_actor_id: string
  target_actor_id: string
  trust_delta: int
  affinity_delta: int
  fear_delta: int
  debt_delta: int
```

### 对外接口

```text
ApplyDelta(graph, delta) -> graph
VisibleRelationshipSlice(graph, actor_id) -> slice
RelationshipModifiers(graph, actor_id) -> [Modifier]
```

### 说明

- 它必须同时支持 solo NPC 关系和 multiplayer player-to-player 边
- 默认情况下，LLM 只看到被选择过的切片，而不是整个 graph

## 4. Settlement Pipeline

建议包名：

```text
internal/settlement
```

### 职责

- 收集已经验证的 action
- 读取 timecore / mapcore / relcore
- 计算确定性的 state delta
- 向 render layer 输出结构化 consequences

### Pipeline 阶段

1. freeze input state
2. normalize actions
3. apply time segment rules
4. apply map / locality effects
5. apply relationship effects
6. compute resource / stat changes
7. generate triggered event set
8. commit canonical deltas

### 对外接口

```text
StepSolo(state, action, deps) -> StepResult
StepMultiplayer(roomState, actions, deps) -> StepResult
```

其中 `deps` 包含：

- timecore
- mapcore
- relcore
- rule tables

## Render 边界

Go 模块输出的是结构化后果，而不是文本。

例如：

```yaml
VisibleConsequence:
  tag: "fatigue_spike"
  severity: 2
  text_hint: "午后的疲惫明显压上来"
```

然后由 render layer 把它们转换成 LLM 输入。

## Storage 边界

这些模块都应当保持 persistence-agnostic。

它们操作的是 orchestrator 传入的内存结构，不自己做数据库或 event log 操作。

持久化属于：

- repository / store
- execution coordinator / kernel
- recovery / replay

## 测试要求

每个模块都应优先具备：

- deterministic unit tests
- edge case coverage
- replay-oriented fixture tests

不要让世界规则的正确性依赖 transport 集成测试来证明。
