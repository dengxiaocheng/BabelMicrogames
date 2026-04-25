# Canonical 状态模型草案

## 目的

为新的 Babel 系统定义 source-of-truth runtime state。

这个 schema 只面向新系统，不镜像旧 Python 的存储布局。

## 设计规则

1. Canonical state 必须显式存在
2. 每个持久化结构都必须带版本
3. 时间推进必须能在不依赖模型记忆的前提下表达
4. Hidden state 与 visible state 必须分开
5. Render cache 是派生物，不是 canonical truth

## 顶层对象

### User

```yaml
User:
  user_id: string
  channel: string
  channel_user_id: string
  profile:
    display_name: string
    locale: string
  created_at: timestamp
  updated_at: timestamp
```

### SoloSession

```yaml
SoloSession:
  session_id: string
  user_id: string
  schema_version: int
  ruleset_id: string
  prompt_pack_id: string
  status: enum(active, paused, archived)
  chapter_id: string
  world_clock:
    day_index: int
    segment: enum(dawn, morning, noon, afternoon, dusk, night)
    absolute_tick: int
  seed_state:
    world_seed: uint64
    rng_counter: uint64
  player_state: PlayerState
  environment_state: EnvironmentState
  relationship_state: RelationshipState
  event_flags: map[string]FlagValue
  pending_action: PendingAction|null
  last_committed_action_id: string|null
  last_render_frame_id: string|null
  created_at: timestamp
  updated_at: timestamp
```

### MultiplayerRoom

```yaml
MultiplayerRoom:
  room_id: string
  schema_version: int
  ruleset_id: string
  prompt_pack_id: string
  status: enum(lobby, active, paused, recovering, archived)
  chapter_id: string
  world_clock:
    day_index: int
    segment: enum(dawn, morning, noon, afternoon, dusk, night)
    absolute_tick: int
  seed_state:
    world_seed: uint64
    rng_counter: uint64
  roster: [RoomMember]
  shared_state: SharedSceneState
  hidden_state: map[player_id]PrivateState
  event_flags: map[string]FlagValue
  pending_turn: PendingTurn|null
  last_committed_turn_id: string|null
  last_render_frame_ids: map[player_id]string
  created_at: timestamp
  updated_at: timestamp
```

## 核心子结构

### PlayerState

```yaml
PlayerState:
  actor_id: string
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
    fishing: int
    crafting: int
  traits:
    obedience: int
    ambition: int
    homesickness: int
    loyalty: int
    risk_tolerance: int
  role:
    job_id: string
    job_level: int
  inventory:
    slots: [InventoryEntry]
    capacity: int
  conditions:
    active: [Condition]
  location:
    zone_id: string
    subzone_id: string
  meta:
    days_worked: int
```

### EnvironmentState

```yaml
EnvironmentState:
  site_id: string
  tower_height_level: int
  weather_id: string
  hazard_level: int
  food_supply_level: int
  supervision_level: int
  morale_pressure: int
  active_modifiers: [Modifier]
```

### RelationshipState

```yaml
RelationshipState:
  edges: [RelationshipEdge]

RelationshipEdge:
  source_actor_id: string
  target_actor_id: string
  affinity: int
  trust: int
  fear: int
  debt: int
  tags: [string]
```

### SharedSceneState

```yaml
SharedSceneState:
  zone_id: string
  stage_id: string
  worker_group_ids: [string]
  public_hazards: [string]
  public_events: [string]
  aggregate_metrics:
    fatigue_pressure: int
    injury_risk: int
    work_pressure: int
```

### PrivateState

```yaml
PrivateState:
  actor_id: string
  hidden_flags: map[string]FlagValue
  private_observations: [string]
  private_inventory_tags: [string]
  secret_objectives: [Objective]
```

## Inventory Schema

```yaml
InventoryEntry:
  item_id: string
  quantity: int
  durability: int|null
  quality: int|null
  tags: [string]
```

## Action Schema

### Solo PendingAction

```yaml
PendingAction:
  action_id: string
  user_input: string
  parsed_action:
    action_type: string
    target_ids: [string]
    parameters: map[string]any
  received_at: timestamp
  idempotency_key: string
```

### Multiplayer PendingTurn

```yaml
PendingTurn:
  turn_id: string
  segment: string
  required_players: [string]
  submissions: [ResolvedAction]
  close_deadline: timestamp|null
```

## Flag 与条件

建议所有 flag / condition 都显式版本化，而不是让 prompt 暗含它们的意义。

```yaml
FlagValue:
  type: enum(bool, int, string)
  bool_value: bool|null
  int_value: int|null
  string_value: string|null

Condition:
  condition_id: string
  source_id: string|null
  stacks: int
  expires_at_tick: int|null
  tags: [string]
```

## Render 与派生状态

以下内容不应进入 canonical truth：

- narration text
- option label wording
- summary markdown
- transport-specific payload

它们都属于派生输出，只应通过 frame / projection 保存。

## 版本迁移要求

- 顶层对象必须带 `schema_version`
- 子结构如有独立生命周期，也应允许单独版本化
- 迁移应是显式步骤，不应靠 prompt 隐式修复

## 核心约束

- 每次 committed execution 都必须对应清晰的 state version 变化
- hidden state 不得通过 public render 泄漏
- seed / clock / pending intent 必须可恢复
- 同一个 segment 不允许出现多个 canonical settlement 结果
