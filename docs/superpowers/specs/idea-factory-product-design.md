# Idea Factory 产品设计文档（Target State）

> 版本：v1.1-target
> 日期：2026-04-19
> 状态：目标态产品规范（不描述实现细节）

## 1. 文档职责与边界

本文件回答四个产品层问题：

- 用户为什么要用 `Idea Factory`
- 用户如何围绕 `workspace` 与 `方向地图` 完成长期探索治理
- 用户如何通过控制面驾驭系统，而不是被系统黑盒牵着走
- 产品成功如何被衡量

本文件不定义：

- `run / turn / checkpoint / memory / skill binding` 的内部实现细节
- 子系统部署、存储一致性、故障恢复策略
- API 字段与错误码

对应内容见：

- 技术设计：[idea-factory-technical-design.md](./idea-factory-technical-design.md)
- 系统架构：[idea-factory-system-architecture.md](./idea-factory-system-architecture.md)
- OpenAPI 契约：[idea-factory-openapi.yaml](./idea-factory-openapi.yaml)

## 2. 产品目标

`Idea Factory` 的产品目标是：让用户在长期存在的 `workspace` 中，把模糊主题持续推进为可比较、可追溯、可干预、可继续深挖的 `方向地图`。

产品不是：

- 聊天式答案生成器
- 一次性脑暴列表工具
- 通用图编辑器
- 终端 agent shell 的换皮版本

## 3. 核心产品心智

### 3.1 三个一等对象

产品层只应把以下三个对象做成强心智：

- `workspace`：长期探索容器
- `方向地图`：当前真相层的可视化投影
- `控制台`：用户治理自治系统的显式入口

这组心智分别吸收了三类参考模式的具体架构元素：

- **`Claude Code` → 控制面 (L1)**：显式控制表面、命令系统、模式切换、治理入口
- **`Codex` → 执行协议层 (L2)**：任务-回合协议、工具风险策略、检查点机制、不可变追踪
- **`Hermes` → 长期能力层 (L3)**：持久会话记忆、技能系统、用户偏好模型、跨运行上下文

### 3.2 产品核心承诺

用户不需要手动搭图，也不需要盯着每一轮执行；但系统必须让用户始终知道：

- 现在在探索什么
- 为什么往这个方向探索
- 我如何改变接下来几轮系统行为
- 系统是否真的吸收了我的治理动作

## 4. 用户角色与职责

### 4.1 用户角色

用户是 `workspace` 的委托者与治理者，核心职责：

- 提供主题、目标、约束、预算与偏好
- 在运行中提交高杠杆 `intervention` 或其他控制动作
- 对地图分支做保留、压制、继续深挖、转产物等判断
- 在关键时点要求系统做 review、comparison、artifact materialization

### 4.2 系统角色

系统默认主驾驶，核心职责：

- 持续推进探索
- 维护方向结构的可解释性
- 在用户干预后调整后续探索重点
- 沉淀长期记忆、技能绑定与工作上下文

## 5. 核心闭环

主闭环更新为：

`模糊主题 -> 自治探索 -> 方向地图 -> 用户控制/干预 -> 新一轮探索 -> 产物或复盘`

其中对用户可见的关键结果应始终包括：

- 当前有哪些方向
- 方向之间如何比较
- 最近哪些变化最重要
- 系统当前正在推进什么
- 干预后系统是否已吸收并改变重心
- 哪些方向已经接近可产出状态

## 6. 主界面信息架构（Map-First + Control Console）

进入 `workspace` 后默认界面建议包含五层：

1. 地图主画布：方向结构、热点分支、折叠弱方向；方向节点通过成熟度标识（`emerging / developing / mature / saturated / folded`）表达生命周期阶段
2. 运行状态层：当前 `run` 是否活跃、当前 `turn` 所处阶段、系统在等待什么
3. 最近变化层：近期新增证据、方向变化、关键 decision、artifact 生成与 intervention 影响
4. 分支详情层：该方向的依据、判断、未知问题、相关 artifact、最近几轮变化原因
5. 控制台入口层：发起 `explore / pause / resume / redirect / review / artifact / policy / memory` 等治理动作

关键要求：

- `方向地图` 是主界面
- `控制台` 是主操作入口
- `run` 与 trace 是解释层，不是首页主视图

## 7. 控制面产品契约（用户视角）

参考 `Claude Code` 的 command surface 思路，`Idea Factory` 需要把高频治理动作产品化成明确操作，而不是只依赖自由文本输入。

建议最小控制动作集合（直接映射自 Claude Code 的控制范式）：

