# Idea Factory 技术设计文档（Target State）

> 版本：v1.1-target
> 日期：2026-04-19
> 状态：目标态技术规范（后端内核优先）

## 1. 文档职责与边界

本文件定义后端内核的目标态：

- 领域对象与状态机
- `workspace -> run -> turn -> graph mutation -> projection` 主链路
- 控制面对象、长期记忆对象与技能对象的语义边界
- 程序侧与 Agent 侧的职责边界
- 失败、恢复与审批语义

本文件不定义：

- 产品页面布局与交互文案（见产品设计）
- 基础设施选型、部署与系统容量细节（见系统架构）

## 2. 内核设计不变量

- `Workspace` 是顶层业务契约，也是长期探索容器
- `Run` 是一次自治推进实例
- `RunTurn` 是运行治理的最小解释单元
- `MainAgent` 是 graph 生长与优化的唯一最终决策者
- `Graph` 是方向结构真相层
- `Projection` 是前端读取面，不是写入面
- `ControlAction` 改变系统的后续行为，但不能绕过工具面直接改写真相层
- 程序侧只保留调度、最小校验、持久化、广播、审批与恢复，不内置 graph 阶段策略
- 长期能力通过 `workspace memory + skill binding + policy` 积累，而不是通过无限膨胀的系统提示词积累

## 3. 核心领域模型与状态流转

### 3.1 Workspace

关键字段：`id`、`topic`、`goal`、`constraints`、`budget`、`status`、`charter`、`policy_ref`。

状态流转：`draft -> active -> paused -> archived`。

说明：

- `Workspace` 是长期 session 容器，但必须 graph-first。
- `charter` 用于承载 workspace 级目标、禁区、偏好与验收口径。

### 3.2 Run

关键字段：`id`、`workspace_id`、`trigger_type`、`mode`、`status`、`started_at`、`finished_at`、`checkpoint_cursor`。

状态流转：

```text
queued -> running -> waiting | completed | failed | cancelled
```

说明：

- `Run` 对应一次受控自治推进。
- `waiting` 表示 run 正在等待审批、外部输入、review 或系统资源，而不是已经失败。
- 程序侧不再显式维护 `ExecutionPlan` / `PlanStep` 作为 graph 生长前置条件。

### 3.3 RunTurn

关键字段：`id`、`workspace_id`、`run_id`、`index`、`status`、`input_context_digest`、`continue_reason`、`started_at`、`finished_at`。

状态流转：

```text
queued -> running -> completed | failed | superseded
```

职责：

- 记录一次模型推理 + 工具调用 + 结果集成
- 作为 intervention 吸收、错误定位、恢复与审计的最小边界
- 解释“系统为什么在这一轮做了这些事”

### 3.4 Agent Session

关键字段：`workspace_snapshot`、`active_run_id`、`recent_turns`、`recent_mutations`、`working_memory_digest`、`loaded_skills`、`active_policy`。

职责：

- 为 `MainAgent` 提供当前 workspace、graph、最近变更、记忆、技能与控制动作上下文
- 记录运行时活动摘要，而不是 planner/step 结构
- 作为恢复和审计的最小运行态边界

### 3.5 BalanceState

关键字段：

- `diverge_converge_mode`
- `research_produce_mode`
- `aggressive_prudent_mode`
- `reason`

说明：

- `BalanceState` 是给 Agent 的节奏提示，不是程序侧 graph 规则机。
- 程序可以更新 balance，但不能依据 balance 直接决定新增哪类节点或边。

### 3.6 ControlAction

关键字段：`id`、`workspace_id`、`kind`、`payload`、`status`、`created_at`、`absorbed_by_run_id`、`reflected_at`。

建议类型：

- `intervention`
- `review_request`
- `artifact_request`
- `resume_request`
- `policy_adjustment`
- `memory_pin`

状态流转：

```text
received -> absorbed -> reflected | rejected
```

### 3.7 WorkspaceMemory

关键字段：`id`、`workspace_id`、`scope`、`kind`、`content`、`source_run_id`、`pinned`、`updated_at`。

建议 scope：

- `run_working`
- `workspace`
- `user_preference`

### 3.8 SkillBinding

关键字段：`id`、`workspace_id`、`skill_key`、`skill_type`、`status`、`activation_rule`、`instructions_ref`。

说明：

- `skill` 提供策略、模板与流程
- `tool` 提供真正会改状态的执行入口
- `skill` 不能绕过 `tool` 直接改 graph

### 3.9 Projection

关键字段：`graph_snapshot`、`focus_branches`、`recent_changes`、`run_summary`、`turn_summary`、`control_effects`。

约束：`Projection` 只读可重建；任何写入都必须经 run 主链路产生。

## 4. 图本体定义（Graph Ontology）

