# Runtime 执行序列

## 目的

定义新系统中 solo 与 multiplayer turn 的执行流。

这些序列既是实现依据，也是测试依据。

## Solo 序列

### 正常路径

1. gateway 接收用户输入
2. coordinator / kernel 加载 solo session
3. coordinator / kernel 写入 input event
4. 获取 solo runtime lease
5. 创建 checkpoint，状态为 `validating`
6. 将输入归一化成合法 action
7. checkpoint 进入 `simulating`
8. Go settlement pipeline 执行一个 segment step
9. canonical state 提交
10. checkpoint 进入 `rendering`
11. 根据 visible consequences 构建 LLM render request
12. render frame 持久化
13. checkpoint 进入 `delivering`
14. 尝试投递
15. checkpoint 进入 `committed`

### 恢复路径

如果中途重启：

- settlement commit 之前：从旧 state version 重跑
- settlement commit 之后但 render 之前：rerender
- render 之后但 delivery 之前：redeliver 已保存 frame

## Multiplayer 序列

### Stage Open

1. coordinator / kernel 加载 room
2. 验证 room stage 当前可打开
3. 加载 roster 和 required players
4. 如有需要，渲染当前 shared / private frame

### Player Submission

1. 接收玩家 action
2. 持久化 input event
3. 验证 membership 和 stage
4. 将 action 写入 `pending_turn`
5. 返回 submission ack

### Turn Close

1. 所有 required actions 到齐，或 deadline 已到
2. 获取 room lease
3. checkpoint 进入 `simulating`
4. Go settlement 为该 room 执行一个 segment
5. canonical room state 提交
6. checkpoint 进入 `rendering`
7. 渲染 shared scene
8. 渲染每个玩家的 private frame
9. 持久化所有 frame
10. checkpoint 进入 `delivering`
11. 投递输出
12. checkpoint 进入 `committed`

### Multiplayer 恢复路径

如果中途重启：

- 已提交的 action 仍保留在 `pending_turn`
- 若 room 尚未 settlement，则从旧 state version 重跑
- 若 state 已 settled 但 private / public frame 缺失，则 rerender
- 若 frame 已存在但 delivery 未完成，则 redeliver

## Segment 所有权

每个 time segment 只能有一个 committed settlement result。

绝不允许：

- 同一 segment 被重复 settlement
- 只对部分玩家完成 canonical commit
- 在没有 committed state version 的前提下产生 narrative output

## Checkpoint 映射

step name 应保持一致，例如：

- `solo.validate`
- `solo.simulate`
- `solo.render`
- `solo.deliver`
- `room.collect`
- `room.simulate`
- `room.render.shared`
- `room.render.private`
- `room.deliver`
