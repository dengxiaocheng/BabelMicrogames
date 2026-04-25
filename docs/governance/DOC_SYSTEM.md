# 文档体系

本文定义 canonical 文档之间的关系，以及在不同任务下应该如何按需读取。

对应的机器可读定义见：

- [DOC_MANIFEST.json](./DOC_MANIFEST.json)

这份 manifest 用来承载：

- canonical 顶层文档集合
- 模块入口与模块文档集合
- topic 路由
- 最小同步矩阵
- 实现前守卫的路径触发规则

本页负责解释它，检查脚本负责验证它。

## 目标

文档体系需要同时满足：

- 能表达稳定系统理解
- 能支撑实现与运维
- 能降低漂移
- 能支持按需读取，而不是每轮全量通读

## Canonical 模块

当前 canonical 文档按四个专业模块组织：

- [../architecture/README.md](../architecture/README.md)
  系统能力、当前架构、目标架构、路线图

- [../runtime/README.md](../runtime/README.md)
  边界、接口、存储、测试

- [./README.md](./README.md)
  文档治理、incoming 处理、同步 gate、变更台账

- [../operations/README.md](../operations/README.md)
  节点运维、watcher、手动接管、hook

## 文档关系

核心关系如下：

- `FEATURE_INHERITANCE.md`
  定义能力范围和继承原则

- `ARCHITECTURE.md`
  定义当前实现基线

- `SYSTEM_TARGET_ARCHITECTURE.md`
  定义长期目标系统形态

- `ROADMAP.md`
  定义阶段顺序和近期队列

- `SUBSYSTEM_BOUNDARIES.md`
  定义所有权边界

- `INTERFACES.md`
  定义代码接口和模块契约

- `STORAGE_SCHEMA.md`
  定义权威持久化模型

- `TESTING.md`
  定义验证义务

- `OPERATIONS.md`
  定义运维模块入口；细节下沉到 `docs/operations/`

- `DESIGN_SYNC_WORKFLOW.md`
  定义文档治理模块入口；细节下沉到 `docs/governance/`

- `REQUIREMENT_SYNC_CHECKLIST.md`
  定义 intake gate

- `REQUIREMENT_CHANGELOG.md`
  定义变化台账

## 最小同步矩阵

当变化发生时，至少应检查这些文档：

- mode / gameplay capability 变化：
  `FEATURE_INHERITANCE.md`、`ROADMAP.md`、`TESTING.md`

- gateway contract 变化：
  `FEATURE_INHERITANCE.md`、`INTERFACES.md`、`TESTING.md`

- kernel / execution flow 变化：
  `ARCHITECTURE.md`、`INTERFACES.md`、`STORAGE_SCHEMA.md`、`TESTING.md`

- repository / recovery / eventlog 变化：
  `STORAGE_SCHEMA.md`、`INTERFACES.md`、`TESTING.md`

- agent / projection / delivery 变化：
  `ARCHITECTURE.md`、`FEATURE_INHERITANCE.md`、`TESTING.md`

- testkit / validation harness 变化：
  `TESTING.md`、`ARCHITECTURE.md`

- requirement registry foundation 变化：
  `ARCHITECTURE.md`、`ROADMAP.md`、`SUBSYSTEM_BOUNDARIES.md`、`INTERFACES.md`、`STORAGE_SCHEMA.md`、`TESTING.md`

- requirement asset 内容变化：
  `FEATURE_INHERITANCE.md`、`TESTING.md`、`REQUIREMENT_CHANGELOG.md`

- 其它 runtime 结构变化：
  `ARCHITECTURE.md`、`SUBSYSTEM_BOUNDARIES.md`、`INTERFACES.md`

- 长期系统方向变化：
  `SYSTEM_TARGET_ARCHITECTURE.md`、`ROADMAP.md`、`SUBSYSTEM_BOUNDARIES.md`

- 运维 / watcher / hook / manual takeover / CI guard 变化：
  `OPERATIONS.md`、`docs/operations/ISSUE_BRIDGE.md`、`README.md`、`AGENTS.md`

- 文档 / requirement guard 自身变化：
  `docs/governance/DOC_SYSTEM.md`、`docs/governance/README.md`、`REQUIREMENT_SYNC_CHECKLIST.md`、`README.md`、`AGENTS.md`

