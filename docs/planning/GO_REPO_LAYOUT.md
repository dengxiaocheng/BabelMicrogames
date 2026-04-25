# Go 仓库布局

## 目标

为新的 Babel runtime 定义一个干净的仓库布局。

这个仓库只服务于新系统，不应去镜像旧 Python 的文件形状。

## 布局草案

```text
babel-runtime/
  cmd/
    gateway/
      main.go
    adminctl/
      main.go

  internal/
    gateway/
    coordinator/
    config/
    store/
    eventlog/
    recovery/
    admin/
    llm/
    render/

    core/
      types/
      rules/
      ids/

    timecore/
    mapcore/
    relcore/
    settlement/

    solo/
    multiplayer/

    testkit/
      fixtures/
      golden/
      builders/
      fakeclock/
      fakellm/
      fakestore/

  schemas/
    state/
    llm/
    events/

  configs/
    prompt-packs/
    rulesets/
    maps/
    balance/
    feature-flags/

  tests/
    integration/
    e2e/
    stress/

  docs/
    architecture/
    operations/
    testing/

  cpp/
    sim_core/
      include/
      src/
      tests/
```

## 包职责

### `cmd/gateway`

- 生产入口
- 启动 HTTP server
- 连接 config、store、recovery 和 runtime services

### `cmd/adminctl`

- 离线或运维工具入口
- 检查 session
- 检查 checkpoint
- 触发 replay 或 recovery 动作

### `internal/gateway`

- channel adapters
- request normalization
- quick ack behavior

### `internal/coordinator`

- runtime routing
- lock acquisition
- idempotency
- handoff 到 solo / multiplayer flow

### `internal/config`

- hot config loading
- config versioning
- prompt pack resolution
- feature flags

### `internal/store`

- canonical persistence APIs
- state read / write
- transaction boundary

### `internal/eventlog`

- append-only runtime event recording
- replay support

### `internal/recovery`

- checkpoint scanning
- lease reclaim
- resume execution

### `internal/core`

共享的底层运行时定义：

- ids
- canonical types
- rule constants

### `internal/timecore`

- time segment definitions
- schedule rules
- time advancement

### `internal/mapcore`

- zone topology
- adjacency
- traversal / co-location rules

### `internal/relcore`

- relationship graph rules
- visibility slices
- relationship deltas

### `internal/settlement`

- deterministic rules pipeline
- Go-first batch state settlement

### `internal/solo`

- solo runtime orchestration
- action application
- render assembly

### `internal/multiplayer`

- room model
- synchronized turn progression
- shared / private outputs

### `internal/llm`

- request building
- provider abstraction
- schema validation
- retries and fallbacks

### `internal/render`

- render frame creation
- delivery payload assembly

### `internal/testkit`

- 可复用测试 harness
- fake implementations
- state / room fixture builders

## 边界规则

1. `solo` 和 `multiplayer` 可以依赖：
   - `core`
   - `timecore`
   - `mapcore`
   - `relcore`
   - `settlement`
   - `store`
   - `eventlog`
   - `llm`
   - `render`

2. `timecore`、`mapcore`、`relcore`、`settlement` 不应依赖：
   - transport
   - HTTP
   - database drivers
   - real LLM clients

3. `testkit` 可以依赖任意 internal package，但生产包不应依赖 `tests/`

## 测试放置方式

建议三层：

- 单元测试放在包旁边
- 可复用 harness 放在 `internal/testkit`
- 更高层集成和压力测试放在 `tests/`
