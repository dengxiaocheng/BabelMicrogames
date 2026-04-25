# Incoming 同步流程

本文定义 incoming 文档何时可以处理，以及如何把它们吸收到 canonical 集合。

## 触发前提

新的设计输入应先放入：

- `docs/incoming/`

但这里只有用户可以投递新文件。

当前约束是：

- 只有用户可以向 `docs/incoming/` 新增文件
- assistant 不能主动在 `docs/incoming/` 创建新文件
- assistant 不能因为“看见了新 incoming 文档”就自行开始处理
- 只有在用户明确要求处理某份 incoming 文档时，才进入下面流程

## 处理流程

当用户明确要求处理某份 incoming 文档时，默认流程应当是：

1. 获取最新仓库状态
2. 读取用户明确指定的 incoming 文档
3. 总结变化点
4. 将变化归类为：
   - 架构级
   - 子系统级
   - gameplay / rules 级
   - Godot-facing
   - Babel / C++ 对齐
   - requirement-management
5. 根据变化类型映射到必须同步的 canonical 文档集合
6. 在改代码之前或同时更新 canonical 文档
7. 在 `REQUIREMENT_CHANGELOG.md` 中登记本次变化
8. 只有在 canonical 集合同步完成后，才把变化传播到实现规划和代码

## 分拣去向

incoming 文档处理后，应进入以下去向之一：

### A. 融合进 Canonical 文档

如果文档定义了稳定方向，就应把其内容吸收到：

- `ARCHITECTURE.md`
- `FEATURE_INHERITANCE.md`
- `INTERFACES.md`
- `SYSTEM_TARGET_ARCHITECTURE.md`
- 或其他 canonical 文档

### B. 归档为规划输入

如果文档有参考价值，但不应成为 canonical 文档，应转入：

- `docs/planning/`

### C. 转化为 Requirement Asset

如果设计更新本质上已经是 gameplay 或产品 requirement，它就不应只停留在 prose 文档里，而应进一步进入 requirement-management 层。

## 归档规则

一旦 incoming 文档已经被处理并完成融合：

- 不应继续作为活跃 loose drop 留在 `docs/incoming/`
- 应进入归档状态
- 归档位置可以是 `docs/planning/` 或仓库定义的其他归档区

## 元数据要求

进入仓库的 incoming 文档应尽量使用：

- `docs/incoming/TEMPLATE.md`

至少应包含：

- `status`
- `date`
- `source`
- `scope`
- `sync_targets`

## 必须同步的目标

任何有意义的输入设计变化，都应检查是否影响：

- `FEATURE_INHERITANCE.md`
- `SYSTEM_TARGET_ARCHITECTURE.md`
- `INTERFACES.md`
- `TESTING.md`
- 未来的 requirement-management assets

如果变化改变了系统方向，至少应更新其中一个文档，并同步登记到：

- `REQUIREMENT_CHANGELOG.md`

## 特殊判定

如果新的设计文档改变了以下任一假设：

- 稳定世界规则
- deterministic core 约束
- simulation 边界
- Babel 集成预期
- 未来 C++ 所有权

那么它属于架构级变化，不能只停留在 `docs/incoming/`。

如果新的设计文档改变了：

- 实时场景预期
- 交互流程
- 已验证的 realtime gameplay loop
- narrative 到 scene 的映射

那么它应被视为未来 Godot requirement pipeline 的一部分。
