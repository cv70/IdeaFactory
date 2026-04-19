# Idea Factory 后端开发任务拆解

> 版本：v0.1-implementation-draft
> 日期：2026-04-19
> 状态：面向实现的后端开发清单

## 1. 文档目的

这份文档把新版目标态和后端实现映射进一步拆成可执行开发任务，目标是让后续开发可以直接按模块推进，而不是继续停留在抽象规格层。

它主要衔接两份文档：

- `IdeaFactory/docs/superpowers/specs/idea-factory-technical-design.md`
- `IdeaFactory/docs/superpowers/implementation/idea-factory-backend-runtime-mapping.md`

## 2. 总体开发原则

后端实现建议遵守五条原则：

- 先立协议骨架，再补长期能力对象
- 先加新模型和新接口，再兼容迁移旧接口
- graph 写入路径必须继续收口到统一工具面
- 每个新增对象都要能在 trace 中被观测到
- 避免一次性大重写，优先在 `domain/exploration` 内演进

## 3. 建议任务分组

建议把后端工作拆成 7 个任务组：

1. `Run / Turn / Checkpoint` 协议骨架
2. `ControlAction` 统一治理模型
3. `Projection / Trace` 升级
4. `Memory` 能力落地
5. `Skill Binding` 能力落地
6. `Policy / Approval` 能力落地
7. `API / 持久化 / 兼容迁移`

## 4. Task Group A：Run / Turn / Checkpoint 协议骨架

### A1. 扩展 `Run` 领域模型

目标文件：

- `IdeaFactory/backend/domain/exploration/schema.go`
- `IdeaFactory/backend/datasource/dbdao/agent_run_record.go`

任务：

- 给 `Run` 增加 `mode`
- 给 `Run` 增加 `waiting_reason`
- 给 `Run` 增加 `latest_checkpoint_id`
- 扩展 `RunStatus`：补 `waiting`、`cancelled`

完成标准：

- 领域层与持久化层都能表达新版 Run 状态
- 旧路径仍能继续创建 run，不会编译断裂

### A2. 正式定义 `RunTurn`

目标文件：

- `IdeaFactory/backend/domain/exploration/schema.go`
- `IdeaFactory/backend/domain/exploration/domain.go`

任务：

- 新增 `RunTurn` 结构体
- 为 `RuntimeWorkspaceState.Turns` 提供正式类型定义
- 为 `RuntimeStateSnapshot.Turns` 提供对外序列化结构

建议字段：

- `id`
- `workspace_id`
- `run_id`
- `index`
- `status`
- `input_context_digest`
- `tool_call_count`
- `graph_mutation_count`
- `continue_reason`
- `started_at`
- `finished_at`

完成标准：

- runtime 中可以稳定记录 turn
- `/runtime` 响应里能返回 turns

### A3. 在运行主链路中落地 turn 生命周期

目标文件：

- `IdeaFactory/backend/domain/exploration/handler_run.go`
- `IdeaFactory/backend/domain/exploration/runtime_agent.go`
- `IdeaFactory/backend/agents/exploration_runtime_handler.go`

任务：

- 在每次主 agent 推进前创建 turn
- 在工具执行与 graph 写入完成后更新 turn 指标
- 在本轮结束时把 turn 标记为 `completed / failed / superseded`

完成标准：

- 一次 run 至少能落出 1 个 turn
- turn 状态变化能进入 trace

### A4. 正式定义 `RunCheckpoint`

目标文件：

- `IdeaFactory/backend/domain/exploration/checkpoint.go`（建议新增）
- `IdeaFactory/backend/domain/exploration/domain.go`

任务：

- 新增 `RunCheckpoint` 结构体
- 在 runtime state 中记录 checkpoints
- 在关键 turn 完成后自动创建 checkpoint

建议首版策略：

- 先只做 metadata checkpoint
- 不做全量 graph snapshot duplication

完成标准：

- 每个 run 至少可产生 checkpoint
- checkpoint 可被 runtime 查询到

### A5. 给 graph mutation 绑定 `run_id + turn_id`

目标文件：

- `IdeaFactory/backend/agentools/append_graph_batch.go`
- `IdeaFactory/backend/domain/exploration/append_graph_batch.go`
- `IdeaFactory/backend/domain/exploration/mutations.go`

