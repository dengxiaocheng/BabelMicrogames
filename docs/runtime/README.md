# 运行时模块

这个模块回答的是系统“谁拥有什么、接口如何连接、状态如何落盘、如何证明没坏”。

适用问题：

- 子系统边界怎么划
- 核心接口契约是什么
- execution / snapshot / artifact 怎么持久化
- 需要补哪些测试

推荐读取顺序：

1. [../SUBSYSTEM_BOUNDARIES.md](../SUBSYSTEM_BOUNDARIES.md)
   子系统所有权边界
2. [../INTERFACES.md](../INTERFACES.md)
   核心接口与模块契约
3. [../STORAGE_SCHEMA.md](../STORAGE_SCHEMA.md)
   持久化模型与记录形状
4. [../TESTING.md](../TESTING.md)
   验证义务与测试模式

按需读取规则：

- 只确认 ownership 时，先读 `SUBSYSTEM_BOUNDARIES.md`
- 只确认接口/调用关系时，先读 `INTERFACES.md`
- 只确认存储与 execution stage 时，先读 `STORAGE_SCHEMA.md`
- 只确认测试义务时，先读 `TESTING.md`
