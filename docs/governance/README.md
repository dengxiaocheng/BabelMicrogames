# 文档治理模块

这个模块回答的是文档体系如何保持一致，以及新输入如何被吸收进 canonical 集合。

适用问题：

- 各份 canonical 文档之间是什么关系
- 应该按什么最小读取集合读取文档
- incoming 文档何时可以处理
- 新 requirement / design drop 应如何同步

推荐读取顺序：

1. [DOC_SYSTEM.md](./DOC_SYSTEM.md)
   文档关系、同步矩阵和按需读取路由
2. [DOC_MANIFEST.json](./DOC_MANIFEST.json)
   文档关系、topic 路由和同步矩阵的机器可读定义
3. [INCOMING_WORKFLOW.md](./INCOMING_WORKFLOW.md)
   incoming 文档的处理流程与分拣规则
4. [../REQUIREMENT_SYNC_CHECKLIST.md](../REQUIREMENT_SYNC_CHECKLIST.md)
   进入实现前的同步检查项
5. [../REQUIREMENT_CHANGELOG.md](../REQUIREMENT_CHANGELOG.md)
   变化台账与状态记录
6. [../../cmd/babel-dev/main.go](../../cmd/babel-dev/main.go)
   仓库级 Go 工具入口，承载文档一致性、同步守卫、requirement asset 校验和 guard 报告渲染
7. [../../internal/devtools/quality/docs_consistency.go](../../internal/devtools/quality/docs_consistency.go)
   轻量检查模块入口和相对链接是否断裂
8. [../../internal/devtools/quality/sync_guard.go](../../internal/devtools/quality/sync_guard.go)
   根据当前变更范围检查同步矩阵是否已反映到 canonical 文档
9. [../../internal/devtools/quality/requirement_assets.go](../../internal/devtools/quality/requirement_assets.go)
   校验 requirement-management foundation 的 registry、schema 和资产引用
10. [../../internal/devtools/quality/guard_report.go](../../internal/devtools/quality/guard_report.go)
   把 guard 状态渲染成 CI summary / PR comment 可复用的 markdown 报告
11. [../../cmd/babel-dev/main.go](../../cmd/babel-dev/main.go)
   其中 `install-hooks` 子命令把上述守卫接到本地 `pre-commit`
12. [../../.github/workflows/docs-sync-guard.yml](../../.github/workflows/docs-sync-guard.yml)
   在 GitHub CI 中执行同一套文档守卫

按需读取规则：

- 只想知道“该读哪几份文档”，先读 `DOC_SYSTEM.md`
- 只想知道“incoming 文档怎么处理”，先读 `INCOMING_WORKFLOW.md`
- 只想确认“开始实现前要过哪些 gate”，先读 `REQUIREMENT_SYNC_CHECKLIST.md`
- 只想确认“当前改动会不会因为文档没同步被 hook 拦下”，先读 `DOC_SYSTEM.md` 和 `sync_guard.go`
- 只想确认“requirement foundation 的 JSON 资产有没有漂移”，先读 `requirement_assets.go`
- 只想确认“CI 最后会把什么报告贴回 PR”，先读 `guard_report.go`
- 只想追踪“某次变化最后落到了哪里”，先读 `REQUIREMENT_CHANGELOG.md`
