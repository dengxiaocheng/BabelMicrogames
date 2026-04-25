# 持久化与事件日志布局

## 目标

定义新系统的持久化模型，覆盖：

- canonical state
- append-only events
- recovery checkpoints
- render cache

## 设计规则

1. State 是 canonical truth
2. Event 是 append-only 历史记录
3. Checkpoint 跟踪执行中间态
4. Render frame 是面向用户的派生缓存

## 存储类别

### 1. Canonical State Tables

保存当前真实状态：

- users
- solo_sessions
- multiplayer_rooms
- room_members
- actor_state
- inventory_state
- relationship_edges
- environment_state

### 2. Event Log

保存历史序列：

- input received
- action accepted
- turn closed
- settlement completed
- render completed
- delivery attempted

### 3. Checkpoints

保存进行中的恢复状态：

- active runtime step
- current execution stage
- resume policy

### 4. Render Frames

保存最近的用户可见输出：

- scene text
- option labels
- private / public variants

## 建议表结构

### `solo_sessions`

关键字段：

- `session_id`
- `user_id`
- `status`
- `schema_version`
- `state_version`
- `chapter_id`
- `world_clock_json`
- `seed_state_json`
- `last_render_frame_id`

### `multiplayer_rooms`

关键字段：

- `room_id`
- `status`
- `schema_version`
- `state_version`
- `chapter_id`
- `world_clock_json`
- `seed_state_json`
- `last_committed_turn_id`

### `room_members`

关键字段：

- `room_id`
- `player_id`
- `join_status`
- `ready_status`
- `role_json`
- `private_state_json`

### `actor_state`

关键字段：

- `runtime_type`
- `runtime_id`
- `actor_id`
- `state_json`

### `relationship_edges`

关键字段：

- `runtime_type`
- `runtime_id`
- `source_actor_id`
- `target_actor_id`
- `edge_json`

### `runtime_events`

关键字段：

- `event_id`
- `runtime_type`
- `runtime_id`
- `event_type`
- `idempotency_key`
- `payload_json`
- `created_at`

### `runtime_checkpoints`

关键字段：

- `checkpoint_id`
- `runtime_type`
- `runtime_id`
- `input_event_id`
- `status`
- `step_name`
- `resume_policy`
- `state_version_before`
- `state_version_after`
- `lease_owner`
- `lease_expires_at`

### `render_frames`

关键字段：

- `frame_id`
- `owner_type`
- `owner_id`
- `state_version`
- `frame_json`
- `created_at`

### `delivery_records`

关键字段：

- `delivery_id`
- `checkpoint_id`
- `frame_id`
- `recipient_id`
- `channel`
- `status`

## 事务模式

### 执行之前

单事务：

1. 插入 `runtime_events`
2. 创建或更新 `runtime_checkpoints`
3. 获取或续租 runtime lease

### 确定性结算之后

单事务：

1. 更新 canonical state tables
2. 更新 `state_version`
3. 追加 settlement event
4. 将 checkpoint 更新到 `rendering`

### 渲染之后

单事务：

1. 写入 render frame
2. 追加 render event
3. 将 checkpoint 更新到 `delivering`

### 投递之后

单事务：

1. 更新 delivery record
2. 追加 delivery event
3. 将 checkpoint 标记为 `committed`

## Replay 用途

Replay 应读取：

- canonical state baseline
- 有序的 `runtime_events`
- 可选 checkpoints

Replay 的用途是：

- debugging
- deterministic validation
- recovery tooling
- test fixtures

## Compaction 策略

- canonical state 始终保存当前真值
- event log 保持 append-only，不做语义改写
- checkpoints 可在进入终态后归档或清理
- render frames 可按保留策略压缩，只保留最近 N 个或关键版本

## 索引建议

至少应为以下查询提供索引：

- `runtime_id + created_at`
- `runtime_id + event_type`
- `checkpoint status + updated_at`
- `delivery status + recipient_id`

## 边界说明

- canonical truth 不应藏在 render frame 中
- checkpoint 不应替代 canonical state
- event log 不应承担当前状态读取责任
- transport-specific payload 不应污染世界状态表
