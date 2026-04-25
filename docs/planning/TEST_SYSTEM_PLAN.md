# 测试系统计划

## 目标

在新运行时建设过程中，同时构建一套一等公民的测试系统。

测试不是后置 QA 层，而是架构的一部分。

## 测试目标

测试系统必须验证：

- deterministic settlement 正确性
- restart recovery 正确性
- state transition 合法性
- render schema 合法性
- multiplayer synchronization
- config hot reload behavior

## 测试层次

### 1. Unit Tests

目标：

- `timecore`
- `mapcore`
- `relcore`
- `settlement`
- `config`
- `recovery`

目的：

- 验证纯逻辑
- 提供快速反馈
- 覆盖边界条件

### 2. Integration Tests

目标：

- store + event log + recovery
- coordinator + solo runtime
- coordinator + multiplayer runtime
- LLM orchestration with fake provider

目的：

- 验证模块接线
- 验证事务与 checkpoint 流

### 3. End-to-End Tests

目标：

- gateway 到 delivery
- solo play continuity
- multiplayer room progression
- active operation 中途重启

目的：

- 验证产品级正确性

### 4. Stress Tests

目标：

- many rooms
- many entities
- repeated segment steps
- delivery backlog

目的：

- 在 C++ 提取前识别热点

## 必要测试基础设施

### Fake Clock

用于：

- deterministic time advancement
- lease expiry tests
- recovery timing

### Fake LLM

用于：

- schema-valid narrative response
- malformed output scenario
- retry path coverage

### Fake Store

用于：

- 无数据库成本的逻辑级测试

### Real DB Integration Harness

用于：

- checkpoint 与 transaction correctness
- replay correctness

## Golden Tests

应维护以下 golden fixture：

- solo settlement outputs
- multiplayer turn outputs
- recovery decisions
- render request / response validation

golden tests 应包含：

- canonical state snapshot
- input action set
- expected state delta
- expected visible consequences

## Recovery 测试矩阵

必须覆盖每个 crash point：

1. input persistence 之后
2. simulation 过程中
3. state commit 之后
4. render 过程中
5. frame persistence 之后
6. delivery 过程中

每个 crash point 都要验证：

- 不发生 duplicate settlement
- action 不丢
- resumed / redelivered output 正确

## Multiplayer 测试矩阵

必须覆盖：

- 缺少一个玩家 action
- duplicate player submission
- mid-turn restart
- shared render 之后、private render 之前重启
- private render 之后、delivery 之前重启
- room reopen after restart

## Property Tests

建议增加 property-based tests，验证：

- 相同 seed 下 deterministic settlement 恒定
- inventory 不会低于允许下限
- 时间不会非法倒退
- adjacency 与 traversal invariant
- 适用场景下的 relationship symmetry

## Performance Tests

在任何 C++ 工作开始前，都应 benchmark Go settlement 在以下场景下的表现：

- 一个 room 内很多 actor
- 并行很多 room
- 顺序执行很多 segment

应输出：

- CPU cost per segment
- memory allocation profile
- top hot functions

只有 profiling 结果才能决定是否需要 C++ 提取。

## 测试目录草案

```text
internal/testkit/
  builders/
  fixtures/
  fakeclock/
  fakellm/
  fakestore/
  restartsim/

tests/
  integration/
  e2e/
  stress/
```

## 最小早期测试集

在大规模实现前，至少应具备：

1. timecore segment advance tests
2. map adjacency tests
3. relationship delta tests
4. solo settlement deterministic tests
5. checkpoint recovery tests
6. `/healthz` 与 config reload tests

## 运维信心测试

还应有一小套偏生产信心的测试：

- 启动服务
- 加载配置
- 创建 solo session
- 结算一次 action
- 重启服务
- 继续同一个 session
