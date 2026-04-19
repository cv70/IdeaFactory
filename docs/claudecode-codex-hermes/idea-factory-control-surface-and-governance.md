# Idea Factory 控制面、命令面与治理机制参考

> 版本：v0.2-draft
> 日期：2026-04-19
> 状态：已与新版 specs 对齐的控制面参考

## 1. 为什么 Idea Factory 需要单独设计控制面

`Idea Factory` 的默认运行方式是“系统主驾驶，用户高杠杆介入”。这种产品如果没有强控制面，会立刻出现三个问题：

- 用户不知道如何改变系统接下来几轮的行为
- 用户看不清当前 run 到底做了什么、为什么这样做
- 工程上很难界定什么是可自动执行的，什么应该停下来等用户判断

这一点上：

- `Claude Code` 提供了很强的显式控制面示范
- `Codex` 提供了很强的审批、工具治理与执行 trace 示范
- `Hermes` 提供了长期入口、多 session、skills、gateway 的扩展思路

因此 `Idea Factory` 需要的不是“更大的聊天框”，而是一套 `workspace control console`。

## 2. 控制面建议分成四层

### 2.1 Intent Layer：用户意图控制

参考 `Claude Code` 的命令面，建议把用户高频治理动作做成显式操作，而不是只靠自由文本：

- `Start Exploration`
- `Pause Auto Run`
- `Resume From Checkpoint`
- `Redirect Focus`
- `Request Review`
- `Generate Artifact`
- `Raise Evidence Standard`
- `Compare Branches`

这些操作可以在 UI 中表现成按钮、面板、快捷操作或命令输入，但语义层应统一成结构化 API。

### 2.2 Protocol Layer：执行协议控制

参考 `Codex`，建议这一层显式暴露：

- 当前 `run` 状态
- 当前 `turn` 编号与阶段
- 当前等待点是 `tool`、`approval`、`review`，还是 `idle`
- 当前 checkpoint 是否可恢复
- 当前 control action 是待吸收、已吸收还是已反映到 projection

这一层的价值是把自治过程变成“可观测协议”，而不是“不可解释黑箱”。

### 2.3 Policy Layer：风险与权限控制

参考 `Codex` 的 sandbox / approval 与 `Claude Code` 的 permission system，建议 `Idea Factory` 采用更贴近 graph 系统的风险分级：

| 级别 | 典型动作 | 默认策略 |
| --- | --- | --- |
| `L0-observe` | 读 graph、读 memory、读 projection | 自动允许 |
| `L1-research` | 检索、抓取、抽取、摘要 | 自动允许并记日志 |
| `L2-graph-write` | 新增 node、edge、decision、artifact ref | 自动允许，但必须结构化校验与追踪 |
| `L3-external-act` | 创建 issue、发通知、调用外部生产系统 | workspace policy 决定，默认审批 |
| `L4-destructive` | 删除、覆盖、批量归档、高成本动作 | 始终审批或双确认 |

`Idea Factory` 的特别之处在于：`L2-graph-write` 虽然不一定要求人工审批，但一定要求强审计。

### 2.4 Capability Layer：技能与入口扩展

参考 `Hermes` 的 skills / gateway 与 `Claude Code` 的 bridge / plugins，建议把以下能力设计成扩展点：

- workspace-specific skills
- domain skill packs
- external connectors
- scheduled triggers
- messaging / webhook entrypoints

但扩展点的底线是：

- 只能增强 context、strategy、execution reach
- 不能绕过主 runtime 直接写 graph truth layer

## 3. “命令面”在 Idea Factory 中应该长什么样

虽然 `Idea Factory` 未必采用 `/command` 交互，但建议保留命令面思想，把常见治理动作统一归入 `control_action`。

建议最小命令集合可对应为：

| 控制动作 | 语义 | 对应效果 |
| --- | --- | --- |
| `explore` | 启动或继续自治探索 | 创建/继续 run |
| `pause` | 暂停自动推进 | workspace 进入更保守态 |
| `resume` | 从 checkpoint 或最新稳定状态继续 | 创建 `resume_request` |
| `redirect` | 改变后续重点方向 | 创建 `intervention` |
| `review` | 要求做漏洞检查或路径复盘 | 创建 `review_request` |
| `artifact` | 将成熟方向转为产物 | 创建 `artifact_request` |
| `policy` | 调整审批、风险、资源约束 | 创建 `policy_adjustment` |
| `memory` | 查看/固定某些长期记忆 | 创建 `memory_pin` |

重点不在交互形式，而在这些动作必须有结构化含义，而不是留给 prompt 去猜。

## 4. 治理对象建议显式持久化

要让系统长期可解释，建议至少把以下对象作为一等持久化对象：

- `workspace_policy`
- `tool_policy`
- `approval_rule`
- `risk_event`
- `control_action`
- `skill_binding`
- `memory_scope`
- `checkpoint`

这样做的价值是：

- 能解释某次行为为什么被允许或阻止
- 能把治理偏好从单轮对话中抽出来
- 能支持 workspace 级模板化复用

## 5. 观测与追踪建议

参考 `Codex` 的 protocol trace 与 `Claude Code` 的 task / permission / command 体系，建议将以下事件做成一等事件流：

- `run_started`
- `turn_started`
- `control_action_received`
- `control_action_absorbed`
- `tool_called`
- `tool_succeeded`
- `tool_failed`
- `approval_requested`
- `approval_resolved`
- `graph_mutation_committed`
- `projection_published`
- `review_requested`
- `review_completed`
- `run_checkpointed`
- `run_completed`

事件要求：

- 全部可归因到 `workspace_id / run_id / turn_id / actor`
- projection 必须能反向引用其来源 mutation 与 decision
- UI 默认展示摘要，但必须允许下钻

## 6. 当前仓库的优先落点

建议优先修改或补强：

- `IdeaFactory/backend/domain/exploration/`：补充 `turn`、`checkpoint`、`control_action`、`review` 语义
- `IdeaFactory/backend/agentools/`：给 tool 增加风险级别与审计归因
- `IdeaFactory/backend/agents/`：把 `MainAgent` 与 specialist agent 的权限边界拉清楚
- `IdeaFactory/docs/superpowers/specs/idea-factory-technical-design.md`：补入 turn protocol、policy、control plane 语义
- `IdeaFactory/docs/superpowers/specs/idea-factory-openapi.yaml`：补齐控制面对象与状态枚举
- `IdeaFactory/frontend/src/types/`：补齐 control plane、turn、checkpoint、memory、skill 相关类型

## 7. 一句话总结

`Idea Factory` 需要的不只是自治能力，还需要“驾驭自治能力的控制台”：用户通过控制面发起结构化治理，runtime 按协议执行，policy 决定风险边界，最终所有结果都回收到 graph truth 与 trace 中。