任务：

- 扩展 graph append 请求上下文
- 所有 mutation event 记录 `run_id + turn_id`
- projection builder 能引用最近 turn 的变更

完成标准：

- mutation 和 turn 有明确归因关系
- 可以回答“哪个 turn 改了什么图”

## 5. Task Group B：ControlAction 统一治理模型

### B1. 定义 `ControlAction` 领域对象

目标文件：

- `IdeaFactory/backend/domain/exploration/control_action.go`（建议新增）

任务：

- 定义 `ControlActionKind`
- 定义 `ControlActionStatus`
- 定义 `ControlAction` 结构体

建议 kinds：

- `intervention`
- `review_request`
- `artifact_request`
- `resume_request`
- `policy_adjustment`
- `memory_pin`

完成标准：

- 后端领域层存在统一控制动作模型

### B2. 让 `InterventionReq` 兼容迁移到 `ControlAction`

目标文件：

- `IdeaFactory/backend/domain/exploration/handler_intervention.go`
- `IdeaFactory/backend/domain/exploration/api.go`

任务：

- 新增 `CreateControlActionReq`
- 保留旧 `InterventionReq` 入口
- 内部统一转换成 `ControlAction{kind: intervention}`

完成标准：

- 旧接口仍可用
- 新的 control action 主流程已经存在

### B3. 在 runtime 中追踪 control action 生命周期

目标文件：

- `IdeaFactory/backend/domain/exploration/domain.go`
- `IdeaFactory/backend/domain/exploration/mutations.go`
- `IdeaFactory/backend/domain/exploration/trace_helpers.go`

任务：

- 在 `RuntimeWorkspaceState` 中增加 `ControlActions`
- 记录 `received / absorbed / reflected / rejected`
- 输出 `control_action_received`、`control_action_absorbed` 等事件

完成标准：

- runtime 和 trace 都能看到 control action 生命周期

### B4. 让 run/turn 能吸收 control action

目标文件：

- `IdeaFactory/backend/domain/exploration/runtime_agent.go`
- `IdeaFactory/backend/domain/exploration/main_agent_prompt.go`

任务：

- 在 context assembly 时注入未完成 control action
- 在 turn 结束后标记是否已 absorbed/reflected
- 为 `resume_request` 与 `review_request` 提供独立逻辑分支

完成标准：

- control action 真正影响 run，而不只是被记录

## 6. Task Group C：Projection / Trace 升级

### C1. 扩展 projection 输出

目标文件：

- `IdeaFactory/backend/domain/exploration/projection_builder.go`

任务：

- 增加 `run_summary`
- 增加 `turn_summary`
- 增加 `control_effects`
- 在最近变化里区分 graph 变化和 control action 影响

完成标准：

- projection 可直接支撑新版前端信息架构

### C2. 扩展 trace 分类

目标文件：

- `IdeaFactory/backend/domain/exploration/trace_helpers.go`
- `IdeaFactory/backend/domain/exploration/realtime.go`

任务：

- 新增 `turn`、`approval`、`control_action`、`memory`、`skill` 分类
- 所有关键动作统一产生 trace event

完成标准：

- trace 能覆盖新版 OpenAPI 所需类别

## 7. Task Group D：Memory 能力落地

### D1. 定义 `WorkspaceMemory` 领域对象

目标文件：

- `IdeaFactory/backend/domain/exploration/memory.go`（建议新增）

任务：

- 定义 `MemoryScope`
- 定义 `WorkspaceMemory`
- 定义最小 memory service 接口

完成标准：

- 领域层有稳定 memory 对象

### D2. 增加 memory 持久化表

目标文件：

- `IdeaFactory/backend/datasource/dbdao/workspace_memory.go`（建议新增）
- `IdeaFactory/backend/datasource/dbdao/dao.go`

任务：

- 新增 memory 表结构
- 补最小 CRUD DAO

完成标准：

- memory 可存可取

### D3. 在 turn 完成后提炼 memory

目标文件：

- `IdeaFactory/backend/domain/exploration/runtime_agent.go`
- `IdeaFactory/backend/domain/exploration/memory.go`

任务：

- 从 turn digest 中提炼高价值结论
- 将 preference 类内容提升为 `user_preference_memory`
- 支持 pinned memory

