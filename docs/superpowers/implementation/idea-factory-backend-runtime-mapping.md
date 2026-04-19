# Idea Factory 后端实现映射说明

> 版本：v0.1-implementation-draft
> 日期：2026-04-19
> 状态：从 target specs 到当前 Go 实现的映射说明

## 1. 文档目的

这份文档不再讨论“目标态应该是什么”，而是回答一个更工程化的问题：

`当前 backend 代码离新版 specs 还有哪些距离，以及 control_action / run-turn-checkpoint / memory / skill / policy 应该怎样映射到现有 Go 结构。`

对应的正式目标态来源：

- `IdeaFactory/docs/superpowers/specs/idea-factory-technical-design.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-system-architecture.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-openapi.yaml`

## 2. 当前后端现状摘要

从当前代码结构看，现有探索内核已经具备几块基础能力：

- domain 主入口集中在 `IdeaFactory/backend/domain/exploration/`
- runtime 内存态集中在 `IdeaFactory/backend/domain/exploration/domain.go`
- 图写入工具已有基础实现：`IdeaFactory/backend/agentools/append_graph_batch.go`
- 运行时 handler 已存在：`IdeaFactory/backend/agents/exploration_runtime_handler.go`
- 持久化层已经有 workspace、graph、mutation、intervention 相关表模型：
  - `IdeaFactory/backend/datasource/dbdao/workspace_state.go`
  - `IdeaFactory/backend/datasource/dbdao/graph_node.go`
  - `IdeaFactory/backend/datasource/dbdao/graph_edge.go`
  - `IdeaFactory/backend/datasource/dbdao/mutation_log.go`
  - `IdeaFactory/backend/datasource/dbdao/intervention_event.go`

但当前模型仍明显偏旧：

- 外部控制动作仍以 `InterventionReq` 为主，而不是统一 `ControlAction`
- runtime 虽然已经出现 `RunTurn`、`RunCheckpoint` 字段入口，但尚未成为一等协议骨架
- memory、skill、policy 在领域模型中还没有独立对象
- API 仍主要围绕 legacy workspace/session/exploration 形态组织

## 3. 新版目标态对象与当前代码映射

### 3.1 Workspace

目标态对象：`Workspace`

当前主要落点：

- `IdeaFactory/backend/domain/exploration/schema.go` 中的 `ExplorationSession`
- `IdeaFactory/backend/datasource/dbdao/workspace_state.go`

建议映射：

| 目标态字段 | 当前承载位置 | 建议处理 |
| --- | --- | --- |
| `id` | `ExplorationSession.ID` | 保留 |
| `topic` / `goal` / `constraints` | `ExplorationSession.Topic` / `OutputGoal` / `Constraints` | 统一命名向 specs 对齐 |
| `status` | `workspace_state` 持久化层 | 保留并扩展 `paused / archived` 语义 |
| `charter` | 暂无稳定字段 | 新增到 workspace state 表或独立表 |
| `policy_ref` | 暂无 | 新增 workspace policy 关联 |

结论：

- 现有 `ExplorationSession` 可以继续作为 workspace 聚合根过渡对象
- 但需要逐步从“前台投影对象”转成“领域聚合对象”

### 3.2 Run

目标态对象：`Run`

当前主要落点：

- `IdeaFactory/backend/domain/exploration/schema.go` 中的 `Run`
- `IdeaFactory/backend/datasource/dbdao/agent_run_record.go`
- `RuntimeWorkspaceState.Runs`

当前问题：

- `RunStatus` 仍是 `pending/running/completed/failed` 旧枚举
- 缺少 `waiting / cancelled`
- `mode`、`waiting_reason`、`latest_checkpoint_id` 还没有成为标准字段

建议处理：

- 保留当前 `Run` 结构体作为过渡基座
- 先扩展字段，再扩展语义，再改 API
- `agent_run_record` 应成为 run 持久化主表，而不是只服务 agent record

### 3.3 RunTurn

目标态对象：`RunTurn`

当前主要落点：

- `RuntimeStateSnapshot.Turns []RunTurn` 已预留
- `RuntimeWorkspaceState.Turns []RunTurn` 已预留
- 相关具体协议语义尚未完全收口

建议处理：

- 在 `IdeaFactory/backend/domain/exploration/schema.go` 正式定义 `RunTurn`
- 在 `handler_run.go`、`runtime_agent.go` 中把每次主 agent 推进收口成 turn
- 每个 turn 至少记录：
  - `id`
  - `workspace_id`
  - `run_id`
  - `index`
  - `status`
  - `input_context_digest`
  - `tool_call_count`
  - `graph_mutation_count`
  - `continue_reason`

结论：

`RunTurn` 是当前后端从“有 runtime”升级成“有协议”的最关键对象。

### 3.4 Checkpoint

目标态对象：`RunCheckpoint`

当前主要落点：

- `RuntimeStateSnapshot.Checkpoints []RunCheckpoint` 已预留
- `RuntimeWorkspaceState.Checkpoints []RunCheckpoint` 已预留

建议处理：

- 在 turn 完成或关键 graph mutation 后自动创建 checkpoint
- 第一个版本只做 metadata checkpoint，不做完整状态快照复制
- checkpoint 最小字段建议：
  - `id`
  - `workspace_id`
  - `run_id`
  - `turn_id`
  - `label`
  - `reason`
  - `created_at`

### 3.5 ControlAction

目标态对象：`ControlAction`

当前主要落点：

- `InterventionReq` / `ApplyIntervention()`
- `IdeaFactory/backend/domain/exploration/handler_intervention.go`
- `intervention_event.go`

建议映射：

