# Idea Factory 技术设计文档（Target State）

> 版本：v1-target
> 日期：2026-03-18
> 状态：目标态技术规范（后端内核优先）

## 1. 文档职责与边界

本文件定义后端内核的目标态：

- 领域对象与状态机
- `run -> agent session -> graph mutation -> projection` 主链路
- 程序侧与 Agent 侧的职责边界
- 失败与恢复语义

本文件不定义：

- 产品页面布局与交互文案（见产品设计）
- 基础设施选型、部署与系统容量细节（见系统架构）

## 2. 内核设计不变量

- `Workspace` 是顶层业务契约
- `Run` 是一次自治推进实例
- `MainAgent` 是 graph 生长与优化的唯一决策者
- `Graph` 是方向结构真相层
- `Projection` 是前端读取面，不是写入面
- `Intervention` 只改变 Agent 后续关注点，不直接改写图真相
- 程序侧只保留调度、最小校验、持久化、广播与恢复，不内置 graph 阶段策略

## 3. 核心领域模型与状态流转

### 3.1 Workspace

关键字段：`id`、`topic`、`goal`、`constraints`、`budget`、`status`。

状态流转：`draft -> active -> paused -> archived`。

### 3.2 Run

关键字段：`id`、`workspace_id`、`trigger_type`、`status`、`started_at`、`finished_at`。

状态流转：`queued -> running -> completed | failed`。

说明：

- `running` 阶段内由 `MainAgent` 自主决定是否继续研究、补充、收敛或产出。
- 程序侧不再显式维护 `ExecutionPlan` / `PlanStep` 作为 graph 生长前置条件。

### 3.3 Agent Session

关键字段：`runs`、`agent_tasks`、`results`、`mutations`、`balance`、`latest_replan_reason`。

职责：

- 为 `MainAgent` 提供当前 workspace、graph、最近变更与 intervention 上下文
- 记录主运行时的活动摘要，而不是 planner/step 结构
- 作为恢复和审计的最小运行态边界

### 3.4 BalanceState

关键字段：

- `diverge_converge_mode`
- `research_produce_mode`
- `aggressive_prudent_mode`
- `reason`（本轮调节解释）

当前实现字段：`divergence`、`research`、`aggression`、`reason`、`updated_at`。

说明：

- `BalanceState` 是给 Agent 的节奏提示，不是程序侧 graph 规则机。
- 程序可以更新 balance，但不能依据 balance 直接决定新增哪类节点或边。

### 3.5 Projection

关键字段：`graph_snapshot`、`focus_branches`、`recent_changes`、`run_summary`、`intervention_effects`。

约束：Projection 只读可重建；任何写入都必须经 run 主链路产生。

## 4. 图本体定义（Graph Ontology）

Graph 是方向结构的真相层。所有 graph 变化必须由 Agent 通过受控 graph tool 追加提交。

### 4.1 节点类型（Node Types）

所有节点共享字段：`id`、`created_by_run_id`、`created_at`、`updated_at`、`status`。

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

程序侧对 graph 的主写接口是批量追加工具：

```text
append_graph_batch(
  workspace_id,
  nodes: []GraphNodeInput,
  edges: []GraphEdgeInput
)
```

约束：

- **只追加**：当前阶段只允许 `AddNode` / `AddEdge`，不允许硬删除、覆盖更新或整图替换
- **原子性**：一次调用内所有新增全成功或全失败
- **可追溯**：每次调用都必须关联 `run_id` 并生成 trace / mutation 事件
- **唯一写入路径**：Agent 只能通过 graph tool 写入图，不允许绕过 runtime 直接改图
- **最小校验**：程序仅校验 workspace 归属、ID 唯一、边引用存在、类型合法

说明：

- graph 优化通过追加新节点/边表达，而不是程序侧执行 merge、rewrite 或 phase fallback。
- 如果后续需要 `superseded` / `folded` 语义，应在节点状态层扩展，而不是恢复程序侧 graph 规则机。

## 5. 方向生命周期（Direction Lifecycle）

### 5.1 生命周期状态

```text
emerging -> developing -> mature -> saturated
```

