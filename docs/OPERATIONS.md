# 运维入口

本仓只记录 ClaudeCode manager 自己的运维入口。

- 端到端流程：[operations/MICROGAME_FACTORY_FLOW.md](./operations/MICROGAME_FACTORY_FLOW.md)
- worker handoff：[operations/CLAUDECODE_MANAGER.md](./operations/CLAUDECODE_MANAGER.md)
- manager 升级路线：[operations/CODEX_MANAGER_INTELLIGENCE.md](./operations/CODEX_MANAGER_INTELLIGENCE.md)

节点、Termux、`s/m` 手动会话、Go issue bridge 和 GitHub hook 的真源在：

```text
/home/openclaw/babel-runtime/docs/operations/
```

本仓脚本只能作为 manager 业务层，不再承载节点控制面。
