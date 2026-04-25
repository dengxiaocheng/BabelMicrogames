# 文档索引

本目录是 Babel 新运行时的统一文档入口。

默认读取规则不是“整套通读”，而是：

1. 先从本页选择模块
2. 进入模块 README
3. 再从模块 README 路由到最小读取集合
4. 只有在信息不足或边界冲突时，才继续扩展

## 架构模块

- [architecture/README.md](./architecture/README.md)
  系统能力、当前架构、目标架构、路线图

核心文档：

- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [FEATURE_INHERITANCE.md](./FEATURE_INHERITANCE.md)
- [SYSTEM_TARGET_ARCHITECTURE.md](./SYSTEM_TARGET_ARCHITECTURE.md)
- [ROADMAP.md](./ROADMAP.md)

## 运行时模块

- [runtime/README.md](./runtime/README.md)
  子系统边界、接口、存储、测试

核心文档：

- [SUBSYSTEM_BOUNDARIES.md](./SUBSYSTEM_BOUNDARIES.md)
- [INTERFACES.md](./INTERFACES.md)
- [STORAGE_SCHEMA.md](./STORAGE_SCHEMA.md)
- [TESTING.md](./TESTING.md)

## 文档治理模块

- [governance/README.md](./governance/README.md)
  文档关系、incoming 流程、同步 gate、变更台账

入口文档：

- [DESIGN_SYNC_WORKFLOW.md](./DESIGN_SYNC_WORKFLOW.md)
- [governance/DOC_SYSTEM.md](./governance/DOC_SYSTEM.md)
- [governance/INCOMING_WORKFLOW.md](./governance/INCOMING_WORKFLOW.md)
- [REQUIREMENT_SYNC_CHECKLIST.md](./REQUIREMENT_SYNC_CHECKLIST.md)
- [REQUIREMENT_CHANGELOG.md](./REQUIREMENT_CHANGELOG.md)

## 运维模块

- [operations/README.md](./operations/README.md)
  节点入口、watcher、manual takeover、hook

入口文档：

- [OPERATIONS.md](./OPERATIONS.md)
- [operations/NODE_RUNTIME.md](./operations/NODE_RUNTIME.md)
- [operations/ISSUE_BRIDGE.md](./operations/ISSUE_BRIDGE.md)
- [operations/COLLAB_MCP.md](./operations/COLLAB_MCP.md)
- [operations/WINDOWS_LOCAL.md](./operations/WINDOWS_LOCAL.md)
- [operations/CLAUDECODE_MANAGER.md](./operations/CLAUDECODE_MANAGER.md)

## 规划档案

下面这些文件仍然有参考价值，但它们属于更早期的规划输入，不是当前的主导航集合：

- [planning/GO_SYSTEM_BLUEPRINT.md](./planning/GO_SYSTEM_BLUEPRINT.md)
- [planning/GO_STATE_SCHEMA.md](./planning/GO_STATE_SCHEMA.md)
- [planning/GO_REPO_LAYOUT.md](./planning/GO_REPO_LAYOUT.md)
- [planning/GO_BUILD_PLAN.md](./planning/GO_BUILD_PLAN.md)
- [planning/GO_MODULE_DESIGN.md](./planning/GO_MODULE_DESIGN.md)
- [planning/GO_PERSISTENCE_EVENTLOG.md](./planning/GO_PERSISTENCE_EVENTLOG.md)
- [planning/LLM_IO_SCHEMA.md](./planning/LLM_IO_SCHEMA.md)
- [planning/RECOVERY_CHECKPOINT_MODEL.md](./planning/RECOVERY_CHECKPOINT_MODEL.md)
- [planning/RUNTIME_STEP_SEQUENCES.md](./planning/RUNTIME_STEP_SEQUENCES.md)
- [planning/CPP_SIM_CONTRACT.md](./planning/CPP_SIM_CONTRACT.md)
- [planning/TEST_SYSTEM_PLAN.md](./planning/TEST_SYSTEM_PLAN.md)

## 新设计暂存区

- [incoming/README.md](./incoming/README.md)
  新设计文档进入后的暂存区，尚未完成 canonical 化前先放在这里。

- [incoming/TEMPLATE.md](./incoming/TEMPLATE.md)
  新 requirement / design 文档的默认模板。

## 文档规则

- `docs/` 下的 canonical prose 默认使用简体中文。
- 代码标识符、接口名、协议关键字、路径和命令示例继续保留英文。
- 高信号的稳定文档优先按模块组织。
- 被替换的草稿和早期设计输入放在 `docs/planning/`。
- `docs/incoming/` 只作为用户投递的暂存区，不是 assistant 的主动工作队列。