| 状态 | 含义 | Agent 默认倾向 |
| --- | --- | --- |
| `emerging` | 刚识别，证据少 | 偏 diverge + research |
| `developing` | 有证据，正在积极探索 | 平衡 research + structuring |
| `mature` | 充分支持，证据稳定 | 偏 converge + produce |
| `saturated` | 无有意义新增信息 | 降低资源投入，优先关注其他方向 |

### 5.2 状态转换触发条件

- 生命周期推进由 `MainAgent` 根据当前图与 recent mutations 判断
- 程序侧不根据阈值自动改写方向状态
- Intervention 可以改变 Agent 对某些方向的投入优先级，但不直接改节点状态

## 6. 主执行链路（Run-Orchestration）

1. `CreateRun`：创建 run 与当前 workspace 快照
2. `AssembleContext`：把 graph、balance、recent mutations、intervention 摘要注入 `MainAgent`
3. `Execute`：`MainAgent` 自主分析当前局面，并按需调用 graph / research / artifact 等受控工具
4. `Integrate`：graph tool 在 runtime 内完成最小校验、原子落库、mutation 记录与广播
5. `Project`：刷新 projection 并推送最近变化
6. `Finalize`：run 进入 `completed/failed`

补充约束：

- 新建 workspace 后立即触发首轮 run，由 `MainAgent` 追加后续 graph batch
- 程序侧仍会创建最小 workspace seed graph，后续生长不再依赖 planner/step
- deterministic 逻辑仅保留在兼容/测试辅助路径，不再作为主运行时 graph 生长链路

## 7. Intervention 与重规划语义

`Intervention` 必须经过以下阶段：

- `received`：已写入待处理
- `absorbed`：被当前或下一轮 run 吸收
- `reflected`：projection 出现可见变化

规则：

- intervention 改变的是 Agent 的上下文和关注重点，不是程序侧 plan graph
- 对用户返回的状态必须能回答“是否已反映到地图”
- 若当前 run 正在执行，可由 `MainAgent` 在同一 run 内吸收；否则在下一轮 run 生效

## 8. Balance 引擎语义

### 8.1 输入信号

- 方向成熟度分布
- 最近证据密度
- 重复路径比例
- 干预频率与方向偏置
- 连续 run 的增量价值

### 8.2 输出用途

- 作为 `MainAgent` 的节奏提示，影响其偏向 `diverge/converge`、`research/produce`、`aggressive/prudent`
- 作为调度层判断是否继续自动推进的参考信号

### 8.3 边界

- 程序侧不得把 balance 直接翻译成“本轮必须新增某类节点”的硬规则
- Agent 失败时程序侧可以切换到更保守的 balance 提示，但不直接代替 Agent 生长图

## 9. 错误模型与恢复边界

- Agent 运行失败：run 标记 `failed`，保留已成功写入的 graph 增量与 trace
- graph tool 校验失败：本次批量追加整体回滚，错误返回 Agent，并记录失败事件
- 投影刷新失败：不回滚图真相层，允许异步补投影
- 流式推送失败：客户端可用 `since_event_id` 补拉

恢复要求：

- 任何中断都可通过 `run + graph + mutation event` 恢复可解释状态
- 不允许出现“前端看到结果但后端无对应 run / mutation 轨迹”
- 重启后 active workspace 可继续调度新 run，但不会伪造未完成 run 的 graph 输出

## 10. 接口契约来源（OpenAPI）

HTTP/WS 对外契约以以下文件为准：

- [idea-factory-openapi.yaml](./idea-factory-openapi.yaml)

本文件只定义语义，不重复字段级接口细节。

## 11. 能力清单验收（技术层）

- 可创建 run 并由 `MainAgent` 在 run 内自主追加 graph
- graph tool 可原子追加 nodes / edges，并生成 mutation 事件
- intervention 可改变后续 run 的 graph 生长重心
- projection 可表达最近 graph 变化与干预效果
- 核心状态可追溯到 run 与 mutation 层

## 12. 一句话总结

技术内核必须是一个由 `MainAgent` 直接驱动 graph 生长的自治 runtime，而不是程序侧维护阶段策略的 planner 系统。
