# 运维模块

这个模块回答的是节点侧怎么启动、怎么等待、怎么 handoff、怎么通过 watcher / hook 驱动继续执行。

适用问题：

- Termux、节点、代理、启动器如何配合
- 服务器常驻怎么运行
- stage issue / watcher / manual takeover 怎么运转
- 服务器节点与其他实时开发节点怎么 handoff
- 哪里能看结构化流程日志，哪里能挂 hook

推荐读取顺序：

1. [NODE_RUNTIME.md](./NODE_RUNTIME.md)
   节点入口、代理、启动器与服务器等待语义
2. [ISSUE_BRIDGE.md](./ISSUE_BRIDGE.md)
   issue bridge、watcher、manual takeover、事件日志与 hook
3. [COLLAB_MCP.md](./COLLAB_MCP.md)
   `online` 会话与 Babel / C++ 专用会话之间的结构化协作状态
4. [WINDOWS_LOCAL.md](./WINDOWS_LOCAL.md)
   Windows 本地运行脚本的归档、交付与运维入口规则
5. [CLAUDECODE_MANAGER.md](./CLAUDECODE_MANAGER.md)
   ClaudeCode worker 与 Codex 管理线程之间的最小交接流

按需读取规则：

- 只处理节点登录、代理、启动器时，先读 `NODE_RUNTIME.md`
- 只处理 GitHub issue 驱动继续执行时，先读 `ISSUE_BRIDGE.md`
- 只处理双会话边界、scope 认领、handoff 和记要同步时，先读 `COLLAB_MCP.md`
- 处理服务器节点与 Windows 等实时开发节点之间的交接时，两份都要读
- 处理 Windows 本地启动器、重写脚本、排障脚本时，读 `WINDOWS_LOCAL.md`
- 处理 ClaudeCode worker 如何交回 Codex manager 时，读 `CLAUDECODE_MANAGER.md`
