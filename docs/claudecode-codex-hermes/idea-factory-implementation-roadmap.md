# Idea Factory 吸收 Claude Code / Codex / Hermes 模式的实施路线图

> 版本：v0.2-draft
> 日期：2026-04-19
> 状态：与新版 specs 对齐的分阶段实施建议

## 1. 总体原则

目标不是“集成三个 agent 产品”，而是把三种成熟模式拆成可落地工程阶段：

- 先补控制面
- 再补执行协议
- 再补长期记忆与技能
- 最后才引入 specialist、多入口与自动化

原因很简单：

- 没有控制面，自治系统不可驾驭
- 没有执行协议，自治系统不可治理
- 没有长期能力，自治系统不可积累
- 过早上多入口和多 agent，只会把问题放大

## 2. 当前新版 specs 已经明确的目标态

目前 `docs/superpowers/specs/` 已经把下面这些对象正式收进目标态：

- `control_action`
- `run`
- `turn`
- `checkpoint`
- `workspace_memory`
- `skill_binding`
- `tool risk policy`

因此本路线图的意义不再是“是否要引入这些概念”，而是“先后怎么实现这些概念”。

## 3. Phase 1：补齐控制面与可观测运行状态

### 目标

优先吸收 `Claude Code` 的控制面思想，让 `Idea Factory` 从“后台 agent 在跑”升级为“用户能看见并驾驭系统在跑”。

### 交付重点

- 增加 `control_action` 领域对象
- 把 `pause / resume / redirect / review / artifact request / policy adjustment` 做成结构化 API
- 为 UI 暴露当前 run / turn / waiting reason / checkpoint 状态
- 明确 control action 的吸收状态：`received / absorbed / reflected`
- 增加基础运行事件流与控制动作事件流

### 建议落点

- `IdeaFactory/backend/domain/exploration/`
- `IdeaFactory/frontend/src/types/`
- `IdeaFactory/frontend/src/lib/`
- `IdeaFactory/docs/superpowers/specs/idea-factory-product-design.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-openapi.yaml`

### 完成标志

系统至少能回答：

- 现在是否在自动推进
- 当前 run 处于哪一轮、在等什么
- 用户最近一次治理动作有没有真正影响到地图
- 是否可以从稳定点继续推进

## 4. Phase 2：把 Run 做成 turn-level 执行协议

### 目标

吸收 `Codex` 的 task / turn / approval 思想，让 `Idea Factory` 的自治推进变成可恢复、可追踪、可解释的协议。

### 交付重点

- 显式增加 `RunTurn`
- 增加 `run_checkpoint`
- 给每次 graph mutation 绑定 `run_id + turn_id`
- 定义 tool risk levels 与 approval rules
- 定义标准事件：`turn_started`、`tool_called`、`graph_mutation_committed`、`projection_published`、`run_completed`

### 建议落点

- `IdeaFactory/backend/domain/exploration/`
- `IdeaFactory/backend/agents/exploration_runtime_handler.go`
- `IdeaFactory/backend/agentools/append_graph_batch.go`
- `IdeaFactory/docs/superpowers/specs/idea-factory-technical-design.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-system-architecture.md`

### 完成标志

系统至少能回答：

- 一个 run 被拆成了几次 turn
- 每次 turn 做了什么工具调用与图谱写入
- 哪个 checkpoint 对应哪次重要方向变化
- 某次审批或失败是在哪一轮出现的

## 5. Phase 3：为 Workspace 增加长期记忆与 Skill Binding

### 目标

吸收 `Hermes` 的 session / memory / skills 思想，让 `workspace` 真正具备长期学习能力。

### 交付重点

- 增加 `workspace_memory`
- 增加 `user_preference_memory`
- 增加 `workspace_skill_binding`
- 支持研究模板、artifact 模板、review 模板按需装载
- 支持 turn 结束后的异步 context prefetch

### 建议的记忆分层

- `run working memory`
- `workspace memory`
- `user preference memory`

### 完成标志

系统不只是“能继续跑”，而是“带着过去学到的东西继续跑”。

## 6. Phase 4：谨慎引入 specialist / review worker / artifact worker

### 目标

只在明确收益场景下吸收 `Hermes` 与 `Claude Code` 展示出的 multi-agent 协作能力。

### 建议先支持三类 specialist

- `ResearchAgent`
- `ReviewAgent`
- `ArtifactAgent`

### 关键约束

- 主判断仍归 `MainAgent`
- specialist 默认没有 graph 直写特权
- specialist 输出必须通过受控 tool 或回收给主 agent
- 是否引入 specialist 的判断标准是“显著缩短时间或提高质量”，不是“架构看起来更先进”

### 完成标志

并行研究、专项审查、artifact 物化可以提升效率，但 workspace 真相层仍然只有一套。

## 7. Phase 5：开放多入口、调度与自动化

### 目标

最后再吸收 `Claude Code` 的 remote / trigger / plugin 与 `Hermes` 的 gateway / long-running shell 思路。

### 可能能力

- 定时探索 run
- webhook 或外部信号触发 control action
- API / workbench / automation bridge 多入口
- workspace 级 skill pack 与 connector 扩展
- 周报式方向变化摘要

### 注意事项

这一步必须最后做，因为：

- 没有稳定协议，多入口只会放大混乱
- 没有稳定 memory，自动化只会堆出脏状态
- 没有稳定 control plane，远程触发会让系统难以治理

## 8. 当前不建议做的事

为防止范围失控，建议暂缓：

- 把 `Idea Factory` 做成通用聊天平台
- 过早支持过多外部消息渠道
- 在 graph truth layer 之外再造一套并行真相系统
- 在没有 turn/checkpoint/policy 的前提下大量引入 subagents
- 把所有探索套路硬编码进主 prompt，而不给 skill 与 policy 留演进空间

## 9. 成功判据

如果这套路线上线后有效，应该能看到：

- 用户能更轻松地介入并改变系统后续几轮行为
- 同一个 workspace 可以长期推进且仍然可解释
- 相似 workspace 的重复劳动明显减少
- 方向地图与 artifact 产出之间的关联更稳定
- 研究、审查、产出三类工作能在同一 runtime 下被统一治理

## 10. 一句话总结

先学 `Claude Code` 做控制台，再学 `Codex` 管执行，再学 `Hermes` 养长期能力；等这三层稳定之后，`Idea Factory` 才适合进入真正的自治探索平台阶段。
