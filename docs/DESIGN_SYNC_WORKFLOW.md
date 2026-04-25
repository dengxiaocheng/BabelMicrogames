# 设计同步总览

本文不再承载全部细节，而是作为文档治理模块的总入口。

默认路径：

1. 先读 [governance/README.md](./governance/README.md)
2. 再按问题类型进入对应子文档

## 读取路由

- 如果要理解 canonical 文档之间的关系、最小同步矩阵和按需读取方式：
  读 [governance/DOC_SYSTEM.md](./governance/DOC_SYSTEM.md)

- 如果要处理用户明确要求吸收的 incoming 文档：
  读 [governance/INCOMING_WORKFLOW.md](./governance/INCOMING_WORKFLOW.md)

- 如果要确认实现前必须过哪些 gate：
  读 [REQUIREMENT_SYNC_CHECKLIST.md](./REQUIREMENT_SYNC_CHECKLIST.md)

- 如果要追踪某次变化已经反映到哪里：
  读 [REQUIREMENT_CHANGELOG.md](./REQUIREMENT_CHANGELOG.md)

## 总原则

- 文档读取默认按需进行，不再每轮通读整套集合
- `docs/incoming/` 只作为用户投递的暂存区
- 只有用户显式要求处理某份 incoming 文档时，才进入融合流程
