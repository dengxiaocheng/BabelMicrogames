# LLM 输入输出 Schema

## 目的

定义新系统如何使用 LLM 的严格边界。

LLM 只负责 narrative 和 option wording，不拥有 canonical state transition。

## 规则

1. LLM 只接收可见事实或被明确授权提供的事实
2. LLM 必须返回结构化输出
3. LLM 不能直接修改 canonical state
4. 输出在投递前必须经过 schema 校验

## 请求类型

### 1. Solo Scene Render

```yaml
SoloRenderRequest:
  request_type: solo_render
  request_id: string
  prompt_pack_id: string
  tone_pack_id: string
  chapter_id: string
  world_clock:
    day_index: int
    segment: string
  visible_state:
    player_summary: string
    environment_summary: string
    recent_summary: string
    consequences: [VisibleConsequence]
  action_context:
    player_action_label: string
    legal_option_tags: [string]
  output_constraints:
    max_scene_chars: int
    max_option_chars: int
    option_count: int
```

### 2. Multiplayer Shared Render

```yaml
MultiplayerSharedRenderRequest:
  request_type: multiplayer_shared_render
  request_id: string
  prompt_pack_id: string
  tone_pack_id: string
  chapter_id: string
  world_clock:
    day_index: int
    segment: string
  shared_visible_state:
    room_summary: string
    shared_consequences: [VisibleConsequence]
    participant_labels: [ParticipantLabel]
  output_constraints:
    max_scene_chars: int
```

### 3. Multiplayer Private Render

```yaml
MultiplayerPrivateRenderRequest:
  request_type: multiplayer_private_render
  request_id: string
  player_id: string
  prompt_pack_id: string
  tone_pack_id: string
  chapter_id: string
  world_clock:
    day_index: int
    segment: string
  shared_summary: string
  private_visible_state:
    player_summary: string
    private_consequences: [VisibleConsequence]
  action_context:
    legal_option_tags: [string]
  output_constraints:
    max_scene_chars: int
    max_option_chars: int
    option_count: int
```

## 共享输入片段

### VisibleConsequence

```yaml
VisibleConsequence:
  tag: string
  text_hint: string
  severity: int
```

### ParticipantLabel

```yaml
ParticipantLabel:
  actor_id: string
  display_name: string
  visible_role: string
```

## 响应 Schema

### SoloRenderResponse

```yaml
SoloRenderResponse:
  scene_text: string
  options: [RenderOption]
  metadata:
    tone_tags: [string]
    safety_flags: [string]
```

### MultiplayerSharedRenderResponse

```yaml
MultiplayerSharedRenderResponse:
  shared_scene_text: string
  metadata:
    tone_tags: [string]
```

### MultiplayerPrivateRenderResponse

```yaml
MultiplayerPrivateRenderResponse:
  private_scene_text: string
  options: [RenderOption]
  metadata:
    tone_tags: [string]
    safety_flags: [string]
```

### RenderOption

```yaml
RenderOption:
  option_id: string
  label: string
  action_tag: string
```

## 禁止输出

模型输出不得包含：

- 暗示 canonical 数值的状态条，除非这是明确要求的派生展示
- 未出现在 consequences 中的 inventory change
- 未经允许引入的新命名角色
- chapter 跳转
- 跨 segment 的时间跃迁
- 在 public output 中泄露 hidden fact

## 校验层

Go 应校验：

- JSON / schema 结构
- 字符串长度
- option 数量
- option id 不重复
- 不存在 forbidden tags
- 不存在非法 hidden fact 泄露

如果校验失败：

1. 用严格 repair prompt 重试
2. 必要时回退到确定性紧凑渲染器

## Prompt Pack 结构

Prompt pack 应可热更新并带版本。

建议组成：

- role / system framing
- tone pack
- format rules
- chapter flavor
- safety and scope constraints

## 示例运行流

1. Go 应用确定性 simulation 结果
2. Go 构建 visible consequence 集合
3. Go 发送紧凑 render request
4. LLM 返回 schema output
5. Go 校验
6. Go 保存 render frame
7. Go 投递给用户

## 未来扩展

可增加的 request type 包括：

- recap render
- injury notice render
- admin summary render
- NPC dialogue stylization

但这些扩展都必须服从同一条核心规则：

LLM 输出是 presentation，不是真值。
