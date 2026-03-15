# Idea Factory 技术设计文档（Target State）

> 版本：v1-target
> 日期：2026-03-14
> 状态：目标态技术规范（后端内核优先）

## 1. 文档职责与边界

本文件定义后端内核的目标态：

- 领域对象与状态机
- `run -> plan -> task -> result -> projection` 主链路
- 平衡引擎策略级规则
- 失败与恢复语义

本文件不定义：

- 产品页面布局与交互文案（见产品设计）
- 基础设施选型、部署与系统容量细节（见系统架构）

## 2. 内核设计不变量

- `Workspace` 是顶层业务契约
- `Run` 是一次自治推进实例
- `ExecutionPlan` 必须显式存在并可演进
- `AgentTask` 是计划步骤的执行单元
- `Graph` 是方向结构真相层
- `Projection` 是前端读取面，不是写入面
- `Intervention` 触发重规划，不直接改写图真相

## 3. 核心领域模型与状态流转

### 3.1 Workspace

关键字段：`id`、`topic`、`goal`、`constraints`、`budget`、`status`。

状态流转：`draft -> active -> paused -> archived`。

### 3.2 Run

关键字段：`id`、`workspace_id`、`trigger_type`、`status`、`current_plan_id`、`started_at`、`finished_at`。

状态流转：`queued -> planning -> dispatching -> integrating -> projected -> completed | failed | cancelled`。

### 3.3 ExecutionPlan / PlanStep

- `ExecutionPlan`：`id`、`run_id`、`version`、`status`。
- `PlanStep`：`id`、`plan_id`、`order`、`kind`、`assigned_agent`、`status`。

Plan 状态：`draft -> active -> superseded | completed | failed`。

Step 状态：`todo -> doing -> done | failed | skipped | invalidated`。

### 3.4 AgentTask / Result

- `AgentTask`：`id`、`step_id`、`agent_role`、`status`、`attempt`、`error_code`。
- `AgentTaskResultSummary`：`task_id`、`evidence_refs`、`claim_refs`、`unknown_refs`、`decision_delta`。

Task 状态：`queued -> running -> succeeded | failed | timeout`。

### 3.5 BalanceState

关键字段：

- `diverge_converge_mode`
- `research_produce_mode`
- `aggressive_prudent_mode`
- `reason`（本轮调节解释）

状态取值：`diverge|converge`、`research|produce`、`aggressive|prudent`。

### 3.6 Projection

关键字段：`graph_snapshot`、`focus_branches`、`recent_changes`、`run_summary`、`intervention_effects`。

约束：Projection 只读可重建；任何写入都必须经 run 主链路产生。

## 4. 图本体定义（Graph Ontology）

Graph 是方向结构的真相层。所有对方向、证据、决策的写入必须经结构化 mutation 批次提交。

### 4.1 节点类型（Node Types）

所有节点共享字段：`id`、`created_by_run_id`、`created_by_task_id`、`created_at`、`updated_at`、`status`。

| 类型 | 用途 | 关键属性 |
| --- | --- | --- |
| **Direction** | 用户可比较的主要方向 | `title`, `summary`, `confidence`, `maturity`, `status` |
| **Evidence** | 支持或反驳方向的信息 | `content`, `source`, `source_type` (research / user_input / inference), `reliability` |
| **Claim** | 系统基于证据做出的断言 | `statement`, `validation_status` (hypothetical / supported / refuted) |
| **Decision** | 带有理由的判断点 | `description`, `chosen_option`, `alternatives`, `rationale` |
| **Unknown** | 已识别的待研究缺口 | `question`, `priority`, `resolution_status` (open / investigating / resolved) |
| **Artifact** | 锚定到方向的物化产出 | `type`, `content_ref`, `version` |

### 4.2 边类型（Edge Types）

所有边共享字段：`id`、`from_id`、`to_id`、`kind`、`created_by_run_id`、`created_at`。

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

```
MutationBatch {
  run_id:    string
  task_id:   string
  operations: [
    AddNode    { type, properties }
    UpdateNode { id, delta }
    FoldNode   { id }
    AddEdge    { from, to, kind }
    RemoveEdge { from, to, kind }
  ]
}
```

约束：

- **原子性**：批次内所有操作全成功或全失败
- **可追溯**：每个批次产生 trace 事件
- **唯一写入路径**：仅 Integration Pipeline 可提交 mutation 批次
- **软删除**：节点永不硬删除；`FoldNode` 将状态设为 `folded`