| 目标态 | 当前实现 | 过渡策略 |
| --- | --- | --- |
| `intervention` | `InterventionReq` | 先包一层 `ControlAction`，保留旧请求兼容 |
| `review_request` | 无 | 新增 `kind=review_request` |
| `artifact_request` | 无 | 新增 `kind=artifact_request` |
| `resume_request` | 无 | 新增 `kind=resume_request` |
| `policy_adjustment` | 无 | 新增 `kind=policy_adjustment` |
| `memory_pin` | 无 | 新增 `kind=memory_pin` |

推荐实施方式：

- 第一阶段不要立刻删掉 `InterventionReq`
- 先新增 `ControlActionReq`
- 然后让 `InterventionReq` 成为 `ControlActionReq{kind: intervention}` 的兼容别名

### 3.6 Memory

目标态对象：

- `run_working_memory`
- `workspace_memory`
- `user_preference_memory`

当前主要落点：

- 暂无独立领域对象
- 部分上下文散落在 `RuntimeWorkspaceState`、prompt 组装和 session 文本字段中

建议新增：

- `IdeaFactory/backend/domain/exploration/memory.go`
- `IdeaFactory/backend/datasource/dbdao/workspace_memory.go`

建议最小结构：

- `id`
- `workspace_id`
- `scope`
- `kind`
- `content`
- `source_run_id`
- `source_turn_id`
- `pinned`
- `updated_at`

第一阶段建议：

- 先只实现 `workspace_memory` 与 `user_preference_memory`
- `run_working_memory` 可先以内存态 + turn digest 过渡

### 3.7 Skill Binding

目标态对象：`SkillBinding`

当前主要落点：

- 暂无独立领域对象
- agent 行为仍主要靠 prompt 和工具硬编码

建议新增：

- `IdeaFactory/backend/domain/exploration/skill_binding.go`
- `IdeaFactory/backend/datasource/dbdao/workspace_skill_binding.go`

建议最小结构：

- `id`
- `workspace_id`
- `skill_key`
- `skill_type`
- `status`
- `activation_rule`
- `instructions_ref`

第一阶段目标不是“复杂技能系统”，而是让 workspace 能声明：

- 当前默认装载哪些研究模板
- 当前默认装载哪些 artifact 模板
- 当前默认装载哪些 review 模板

### 3.8 Policy / Approval

目标态对象：

- `workspace_policy`
- `tool_policy`
- `approval_rule`

当前主要落点：

- 工具风险主要隐含在代码路径中
- 没有稳定的 policy 持久化对象

建议新增：

- `IdeaFactory/backend/domain/exploration/policy.go`
- `IdeaFactory/backend/datasource/dbdao/workspace_policy.go`

建议先不要过度设计审批系统，先做三件事：

- 给 tool 标记 `risk_level`
- 给 workspace 记录默认 policy
- 在 trace 中记录 `approval_requested / approval_resolved`

## 4. 当前文件到目标态职责映射

### 4.1 最值得继续演进的文件

- `IdeaFactory/backend/domain/exploration/schema.go`
  - 承担领域对象收口
- `IdeaFactory/backend/domain/exploration/domain.go`
  - 承担 runtime 内存态收口
- `IdeaFactory/backend/domain/exploration/api.go`
  - 承担 legacy API 向目标态 API 过渡
- `IdeaFactory/backend/domain/exploration/handler_run.go`
  - 承担 run/turn/checkpoint 生命周期
- `IdeaFactory/backend/domain/exploration/handler_intervention.go`
  - 承担 intervention -> control_action 兼容迁移
- `IdeaFactory/backend/domain/exploration/projection_builder.go`
  - 承担 projection 对 turn/control_action 的映射
- `IdeaFactory/backend/agentools/append_graph_batch.go`
  - 承担 graph mutation 的 run/turn 归因

### 4.2 建议新增的文件

- `IdeaFactory/backend/domain/exploration/control_action.go`
- `IdeaFactory/backend/domain/exploration/memory.go`
- `IdeaFactory/backend/domain/exploration/skill_binding.go`
- `IdeaFactory/backend/domain/exploration/policy.go`
- `IdeaFactory/backend/domain/exploration/checkpoint.go`

## 5. 推荐迁移顺序

### Phase A：先收口协议骨架

目标：先把 `run / turn / checkpoint / control_action` 建出来。

建议顺序：

1. 扩展 `Run` 枚举和字段
2. 正式定义 `RunTurn`
3. 正式定义 `RunCheckpoint`
4. 新增 `ControlAction` 结构体
5. 让 runtime 和 projection 都能看见这些对象

### Phase B：再收口长期能力对象

目标：补 `memory / skill / policy`。

建议顺序：

1. `workspace_memory`
2. `user_preference_memory`
3. `workspace_skill_binding`
4. `workspace_policy`

### Phase C：最后替换 legacy API

目标：让现有 `intervention` / `session` / `workbench` API 逐步过渡到新版 OpenAPI。

建议方式：

- 先加新接口
- 再让旧接口调用新实现
- 最后在前端完成迁移后清理旧结构

## 6. 实施时的兼容原则

- 不要一次性推翻 `ExplorationSession`
- 不要先做复杂 swarm 再补 run/turn
- 不要先做完整 memory engine 再补 checkpoint
- 不要让 tool 绕过 `append_graph_batch` 直接写 graph
- 不要把 `ControlAction` 又退回成只剩 `InterventionReq`

## 7. 一句话总结

当前后端最现实的推进方式不是“重写 exploration domain”，而是以现有 `domain/exploration` 为基座，先把 `control_action + run/turn/checkpoint` 这套协议骨架立起来，再逐步补 `memory + skill + policy`。