Graph 是方向结构的真相层。所有 graph 变化必须由 Agent 通过受控 graph tool 追加提交。

### 4.1 节点类型（Node Types）

所有节点共享字段：`id`、`created_by_run_id`、`created_by_turn_id`、`created_at`、`updated_at`、`status`。

| 类型 | 用途 | 关键属性 |
| --- | --- | --- |
| **Direction** | 用户可比较的主要方向 | `title`, `summary`, `confidence`, `maturity`, `status` |
| **Evidence** | 支持或反驳方向的信息 | `content`, `source`, `source_type`, `reliability` |
| **Claim** | 系统基于证据做出的断言 | `statement`, `validation_status` |
| **Decision** | 带有理由的判断点 | `description`, `chosen_option`, `alternatives`, `rationale` |
| **Unknown** | 已识别的待研究缺口 | `question`, `priority`, `resolution_status` |
| **Artifact** | 锚定到方向的物化产出 | `type`, `content_ref`, `version` |

### 4.2 边类型（Edge Types）

所有边共享字段：`id`、`from_id`、`to_id`、`kind`、`created_by_run_id`、`created_by_turn_id`、`created_at`。

| 边 | From → To | 含义 |
| --- | --- | --- |
| `supports` | Evidence/Claim → Direction | 支持该方向 |
| `contradicts` | Evidence/Claim → Direction | 挑战该方向 |
| `branches_from` | Direction → Direction | 子方向从父方向派生 |
| `competes_with` | Direction ↔ Direction | 替代方案 / 权衡关系 |
| `raises` | Evidence/Claim → Unknown | 引发该问题 |
| `resolves` | Evidence/Claim → Unknown | 回答该问题 |
| `justifies` | Evidence/Claim → Decision | 用于该判断 |
| `produces` | Direction → Artifact | 产出该物化产物 |

### 4.3 图变异契约（Graph Mutation Contract）

程序侧对 graph 的主写接口是批量追加工具：

```text
append_graph_batch(
  workspace_id,
  run_id,
  turn_id,
  nodes: []GraphNodeInput,
  edges: []GraphEdgeInput
)
```

约束：

- **只追加**：当前阶段只允许 `AddNode` / `AddEdge`，不允许硬删除、覆盖更新或整图替换
- **原子性**：一次调用内所有新增全成功或全失败
- **可追溯**：每次调用都必须关联 `run_id + turn_id` 并生成 trace / mutation 事件
- **唯一写入路径**：Agent 只能通过 graph tool 写入图，不允许绕过 runtime 直接改图
- **最小校验**：程序仅校验 workspace 归属、ID 唯一、边引用存在、类型合法

说明：

- graph 优化通过追加新节点/边表达，而不是程序侧执行 merge、rewrite 或 phase fallback
- 如果后续需要 `superseded` / `folded` 语义，应在节点状态层扩展，而不是恢复程序侧 graph 规则机

## 5. 方向生命周期（Direction Lifecycle）

### 5.1 生命周期状态

```text
emerging -> developing -> mature -> saturated -> folded
```

| 状态 | 含义 | Agent 默认倾向 |
| --- | --- | --- |
| `emerging` | 刚识别，证据少 | 偏 diverge + research |
| `developing` | 有证据，正在积极探索 | 平衡 research + structuring |
| `mature` | 充分支持，证据稳定 | 偏 converge + produce |
| `saturated` | 无有意义新增信息 | 降低资源投入 |
| `folded` | 暂停主资源投入但可重新激活 | 低优先观察 |

### 5.2 状态转换触发条件

- 生命周期推进由 `MainAgent` 根据当前图与 recent mutations 判断
- 程序侧不根据阈值自动改写方向状态
- `ControlAction` 可以改变资源优先级，但不直接改节点状态

## 6. 主执行链路（Run-Orchestration）

1. `CreateRun`：创建 run 与当前 workspace 快照
2. `AssembleContext`：装载 graph、balance、recent turns、recent mutations、memory、skill bindings、control actions
3. `StartTurn`：创建新 turn，并生成当前轮上下文摘要
4. `Reason`：`MainAgent` 分析当前局面并决定是否调用 graph / research / artifact / review 工具
5. `Integrate`：tool 在 runtime 内完成最小校验、原子落库、mutation 记录与广播
6. `Checkpoint`：在 turn 或关键结果后生成可恢复 checkpoint
7. `Project`：刷新 projection 并推送最近变化
8. `Evaluate`：判断继续下一 turn、进入 waiting、或完成 run
9. `Finalize`：run 进入 `completed / failed / cancelled`

补充约束：

- 新建 workspace 后立即触发首轮 run，由 `MainAgent` 追加后续 graph batch
- deterministic 逻辑仅保留在兼容/测试辅助路径，不再作为主运行时 graph 生长链路
- `MainAgent` 是唯一统一判断者；specialist 只能提供建议、研究结果或产物草稿

