# C++ 仿真契约

## 目的

本文定义 Go 编排层与未来 C++ 仿真核心之间应保持的狭窄边界。

原则很简单：

- C++ 负责确定性计算
- Go 负责编排、持久化、恢复和用户侧流程

## 契约规则

1. 一次调用代表一次明确的 simulation step
2. 输入必须完全显式
3. 相同输入和种子必须得到相同输出
4. 不允许隐藏的全局可变状态
5. C++ 不做文本生成

## Step 粒度

推荐的初始粒度：

- 一个 room 或一个 solo session
- 一个离散时间片
- 一批已经通过验证的 action

例如：

- solo session，第 4 天，上午时段
- 玩家 action 已经被接受
- 执行一次确定性的 segment settlement

## 输入包络

```yaml
SimulationStepRequest:
  contract_version: int
  runtime_type: enum(solo, multiplayer)
  runtime_id: string
  ruleset_id: string
  chapter_id: string
  world_clock:
    day_index: int
    segment: string
    absolute_tick: int
  seed:
    world_seed: uint64
    rng_counter: uint64
  environment: EnvironmentSnapshot
  entities: [EntitySnapshot]
  assignments: [WorkAssignment]
  actions: [ResolvedAction]
  modifiers: [ModifierSnapshot]
  flags: [FlagSnapshot]
```

## 核心输入结构

### EntitySnapshot

```yaml
EntitySnapshot:
  entity_id: string
  entity_type: enum(player, npc, worker_group, supervisor, resource_node)
  location_id: string
  stats:
    stamina: int
    spirit: int
    satiety: int
    health: int
    stress: int
  skills:
    hauling: int
    masonry: int
    ropework: int
    crafting: int
  traits:
    obedience: int
    ambition: int
    risk_tolerance: int
  inventory_refs: [InventoryRef]
  condition_ids: [string]
```

### WorkAssignment

```yaml
WorkAssignment:
  assignment_id: string
  entity_id: string
  task_type: string
  target_id: string|null
  effort_level: int
  tool_item_id: string|null
```

### ResolvedAction

```yaml
ResolvedAction:
  action_id: string
  actor_id: string
  action_type: string
  target_ids: [string]
  parameters: map[string]scalar
```

### ModifierSnapshot

```yaml
ModifierSnapshot:
  modifier_id: string
  scope: enum(entity, room, environment)
  target_id: string|null
  effect_type: string
  magnitude: int
```

## 输出包络

```yaml
SimulationStepResult:
  contract_version: int
  runtime_id: string
  accepted: bool
  next_rng_counter: uint64
  entity_deltas: [EntityDelta]
  environment_delta: EnvironmentDelta
  resource_deltas: [ResourceDelta]
  triggered_events: [TriggeredEvent]
  aggregate_metrics: AggregateMetrics
  debug_counters: DebugCounters
```

## Delta 结构

### EntityDelta

```yaml
EntityDelta:
  entity_id: string
  stat_changes:
    stamina: int
    spirit: int
    satiety: int
    health: int
    stress: int
  skill_changes: map[string]int
  inventory_changes: [InventoryDelta]
  added_conditions: [string]
  removed_conditions: [string]
  relationship_changes: [RelationshipDelta]
  location_change: string|null
```

### ResourceDelta

```yaml
ResourceDelta:
  resource_id: string
  amount_delta: int
```

### TriggeredEvent

```yaml
TriggeredEvent:
  event_id: string
  visibility: enum(public, private, hidden)
  owner_id: string|null
  severity: int
  tags: [string]
  payload: map[string]scalar
```

## Go 在调用 C++ 之前必须完成的事

Go 必须先：

- 验证 action 合法性
- 将用户输入解析为合法 action 类型
- 冻结本次 step 的实体集合
- 冻结本次 step 的时间片
- 提供确定性的 seed 状态

这样 C++ 核心就不需要处理 transport 层歧义。

## Go 在收到结果之后必须完成的事

Go 必须：

- 把 delta 应用回 canonical state
- 持久化 checkpoint
- 派生可见状态切片
- 调用 LLM 做 narration / render
- 持久化 render frame

## 确定性要求

只要以下输入相同：

- input snapshot
- ruleset version
- seed

C++ 核心就必须返回相同结果。

明确禁止：

- 依赖 wall clock
- 使用隐式随机源
- 访问外部 I/O

## 建议 ABI

近期推荐方式：

- 从 C++ 暴露 C ABI
- Go 通过 cgo 调用

建议 API 形状：

```c
int babel_sim_step(
  const uint8_t* request_bytes,
  size_t request_len,
  uint8_t** response_bytes,
  size_t* response_len
);

void babel_sim_free(void* ptr);
```

## 版本管理

- request / response 都应携带 `contract_version`
- Go 侧 adapter 负责版本协商
- 不允许 Go 和 C++ 通过隐式结构布局耦合

## 验证要求

任何进入 C++ 的候选路径，在 Go 版本上都必须先满足：

- 规则正确
- 回放正确
- 测试稳定
- profiling 证明它确实是热点

否则不要提取。
