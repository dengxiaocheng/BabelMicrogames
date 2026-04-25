# 需求同步检查表

这份检查表用于降低设计漂移风险。

只要有有意义的新 requirement 文档被新增或更新，在进入实现前都应先过一遍这份检查表。

## Intake

- 确认文档已放在 `docs/incoming/`
- 确认这是用户主动要求处理的 incoming 文档，而不是助手自行看到后就开始处理
- 确认元数据字段存在
- 在 `docs/REQUIREMENT_CHANGELOG.md` 中新增或更新记录
- 识别这个文档属于哪一类：
  - exploratory
  - validated
  - architecture-significant
  - implementation-significant

## Canonical 反映

检查这次变化是否应更新以下任一文档：

- `docs/FEATURE_INHERITANCE.md`
- `docs/SYSTEM_TARGET_ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `docs/SUBSYSTEM_BOUNDARIES.md`
- `docs/INTERFACES.md`
- `docs/STORAGE_SCHEMA.md`
- `docs/TESTING.md`

如果答案是“应当更新”，那就应在代码规划之前或同时完成。

## Babel / C++ 对齐

- 这次变化是否修改了稳定世界规则？
- 它是否影响 deterministic core 的前提？
- 它是否改变了未来 C++ 的所有权边界？
- 它是否与已知的 Babel 侧约束冲突？

如果是，那么它就是架构级变化，不能只停留在 `docs/incoming/`。

## Godot 同步

- 这次变化是否影响实时场景行为？
- 它是否影响场景节奏或 sequencing？
- 它是否引入了未来应变成 Godot-facing 的已验证机制？

如果是，就应记录它是未来 Godot 同步的候选 requirement。

## Requirement-System 反映

- 这还是一个原始想法，还是已经成为被验证过的 gameplay？
- 如果已经被验证，它还能只保留为 prose 吗？

如果它已经被验证，最终应转化为 requirement-system assets，而不是只作为文档保留。

## 实现规划

- 哪个子系统拥有这次变化？
- 哪个子系统必须明确不拥有这次变化？
- 完成前需要具备哪些测试？
- 它属于当前 phase，还是后续 phase？
- 这次新增实现是否仍然遵守“默认只用 Go 和 C++，不再新增 Python”的仓库规则？
- 如果这次实现会改动 runtime / scripts / operations flow，当前 diff 是否已经同步修改了对应的 canonical 文档？
- 在本地提交前，是否已经运行 `go run ./cmd/babel-dev check-docs-sync-guard`，或已经安装 `.githooks/pre-commit`？
- 如果这次实现会改动 `requirements/` 下的 registry、schema 或 asset，是否已经运行 `go run ./cmd/babel-dev check-requirement-assets`？
- 如果这次实现会改动 CI guard / guard report，本次 PR 是否能从 sticky comment 或 `.ci-artifacts/` 里直接看见结果？
- 如果要依赖 GitHub 侧兜底，这次变更是否会被 `.github/workflows/docs-sync-guard.yml` 正常覆盖到？

## 收尾

在开始实现之前，确认以下事项已经满足：

- 这份 incoming 文档的处理动作是由用户显式触发的
- changelog 已更新
- 必要的 canonical 文档已更新
- 所有权边界清晰
- 测试预期已识别
- acceptance status 已记录