## 5. 方向生命周期（Direction Lifecycle）

### 5.1 生命周期状态

```
emerging → developing → mature → saturated
                ↑ (unfold)
    folded ←────┘ (fold/suppress)
```

| 状态 | 含义 | 平衡引擎行为 |
| --- | --- | --- |
| `emerging` | 刚识别，证据少 | 偏 diverge + research |
| `developing` | 有证据，正在积极探索 | 平衡 research + structuring |
| `mature` | 充分支持，证据稳定 | 偏 converge + produce |
| `saturated` | 无有意义新增信息 | 停止投入，考虑 folding |
| `folded` | 已压制，不再分配资源 | 跳过，除非发现新矛盾证据 |

### 5.2 状态转换触发条件

- `emerging → developing`：关联 evidence 数量超过阈值 或 N 个 research task 涉及该方向
- `developing → mature`：置信度达标 且 近期 run 无新矛盾证据
- `mature → saturated`：连续 run 未产生有意义新增证据
- `any → folded`：用户干预 或 平衡引擎压制
- `folded → developing`：用户 unfold 干预 或 发现新矛盾证据

## 6. 主执行链路（Run-Orchestration）

1. `CreateRun`：创建 run 与初始上下文
2. `Plan`：主代理读取 workspace/graph/balance 生成 `ExecutionPlan`
3. `Dispatch`：按 step 派发 `AgentTask`
4. `Execute`：子代理经受控工具层执行并返回结构化结果
5. `Integrate`：主代理合并结果，产出 `decision/evidence/unknown` 变化
6. `Project`：刷新 projection 并推送状态摘要
7. `Finalize`：run 进入 `completed/failed/cancelled`

## 7. Intervention 与重规划语义

`Intervention` 必须经过以下阶段：

- `received`：已写入待处理
- `absorbed`：被当前或下一轮 run 吸收
- `replanned`：形成新 plan version
- `reflected`：projection 出现可见变化

规则：

- 干预到达后，当前 plan 的未执行步骤可被 `invalidated/skipped`
- 新 plan 必须关联触发干预 ID
- 对用户返回的状态必须能回答“是否已反映到地图”

## 8. 平衡引擎策略级规则

### 8.1 输入信号

- 方向成熟度分布（各 lifecycle 状态的方向数量）
- 最近证据密度
- 重复路径比例
- 干预频率与方向偏置
- 连续 run 的增量价值

### 8.2 决策规则

- `count(Direction where maturity=emerging)` 占比过高且证据稀薄：偏 `diverge + research`
- `count(Direction where maturity=developing)` 过多但可比性弱：偏 `converge + research`
- `count(Direction where maturity=mature)` 占主导且证据充分：偏 `converge + produce`
- 连续停滞（`count(Direction where maturity=saturated)` 上升）且用户接受探索风险：偏 `aggressive`
- 已有高置信路径（`exists(Direction where confidence > threshold AND maturity=mature)`）：偏 `prudent`

### 8.3 回退规则

- 任一模式连续 N 轮无增量，自动回退到中性组合
- 出现高风险失败（工具不可用、关键任务超时）时强制 `prudent`

## 9. 错误模型与恢复边界

- 任务级失败：重试不超过上限，失败留痕，不阻断全局 run
- 计划级失败：run 标记 `failed`，保留中间可追溯状态
- 投影刷新失败：不回滚图真相层，允许异步补投影
- 流式推送失败：客户端可用 `since_event_id` 补拉

恢复要求：

- 任何中断都可通过 `run + plan + task + event` 恢复可解释状态
- 不允许出现“前端看到结果但后端无对应 run 轨迹”

## 10. 接口契约来源（OpenAPI）

HTTP/WS 对外契约以以下文件为准：

- [idea-factory-openapi.yaml](./idea-factory-openapi.yaml)

本文件只定义语义，不重复字段级接口细节。

## 11. 能力清单验收（技术层）

- 可创建 run 并产生显式 plan 与 step
- step 可驱动 task 执行并产出结构化 summary
- intervention 可触发 plan version 变化
- projection 可表达最近变化与干预效果
- 核心状态可追溯到 run/plan/task 层

## 12. 一句话总结

技术内核必须是可重规划、可追溯、可恢复的 run 编排系统，而不是隐式循环逻辑。