完成标准：

- 新 run 能装载历史 memory
- memory 不再只是 prompt 临时上下文

## 8. Task Group E：Skill Binding 能力落地

### E1. 定义 `SkillBinding` 领域对象

目标文件：

- `IdeaFactory/backend/domain/exploration/skill_binding.go`（建议新增）

任务：

- 定义 `SkillBinding`
- 定义 `SkillBindingStatus`
- 定义最小 activation 规则结构

### E2. 增加 skill binding 持久化表

目标文件：

- `IdeaFactory/backend/datasource/dbdao/workspace_skill_binding.go`（建议新增）

任务：

- 新增 skill binding 表
- 支持 workspace -> skills 查询

### E3. 在 context assembly 中装载 skills

目标文件：

- `IdeaFactory/backend/domain/exploration/runtime_agent.go`
- `IdeaFactory/backend/domain/exploration/main_agent_prompt.go`

任务：

- 根据 workspace 和 run mode 装载 skills
- 区分 `research_skill`、`artifact_skill`、`review_skill`

完成标准：

- 不同 run mode 能感知不同 skill bindings

## 9. Task Group F：Policy / Approval 能力落地

### F1. 定义 policy 模型

目标文件：

- `IdeaFactory/backend/domain/exploration/policy.go`（建议新增）

任务：

- 定义 `workspace_policy`
- 定义 `tool_policy`
- 定义 `approval_rule`

### F2. 给 tool 增加 risk level

目标文件：

- `IdeaFactory/backend/agentools/runtime_tools.go`
- `IdeaFactory/backend/agentools/append_graph_batch.go`
- 其他 tool 文件

任务：

- 给 tool 元信息补 `risk_level`
- 首版至少区分 `L0 / L1 / L2 / L3 / L4`

### F3. 在执行路径中记录审批事件

目标文件：

- `IdeaFactory/backend/domain/exploration/runtime_agent.go`
- `IdeaFactory/backend/domain/exploration/trace_helpers.go`

任务：

- 当 tool 触发高风险规则时记录 `approval_requested`
- 允许先只做事件与状态，不做完整人工审批工作流

完成标准：

- 新版 specs 中的 approval 语义在 runtime 中有真实落点

## 10. Task Group G：API / 持久化 / 兼容迁移

### G1. 增加新版 API handler

目标文件：

- `IdeaFactory/backend/domain/exploration/api.go`
- `IdeaFactory/backend/domain/exploration/routes.go`

任务：

- 增加 `/control-actions`
- 增加 `/runs/{runId}/turns`
- 增加 `/runs/{runId}/checkpoints`
- 增加 `/memory`
- 增加 `/skills`

### G2. 兼容旧 API

任务：

- 旧 `intervention` 接口内部复用 control action 流程
- 旧 `runtime` 接口逐步扩展返回新字段
- 旧 workbench 读取路径优先从 projection 派生

### G3. 补测试

建议优先补：

- `control_action` 生命周期测试
- `run -> turn -> checkpoint` 生命周期测试
- mutation 与 turn 归因测试
- memory/skill 查询测试
- projection 输出字段测试

## 11. 推荐实施顺序

最推荐的落地顺序：

1. A1-A5：先立 `run / turn / checkpoint`
2. B1-B4：再立 `control_action`
3. C1-C2：补 projection / trace
4. G1-G2：先把 API 跑通
5. D1-D3：再补 memory
6. E1-E3：再补 skill binding
7. F1-F3：最后补 policy / approval
8. G3：补测试并做旧接口收敛

## 12. Definition of Done

当下面这些条件满足时，可以认为后端完成了第一轮目标态落地：

- run 可以拆成 turn，turn 可以生成 checkpoint
- control action 不再只等于 intervention
- graph mutation 全部可绑定到 run + turn
- projection 能反映 turn 和 control action 的影响
- workspace memory 可以跨 run 生效
- workspace skill binding 可以被装载
- runtime 中至少存在基础 risk policy 与 approval event
- 新版 OpenAPI 的关键接口已有后端落点

## 13. 一句话总结

后端开发最重要的不是“继续加 agent 行为”，而是先把 `run-turn-checkpoint-control_action` 这套协议骨架立稳，再让 memory、skill、policy 长到这根骨架上。