## 7. ControlAction、Intervention 与 Review 语义

### 7.1 控制动作统一模型

`Intervention` 不再被视为孤立对象，而是 `ControlAction` 的一个子类。

统一控制动作模型的价值：

- 前端可以做统一控制台
- 后端可以做统一审批与追踪
- projection 可以统一解释“最近哪些治理动作产生了效果”

### 7.2 Intervention 生命周期

`Intervention` 必须经过以下阶段：

- `received`
- `absorbed`
- `reflected`
- `rejected`

规则：

- intervention 改变的是 Agent 的上下文和关注重点，不是程序侧 plan graph
- 对用户返回的状态必须能回答“是否已反映到地图”
- 若当前 run 正在执行，可在同一 run 的下一 turn 吸收；否则在下一轮 run 生效

### 7.3 Review 语义

`ReviewRequest` 的目标不是产出普通文本，而是对当前 graph / decision / evidence 做结构化审查，至少包括：

- 反证与风险
- 证据缺口
- 过早收敛迹象
- 推荐下一步验证动作

## 8. Memory 与 Skill Binding 语义

### 8.1 Memory 分层

建议分为：

- `run working memory`：当前 run 的局部工作摘要
- `workspace memory`：长期探索中的稳定经验与结论
- `user preference memory`：用户偏好、风格、风险阈值与治理方式

规则：

- 并非所有 turn 内容都进入长期记忆
- 只有高价值、可复用、相对稳定的信息才应上升到 `workspace memory`
- preference memory 必须与显式用户行为或反复表达绑定

### 8.2 Skill Binding

推荐 skill 类型：

- `research_skill`
- `graph_refinement_skill`
- `artifact_skill`
- `review_skill`

规则：

- skill 默认不全量注入
- 只有任务相关时才装载
- skill 可以提供流程和模板，但不能直接修改 graph truth layer

## 9. Tool Risk Policy 与审批语义

吸收 `Codex` 的 sandbox / approval 思想后，建议把工具风险分成：

| 级别 | 示例 | 默认策略 |
| --- | --- | --- |
| `L0-observe` | 读 graph、读 projection、读 memory | 自动允许 |
| `L1-research` | 外部检索、抓文档、提摘要 | 自动允许并记日志 |
| `L2-graph-write` | 追加节点、边、decision、artifact ref | 自动允许，但必须 schema 校验与事件记录 |
| `L3-external-act` | 发通知、建 issue、调用外部生产系统 | workspace policy 控制，默认审批 |
| `L4-destructive` | 删除、覆盖、批量归档、高成本操作 | 始终审批或双确认 |

说明：

- `L2-graph-write` 不一定要求人工审批，但一定要求强审计
- tool risk policy 必须是持久化对象，而不是 prompt 内部约定

## 10. 错误模型与恢复边界

- Agent 运行失败：run 标记 `failed`，保留已成功写入的 graph 增量与 trace
- turn 运行失败：当前 turn 标记 `failed`，run 可重试或进入 `waiting`
- graph tool 校验失败：本次批量追加整体回滚，错误返回 Agent，并记录失败事件
- projection 刷新失败：不回滚图真相层，允许异步补投影
- 流式推送失败：客户端可用 `since_event_id` 补拉
- review 或 artifact worker 失败：记录为专项任务失败，不应污染已确认的 graph truth

恢复要求：

- 任何中断都可通过 `workspace + run + turn + mutation event + checkpoint` 恢复可解释状态
- 不允许出现“前端看到结果但后端无对应 run / turn / mutation 轨迹”
- 重启后 active workspace 可继续调度新 run，但不会伪造未完成 run 的 graph 输出

## 11. 接口契约来源（OpenAPI）

HTTP/WS 对外契约以以下文件为准：

- [idea-factory-openapi.yaml](./idea-factory-openapi.yaml)

本文件只定义语义，不重复字段级接口细节。

## 12. 能力清单验收（技术层）

- 可创建 run 并由 `MainAgent` 在 run 内自主追加 graph
- run 可拆为多个 `turn`，且每个 turn 可追溯
- graph tool 可原子追加 nodes / edges，并生成 mutation 事件
- intervention、review、artifact request 等控制动作可统一治理
- projection 可表达最近 graph 变化、turn 摘要与控制动作影响
- workspace memory 与 skill binding 能跨 run 生效
- checkpoint / resume 语义清晰
- 核心状态可追溯到 run、turn、mutation 与 policy 层

## 13. 一句话总结

技术内核必须是一套由 `MainAgent` 直接驱动 graph 生长、由 `turn` 级协议治理执行、由 `memory + skill + policy` 经营长期能力的自治 runtime。
