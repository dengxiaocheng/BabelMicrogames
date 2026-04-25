# Windows 本地脚本归档

这个目录用于归档 Windows 本地直接执行的运行脚本。

范围包括：

- 本地启动器
- 本地重写脚本
- 本地排障脚本
- 本地一键运行脚本
- 本地手工恢复入口

约定：

- 可复用的 Windows 本地脚本先落到这里，再在聊天里引用
- 不要把 Windows 本地运行脚本散落到仓库根目录
- 对应说明统一收口到 `docs/operations/WINDOWS_LOCAL.md`
- 命名优先使用清晰的功能名，例如：
  - `rewrite_*.ps1`
  - `check_*.ps1`
  - `launch_*.cmd`
  - `resume_*.bat`