- incoming requirement 被用户要求处理：
  `INCOMING_WORKFLOW.md`、`REQUIREMENT_SYNC_CHECKLIST.md`、`REQUIREMENT_CHANGELOG.md`，以及所有受影响的 canonical 文档

## 实现前守卫

同步矩阵现在不只是人工提醒。

`DOC_MANIFEST.json` 还定义了 `implementation_guard`：

- 哪些变更路径会触发哪些同步矩阵项
- 哪些路径应被忽略，避免纯文档或纯测试变更造成噪音

对应实现：

- `cmd/babel-dev/main.go`
  仓库级 Go 工具入口；默认统一承载文档一致性、同步守卫、requirement asset 校验和 guard 报告渲染

- `internal/devtools/quality/docs_consistency.go`
  校验 manifest、模块入口和链接结构

- `internal/devtools/quality/sync_guard.go`
  根据 `git diff` 或显式传入的路径，检查当前变更是否至少同步到了对应的 canonical 文档

- `internal/devtools/quality/requirement_assets.go`
  校验 `requirements/` registry、schema、asset 路径和跨资产引用

- `internal/devtools/quality/guard_report.go`
  把 guard 的结构化状态渲染成 markdown，供 CI summary 和 PR sticky comment 复用

- `cmd/babel-dev install-hooks`
  把 `.githooks/pre-commit` 安装到当前仓库本地配置

- `.githooks/pre-commit`
  默认串起上述三个检查

- `.github/workflows/docs-sync-guard.yml`
  在 GitHub 上对 PR / `main` push 执行同一套 manifest、测试和 diff-based 守卫

这意味着：

- 代码、guard 脚本或 requirement asset 改变后，如果没有任何相关 canonical 文档一起变化，提交前就会失败
- 相同的检查也会在 GitHub CI 中再次执行，不依赖某台机器是否装了本地 hook
- CI 失败时会上传 `.ci-artifacts/`，让排查不必只靠 Actions 控制台输出
- PR 流程里还会更新一条 sticky comment，把最新 guard 状态直接贴回 PR 讨论串
- 守卫是保守的最小门，不会要求每次都重读整套文档
- 文档同步的“需要读哪些文档”仍然由 topic 路由决定，而不是由 hook 强迫全量读取
- 从这轮开始，仓库级新工具默认只允许用 `Go` 和 `C++`；不再新增 Python 守卫脚本

## 按需读取路由

推荐的 topic -> read set 如下：

- 当前实现怎么工作：
  `ARCHITECTURE.md`、`SUBSYSTEM_BOUNDARIES.md`

- 长期目标应该收敛到哪里：
  `SYSTEM_TARGET_ARCHITECTURE.md`、`ROADMAP.md`

- 功能范围和继承边界：
  `FEATURE_INHERITANCE.md`、`ROADMAP.md`

- 接口、执行流、包契约：
  `INTERFACES.md`、`ARCHITECTURE.md`

- 持久化、execution record、state schema：
  `STORAGE_SCHEMA.md`、`INTERFACES.md`、`TESTING.md`

- 测试义务和验证方式：
  `TESTING.md`

- 节点、代理、启动器、服务器等待：
  `docs/operations/NODE_RUNTIME.md`

- issue bridge、watcher、手动接管、hook：
  `docs/operations/ISSUE_BRIDGE.md`

- 新 incoming 文档的处理：
  `INCOMING_WORKFLOW.md`、`REQUIREMENT_SYNC_CHECKLIST.md`、`REQUIREMENT_CHANGELOG.md`
  然后再读用户明确要求处理的那份 incoming 文档

- 文档体系本身如何保持一致：
  `DOC_SYSTEM.md`、`REQUIREMENT_SYNC_CHECKLIST.md`

## 读取纪律

默认规则：

1. 先读 [../INDEX.md](../INDEX.md)
2. 再进入对应模块 README
3. 从模块 README 路由到最小文档集合
4. 只有在信息不足、边界冲突或契约不清时，才扩展读取

这意味着：

- 不应在每次对话里重新通读所有 canonical 文档
- 不应把“文档很多”当作“必须全读”的理由
- 文档设计本身必须服务于 topic-based routing
