# Idea Factory 运行时与长期状态映射

> 版本：v0.2-draft
> 日期：2026-04-19
> 状态：已与新版 specs 对齐的运行时设计参考

## 1. 映射目标

`Idea Factory` 现在的目标态主链路已经明确更新为：

`workspace -> run -> turn -> graph mutation -> projection`

真正补强的重点有四件事：

- 把 `run` 从“后台任务”升级成“可治理协议”
- 把 `turn` 提升为最小解释单元
- 把 `workspace` 从“主题容器”升级成“长期能力容器”
- 把 `intervention` 升级成 `control_action` 家族中的结构化治理事件

## 2. 从 Claude Code 吸收什么：模式切换与任务恢复

`Claude Code` 的价值在于，它不把所有控制都塞进自然语言，而是显式提供：

- command surface
- task management
- resume / share / review
- plan mode
- remote / cron / trigger

映射到 `Idea Factory`，新版 specs 已经形成这些运行态对象：

- `run_mode`：当前 run 是偏探索、偏审查、偏产出、偏恢复
- `control_action`：用户发起的结构化控制动作，如暂停、继续、重定向、复盘
- `checkpoint`：允许从某个稳定状态继续推进
- `review_request`：用户要求系统解释为什么当前地图这样演进

这些对象不一定需要做成命令行 slash command，但已经应该成为 API 和前端控制台中的一等能力。

## 3. 从 Codex 吸收什么：Run / Turn / Checkpoint

### 3.1 推荐映射关系

| Codex 概念 | Idea Factory 对应 | 说明 |
| --- | --- | --- |
| `Session` | `Workspace runtime context` | 长期容器，但对本系统来说要 graph-first |
| `Task` | `Run` | 一次自治推进实例 |
| `Turn` | `RunTurn` | 一次模型推理 + 工具调用 + 结果集成 |
| `response_id` / bookmark | `run_checkpoint` | 用于恢复、分叉、解释 |
| `interrupt` | `control_action redirect` | 把后续 run / turn 的重心改向 |

### 3.2 为什么 `RunTurn` 必须显式化

如果没有 `RunTurn`，系统很快会失去解释能力。显式化后，每一轮都能回答：

- 这轮为什么开始
- 这轮读取了哪些 graph / memory / control action 摘要
- 这轮调用了哪些工具
- 这轮产生了哪些 graph delta
- 这轮为什么继续、等待或完成

新版 specs 已将 `RunTurn` 明确为运行治理的最小解释单元。

### 3.3 为什么 checkpoint 对 Idea Factory 很重要

`Idea Factory` 不是一次性写作文任务，而是长期探索系统。因此 checkpoint 的作用不只是故障恢复，还包括：

- 从一个高质量地图状态继续推进
- 在 intervention 或 review 后回看“改变是从哪里开始生效的”
- 对成熟分支做 artifact 物化时保留稳定参照点
- 支持后续的 review、comparison、branch replay

## 4. 从 Hermes 吸收什么：Workspace Memory 与 Skill Binding

### 4.1 Workspace 不是消息线程，而是长期能力容器

新版 specs 推荐把 `Workspace` 理解成这几类状态的统一宿主：

- 目标与约束
- graph truth layer
- 历史 run 与 turn 摘要
- 用户偏好与治理偏好
- 领域技能绑定
- 高价值证据与决策记忆

也就是说，`workspace` 至少要像 `Hermes session + memory + skill context` 的组合体，而不是简单的 topic bucket。

### 4.2 推荐的记忆分层

建议分成三层：

| 记忆层 | 用途 | 生命周期 |
| --- | --- | --- |
| `run working memory` | 当前 run 的局部工作摘要 | run 内有效 |
| `workspace memory` | 长期探索中的稳定经验与结论 | 跨 run 持续 |
| `user preference memory` | 用户对方向、风险、表达方式的偏好 | 跨 workspace 可选复用 |

这样做的价值是把短期推理噪声与长期有效经验分开。

### 4.3 推荐的 skill binding

建议把 skill 做成按需绑定对象，而不是长 prompt 常驻内容。

典型 skill 类型：

- `research skill`
- `graph refinement skill`
- `artifact skill`
- `review skill`

这类 skill 的关键不是“会的多”，而是：

- 任务相关时才装载
- 有自己的局部说明与质量门槛
- 只能通过受控 tool 修改真相层

## 5. ControlAction 家族为什么比单独的 Intervention 更合理

结合 `Claude Code` 的显式 command surface 与 `Codex` 的 interrupt 语义，`Idea Factory` 不应再把 `intervention` 视为唯一治理对象，而应把它放进统一的 `control_action` 家族：

- `intervention`
- `review_request`
- `artifact_request`
- `resume_request`
- `policy_adjustment`
- `memory_pin`

这样做后，系统就不再只是“听见了用户一句话”，而是“吸收了一条结构化治理命令”。

## 6. 子代理只适合在受限场景引入

`Hermes` 与 `Claude Code` 都展示了 multi-agent / coordinator 的可能性，但 `Idea Factory` 不该一开始就把系统建成 swarm。

推荐只在三类场景引入 specialist：

- 并行研究：不同外部源同步扫描
- 产物物化：artifact 生成与地图推进分离
- 审查验证：反证、风险、合规、缺口扫描

且必须遵守：

- specialist 默认不拥有 graph 直写特权
- specialist 输出必须回收给 `MainAgent`
- `MainAgent` 才是 workspace 当前真相层的统一解释者

## 7. 推荐新增的数据对象

如果要把三方模式系统化落地，新版 specs 已经把下列对象列为目标态核心：

- `run_turns`
- `run_checkpoints`
- `workspace_memories`
- `user_preference_memories`
- `workspace_skill_bindings`
- `control_actions`

这些对象不会推翻现有设计，反而会把已有 `workspace / run / graph / projection` 体系补得更完整。

## 8. 一句话总结

`Claude Code` 提醒我们把“控制动作”显式化，`Codex` 提醒我们把 run 拆到 turn 级治理，`Hermes` 提醒我们把 workspace 经营成长期能力容器；三者合起来，才是 `Idea Factory` 的完整运行时基线。
