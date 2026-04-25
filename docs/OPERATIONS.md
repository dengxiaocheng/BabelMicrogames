# 运维总览

本文不再承载全部细节，而是作为运维模块的总入口。

默认路径：

1. 先读 [operations/README.md](./operations/README.md)
2. 再按问题类型进入对应子文档

## 读取路由

- 如果要处理 Termux、节点、代理、启动器和服务器等待语义：
  读 [operations/NODE_RUNTIME.md](./operations/NODE_RUNTIME.md)

- 如果要处理 stage issue、watcher、manual takeover、结构化操作日志和 hook：
  读 [operations/ISSUE_BRIDGE.md](./operations/ISSUE_BRIDGE.md)

- 如果要处理当前 `online` 会话与 Babel / C++ 专用会话之间的结构化协作上下文：
  读 [operations/COLLAB_MCP.md](./operations/COLLAB_MCP.md)

- 如果要处理 ClaudeCode worker 如何把结果交回 Codex 管理线程：
  读 [operations/CLAUDECODE_MANAGER.md](./operations/CLAUDECODE_MANAGER.md)

- 如果要处理服务器节点和另一台实时开发节点之间的协同 handoff：
  先读 [operations/NODE_RUNTIME.md](./operations/NODE_RUNTIME.md)，再读 [operations/ISSUE_BRIDGE.md](./operations/ISSUE_BRIDGE.md)

## 总原则

- 运维知识应优先进入模块化文档，而不是留在聊天记录里
- 流程事件优先写入结构化日志或 hook，而不是向前台对话打印大量过程性噪音
- 会话与 handoff 识别应优先依赖结构化 state、hook 和固定入口，而不是让 assistant 在每轮对话里重新阅读整套流程文档
- 节点本地配置与仓库托管内容必须保持明确边界
