# 测试策略

测试是运行时设计的一部分，不是后置补丁。

## 必备测试层

- pure deterministic core test
- kernel execution-state-machine test
- repository transaction、idempotency、lease test
- `free_chat`、`project_consult`、`solo_scene`、`room_scene` 的 mode contract test
- projection visibility test
- delivery idempotency / retry test
- replay / restart simulation test
- requirement asset compatibility test
- requirement registry / asset reference validation test
- requirement registry loader / kernel integration test

## 必备支撑包

- fake clock
- fake repository
- fake agent supervisor
- fake projector
- fake dispatcher
- restart simulator
- mode fixture builder
- `SceneHost` ABI fixture

现有脚手架中的 fake / builder 仍然可能有价值，但测试支撑应逐步转向 redesigned kernel model。

## 必须证明的场景

1. 同一个 inbound envelope 不能双重提交 canonical state
2. execution 能从每个已持久化 stage 恢复
3. agent task failure 不会损坏 canonical runtime state
4. free chat / consult mode 不会在 kernel 之外制造 hidden state authority，并且只通过已持久化 execution stage 接受 artifact
5. 在 agent artifact 固定时，solo scene / room scene 在 replay 下仍然保持 deterministic
6. projection 和 delivery 可以被重跑，而不改变 canonical state
7. file-backed operational memory 可以从 collected artifact 重新生成，但不会变成 canonical truth
8. ruleset / prompt pack revision change 会在发布前反映到测试中
9. requirement registry 中的 asset path、registry entry 和跨资产引用在本地 hook 与 CI 中都必须可验证
10. runtime 上声明的 requirement bundle 必须能被 loader 解析，并显式进入 mode 执行输入
11. Go / C++ `SceneHost` 共享 ABI 应保持 fixture 级一致，至少覆盖 `solo_step` 与 `room_step`
12. Babel/C++ 共享库一旦可产出，必须能通过 `verify-scene-host-library` 跑过同一套 `solo_step / room_step` fixture 校验
13. Godot-facing projection 必须派生自 canonical snapshot，而不是 handler 层的 ad hoc state

当前仓库还应保留一条最小 shared-library smoke harness：测试可以临时编出 fixture-based `.so`，并用真实 `dlopen + babel_sim_step` 跑通 `verify-scene-host-library`，确保 Go host 侧 loader、ABI 验证器和未来 Babel/C++ 共享库不会只停在静态文档约定上。

接入层测试还应覆盖：当 `wechatapp` 配置了共享库路径时，启动必须先完成 fixture 验证，`/healthz` 也必须回报当前 host 模式、来源与验证状态，避免联调时只能从日志猜测当前到底跑的是 fallback 还是 shared-library。

如果运行态配置了 `BABEL_SCENE_CORE_LIBRARY=@collab`，测试还必须覆盖：当 collaboration MCP 中没有可消费的 `scene_host_library` artifact 时，服务启动应直接失败，而不是静默回退到 Go fallback。否则双会话联调时会把“C++ 产物缺失”误报成“系统正常运行”。
