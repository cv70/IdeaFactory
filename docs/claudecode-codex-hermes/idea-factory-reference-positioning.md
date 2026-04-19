# Idea Factory 对 Claude Code / Codex / Hermes 的定位与吸收边界

> 版本：v0.2-draft
> 日期：2026-04-19
> 状态：已与新版 specs 对齐的架构定位说明

## 1. 为什么要同时参考三个系统

`Idea Factory` 当前要解决的问题，不是“怎么把模型包成一个聊天 UI”，也不是“怎么做一次性任务自动化”，而是：

- 在长期存在的 `workspace` 中持续推进探索
- 把探索过程沉淀成 `graph truth layer`
- 允许用户以较低频但高杠杆的方式介入方向
- 让系统越跑越懂当前主题、当前用户、当前策略

这要求系统同时具备三种能力：

- 强控制面
- 强执行协议
- 强长期能力

这也是新版 `docs/superpowers/specs/` 最终采用 `控制面 + 执行协议层 + 长期能力层` 三层主线的原因。

## 2. 三个参考系各自代表什么

### 2.1 Claude Code：控制面与操作模式设计

从本地 `claudecode/README.md` 可提炼出几个重要特征：

- 工具系统、命令系统、权限系统是明确分层的
- `/review`、`/memory`、`/skills`、`/resume`、`/tasks` 等命令是用户控制自治系统的显式入口
- 存在 `plan mode`、`remote sessions`、`cron triggers`、`plugin`、`bridge` 等控制面能力
- `memory directory`、`team memory sync`、`coordinator` 表明系统不只是单轮问答器

对 `Idea Factory` 的启发不是去复制 slash command，而是理解一件事：

`一个长期自治系统必须有显式控制面，否则用户无法低成本、可预测地驾驭它。`

对应到新版 specs，这层已经被收束为：

- `控制台`
- `control_action`
- `review / resume / artifact / policy / memory` 统一入口

### 2.2 Codex：受控执行协议与工具纪律

从 `codex/README.md`、`codex/AGENTS.md`、`codex-rs/docs/protocol_v1.md` 可提炼出几个关键词：

- `task / turn` 执行协议清晰
- `sandbox / approval` 是协议层的一等对象
- 工具调用、文件修改、命令执行都在强治理边界内
- runtime 与具体 UI 解耦，可被不同入口驱动
- 指令作用域、工作目录说明、输出格式约束都被制度化

对 `Idea Factory` 的启发是：

- `Run` 必须是可恢复、可追踪、可中断的协议对象
- `RunTurn` 必须成为最小解释单元
- 所有 graph 写入都必须像提交补丁一样被显式记录
- “工具能不能做”与“系统该不该做”必须由 policy 控制，而不是由 prompt 临场发挥

对应到新版 specs，这层已经被收束为：

- `run -> turn -> checkpoint`
- `waiting / approval / trace`
- `tool risk policy`

### 2.3 Hermes Agent：长期会话、记忆、技能与多入口

从 `hermes-agent/README.md` 与 `hermes-agent/docs/honcho-integration-spec.md` 可以提炼出：

- `persistent sessions`
- `memory`
- `skills`
- `subagents`
- `gateway`
- `async context prefetch`

对 `Idea Factory` 的启发是：

- `workspace` 必须是长期容器，而不是一条临时消息线程
- `MainAgent` 需要历史经验、用户偏好、策略记忆，而不只是本轮 prompt
- 高复用探索流程应该沉淀成 skill，而不是永久堆进系统提示词

对应到新版 specs，这层已经被收束为：

- `workspace_memory`
- `user_preference_memory`
- `skill_binding`

## 3. 三者的关系：不是替代，而是三层拼装

最适合 `Idea Factory` 的理解方式如下：

| 参考系 | 更像什么 | 对 Idea Factory 的可吸收层 |
| --- | --- | --- |
| `Claude Code` | 控制面 / 操作面 | 用户如何发起、切换、介入、审阅、恢复、治理 run |
| `Codex` | 执行协议 / 工具纪律 | run、turn、tool、approval、checkpoint、trace |
| `Hermes` | 长期能力层 | workspace memory、skills、session continuity、subagents |

换句话说：

- `Claude Code` 让我们理解“用户怎样真正控制 agent”
- `Codex` 让我们理解“agent 怎样真正可治理地执行”
- `Hermes` 让我们理解“系统怎样真正形成长期能力”

## 4. Idea Factory 不该直接复制什么

### 4.1 不复制 Claude Code 的主交互形态

`Claude Code` 的主心智是终端内的 agent shell。`Idea Factory` 的主心智应该始终是 `方向地图 + workspace control console`。

因此：

- 可以吸收命令面、模式切换、任务恢复这些思想
- 但不应让 shell transcript 成为产品主界面

### 4.2 不复制 Codex 的代码任务中心形态

`Codex` 天然围绕本地代码任务、文件、补丁、命令与审批展开。`Idea Factory` 虽然也会调用研究、文件、代码、artifact 工具，但其中心是真相层图谱，而不是 patch queue。

因此：

- 可以吸收 task-turn 协议与工具纪律
- 但不应把整个系统产品化成“更会写代码的命令行代理”

### 4.3 不复制 Hermes 的通用个人助理形态

`Hermes` 的长期方向更接近广义 agent shell / assistant OS。`Idea Factory` 则是围绕“探索、判断、方向管理、artifact 物化”的专用系统。

因此：

- 可以吸收长期会话、记忆、技能、gateway
- 但不应演化成一个泛聊天或泛消息代理中枢

## 5. 对 Idea Factory 的推荐三层分工

### 5.1 L1：控制面与治理面

这一层更多参考 `Claude Code`。

职责：

- 提供可显式操控的用户入口
- 提供模式切换、介入、审阅、恢复、调度等能力
- 把“用户想怎样掌控系统”从 prompt 中拿出来，做成结构化操作

### 5.2 L2：执行协议层

这一层更多参考 `Codex`。

职责：

- 定义 `workspace / run / turn / tool / checkpoint / approval` 的硬边界
- 保证每次高影响动作都可追溯
- 保证 run 在失败、中断、恢复时语义清晰

### 5.3 L3：长期能力层

这一层更多参考 `Hermes`。

职责：

- 管理 workspace memory、user preference memory、skill binding
- 支持跨 run 的策略延续
- 在必要时支持 specialist / subagent 协作

## 6. 最终定位

最合理的定位不是“Idea Factory 借鉴了三个 agent 产品”，而是：

- 用 `Claude Code` 的思路设计控制台
- 用 `Codex` 的思路治理执行
- 用 `Hermes` 的思路经营长期能力
- 用 `Idea Factory` 自己的 graph-first 语义定义产品真相层

一句话总结：`Claude Code` 管人机操控，`Codex` 管执行纪律，`Hermes` 管长期进化，而 `Idea Factory` 负责把三者收束为一套面向探索地图的专用 operating system。