- `Start Exploration` ↔ Claude Code's `/agent` / autonomous mode initiation
- `Pause Auto Run` ↔ Claude Code's plan mode toggle / session interruption
- `Resume From Checkpoint` ↔ Claude Code's `/resume` command functionality
- `Redirect Focus` ↔ Claude Code's intervention system for changing agent behavior
- `Raise Evidence Standard` ↔ Claude Code's `/review` for adjusting confidence thresholds
- `Request Review` ↔ Claude Code's `/review` command for process examination
- `Compare Branches` ← Claude Code's session comparison and branching concepts
- `Generate Artifact` ↔ Claude Code's `/share` / output materialization
- `Pin Memory` ← Claude Code's `/memory` persistent storage system
- `Adjust Policy` ← Claude Code's `/config` for runtime behavior modification

这些动作的交互形态可以是按钮、命令框、侧栏面板或快捷入口，但语义必须结构化。

## 8. Intervention 与控制动作产品契约

用户发出干预后，产品至少必须返回四类反馈（源自 Claude Code 的控制反馈模型）：

- `已接收`：系统是否接受了该动作（Command acknowledgment）
- `已吸收`：系统是否把该动作纳入后续运行上下文（State integration confirmation）
- `已反映`：地图、状态或产物层出现了哪些变化（Visible effect demonstration）
- `后续影响`：接下来系统将如何调整探索或审查重心（Future behavior prediction）

如果这四类反馈缺失，用户会把系统视为黑盒。

说明：

- `intervention` 是控制动作的一种，重点改变后续探索重点
- `review request`、`artifact request`、`policy adjustment` 等属于同一控制面家族
- 产品层应统一把它们展示为“治理动作”，而不是散落在不同视图里

## 9. Review、Resume 与 Artifact 的产品位置

吸收 `Claude Code` 与 `Codex` 的经验，以下三类动作必须是一等用户能力：

### 9.1 Review

用户需要能要求系统回答：

- 为什么当前地图这样演进（Causal explanation）
- 某一方向的证据是否足够（Evidence adequacy assessment）
- 当前结论可能遗漏了哪些反证或风险（Contradiction scanning）

这直接映射自 Claude Code 的 `/review` 系统，但应用于方向地图而非代码。

### 9.2 Resume

用户需要能从稳定 checkpoint 或最新一致状态继续，而不是只能重新发起一个模糊 run。

这直接映射自 Claude Code 的 `/resume` 系统和 Codex 的 checkpoint-recovery 协议。

### 9.3 Artifact

当方向成熟后，用户需要一键把方向转成产物，而不是离开地图另开一个工作流。

这结合了 Claude Code 的输出分享机制和 Codex 的 artifact 生成协议。

## 10. 长期连续性产品契约

吸收 `Hermes` 的 session / memory / skills 思想后，产品层需要明确以下长期能力：

- `workspace` 会保留历史 run 的高价值结论（Persistent knowledge base）
- 系统会记住用户在这个 workspace 中反复强调的偏好与标准（User preference learning）
- 常用探索套路会沉淀为 skill，并在相关任务中自动装载（Skill acquisition and application）
- 新一轮探索不应表现得像失忆重来（Cross-run context continuation）

产品上应至少让用户看到：

- 当前有哪些长期记忆在生效（Active memory display）
- 当前有哪些 skill 或工作模板被装载（Loaded skill inventory）
- 某一条偏好或政策是从何时开始影响系统行为的（Preference timeline）

## 11. 北极星与能力验收（产品层）

北极星：用户是否持续沿同一 `workspace` 深挖高价值方向，而不是反复重开主题。

产品层能力验收：

- 能看到至少 2-3 条可比较方向，而非一堆平铺想法（Structured exploration vs. brainstorming）
- 能解释方向变化，不只展示新内容（Change justification vs. mere addition）
- 用户可以显式地暂停、继续、转向、审查、转产物（Explicit control availability）
- 干预后能观察到可见结构变化与后续重心变化（Control absorption verification）
- 用户能看见系统当前正在做什么，而不是只看到最终结果（Process transparency）
- 能观察到方向生命周期推进（`emerging -> developing -> mature`）
- 能感知系统存在长期记忆与 skill，而不是每轮都像首次运行（Long-term capability evidence）

## 12. 术语主入口与引用规则

产品默认术语：`workspace`、`方向地图`、`控制台`、`intervention`、`artifact`、`review`。

以下术语仅作解释引用，不在本文件定义：`run`、`turn`、`checkpoint`、`control_action`、`workspace_memory`、`skill_binding`、`policy`、`trace_summary`。

这些术语的定义唯一来源为技术设计与 OpenAPI。

## 13. 一句话总结

`Idea Factory` 的产品面必须始终围绕“用户治理方向地图 + 用户通过控制台驾驭自治系统”，而不是让用户直接面对一个难以预测的运行内核。

这一定位直接来源于三层架构的融合：
- Claude Code 提供控制面范式（用户如何真正控制系统）
- Codex 提供执行协议范式（系统如何安全可追溯地执行）
- Hermes 提供长期能力范式（系统如何随时间学习和改进）