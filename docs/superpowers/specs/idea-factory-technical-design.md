# Idea Factory 技术设计文档

> 版本：v1
> 日期：2026-03-13
> 状态：当前技术架构基线

更细的系统分层协作、关键时序、数据职责和非功能约束，见 [《Idea Factory 系统架构设计文档》](./idea-factory-system-architecture.md)。

## 1. 技术目标与设计不变量

v1 技术设计服务于一个明确目标：支撑一个以 `workspace` 为核心容器、以系统主驾驶为默认运行方式、以 `方向地图` 为主投影的自治探索系统。

必须成立的技术不变量：

- `Graph is the source of truth`：系统判断、证据和方向结构必须进入图，而不是散落在 prompt、日志或临时缓存里。
- `Workspace is the primary contract`：顶层产品对象是 `workspace`，不是一次性 `session` 或单次 `run`。
- `Runtime is event-driven`：持续自治、随时干预和可追溯增长都建立在 `event-driven state machine` 之上。
- `External behavior is streaming`：产品层应看到持续运行与节点级变化，而不是整图重算式刷新。
- `Frontend consumes projections`：前端主要消费 `projection`，不直接承担原始图结构的拼装责任。
- `Intervention is high-level intent`：用户输入的是治理意图，系统负责把它翻译成运行策略与图更新。

## 2. 顶层系统分层

系统建议拆为六层，但它们的边界要围绕产品语义而不是纯技术栈组织。

### 2.1 Interface Layer

负责提供：

- Web workbench
- API
- SDK / CLI
- 实时订阅或事件消费入口

这一层只暴露稳定能力，不泄露底层存储细节。

### 2.2 Application Layer

负责：

- `workspace` 生命周期
- 运行配置、预算和权限
- 任务调度入口
- `intervention` 接收与编排入口
- 对外接口聚合

### 2.3 Runtime Layer

负责：

- 解释目标与约束
- 驱动 `event-driven state machine`
- 选择下一步探索策略
- 触发研究、图变更、评估和物化
- 管理长时间运行与中途干预吸收

### 2.4 Graph Intelligence Layer

负责：

- 节点与边的读写
- 语义去重、聚类、压缩
- 路径与方向评分
- `decision`、`evidence`、`unknown` 的结构化挂接
- 快照、分支、重放和 `projection` 生成

### 2.5 Research & Context Layer

负责：

- 外部搜索或研究源接入
- 用户文档与资料摄取
- 内容切片、归一化和抽取
- 把输入变成可挂接的 `evidence` / `claim` / `unknown`

### 2.6 Storage & Streaming Layer

负责：

- 结构化元数据持久化
- 图数据和 mutation 历史
- 原始资料与物化结果存储
- 事件日志、异步任务和前端流式同步

## 3. 顶层领域模型

### 3.1 `Workspace`

`workspace` 是顶层容器，承载：

- 主题与目标
- 约束、预算和偏好
- 可用上下文源
- 当前地图状态
- 历史 `run`
- 已记录的 `intervention`
- 已物化的 `artifact`

### 3.2 `Run`

`run` 是一次自治推进单元，不是顶层产品对象。它负责记录：

- 触发来源
- 本轮状态迁移过程
- 吸收了哪些输入或 `intervention`
- 生成了哪些图变化、`decision` 和 `artifact`

如果未来实现中仍保留 `session`，它也只能是 `workspace` 内部的上下文边界，不应成为对外主契约。

### 3.3 图对象

图中至少应存在以下一等对象：

- `Topic`
- `Question`
- `Tension`
- `Hypothesis`
- `Opportunity`
- `Idea`
- `Evidence`
- `Claim`
- `Decision`
- `Unknown`

这些对象共同表达方向结构、支持材料、系统判断和待解空白。

### 3.4 `Intervention`

`intervention` 不是前端临时输入，而是必须持久化的治理对象。它至少要表达：

- 谁在何时发起
- 目标意图是什么
- 影响范围是整个 `workspace`、某个方向还是某条路径
- 是否已被 runtime 吸收
- 吸收后引发了哪些状态变化

### 3.5 `Projection`

`projection` 是面向产品消费的稳定读取面。典型类型包括：

- `workspace overview projection`
- `direction map projection`
- `path projection`
- `runtime status projection`
- `artifact projection`

### 3.6 `Artifact`

`artifact` 是从图中物化出的结果层对象，只服务于消费和复用，不反向替代图真相层。第一阶段只需要少量、可追溯的物化形态。

## 4. Runtime 模型

### 4.1 选择

v1 采用 `event-driven state machine`，而不是长生命周期黑盒 agent 或固定 DAG。

原因：

- 需要支持长时间运行和中途治理
- 需要把系统行为变成可追溯状态迁移
- 需要把多种输入统一建模为事件
- 需要把恢复、重放和观察建立在同一机制上

### 4.2 主要事件来源

系统至少要吸收以下事件类型：

- `workspace.created`
- `workspace.goal.updated`
- `run.started`
- `context.ingested`
- `research.completed`
- `intervention.received`
- `budget.updated`
- `projection.refresh.requested`
- `run.completed`
- `run.failed`

### 4.3 主要状态

运行态围绕以下阶段组织：

1. `interpret`
2. `explore`
3. `structure`
4. `evaluate`
5. `materialize`
6. `reflect`

这些状态不是一次性 pipeline 的硬编码步骤，而是 runtime 在不同事件推动下反复进入的能力态。

### 4.4 输出

每轮状态迁移的标准输出应包括：

- 图变更
- 新的或被替换的 `decision`
- 新挂接的 `evidence` / `claim` / `unknown`
- `projection` 刷新信号
- 下一步运行策略

系统默认不因为进入关键状态而暂停等待批准。用户可以随时发起 `intervention`，runtime 应在不中断整体运行的前提下吸收并重排后续行为。

## 5. 图更新与一致性模型

产品层希望看到的是 `节点级流式变更`，但内部仍然需要一致性边界。

### 5.1 对外表现

- 地图应像持续生长的结构，而不是每隔一段时间整页刷新
- 用户需要看到方向升温、分叉、合并、降权和停滞
- `intervention` 后的结构变化应尽快反映到 `projection`

### 5.2 内部一致性边界

内部实现可以采用 mutation 批次或等价事务边界来保证：

- 一组相关节点、边和 `decision` 同时提交
- 失败时可回滚或标记失败
- 后续可以重放并重建某一版本的地图

### 5.3 快照与重放

图层至少应支持：

- `snapshot`
- mutation history
- branch or alternative path tracking
- projection rebuild

这样系统才能回答：

- 当前地图相比上一个稳定状态发生了什么
- 某个方向为何被提升或压低
- 某个 `decision` 是何时被替换的
- 某次 `intervention` 具体改变了哪些结构

## 6. Research 与 Context 注入

外部研究与用户上下文不应走多套主流程。统一处理路径应为：

`ContextSource -> normalized fragments -> Evidence / Claim / Unknown -> graph attachment`

这条路径的作用是：

- 让搜索、笔记、网页、文件和临时输入进入同一体系
- 让 runtime 可以统一评估证据密度和空白区
- 让 `decision` 和 `artifact` 可以稳定追溯来源

v1 中，研究层至少要提供三类能力：

- 来源登记与状态管理
- 片段切分与归一化
- 与现有方向结构的语义挂接

## 7. 前后端边界与接口方向

### 7.1 主契约

前端主要消费 `projection API`，而不是原始图表结构。这样做的原因是：

- 前端的主任务是呈现产品语义，而不是复刻图计算逻辑
- 图模型和评分逻辑在 v1 到 v2 之间大概率持续演进
- `projection` 更容易稳定支撑地图、侧栏、运行态和详情面板

### 7.2 API 能力分组

对外接口应按能力组织，而不是按页面按钮组织：

- `workspace` 创建、读取、更新与状态查询
- `run` 启动、状态读取与历史查询
- `projection` 读取与订阅
- `intervention` 提交与处理状态查询
- `context source` 注入与重处理
- `artifact` 物化与追溯读取

### 7.3 原始图读取边界

前端仍可在局部深查场景读取原始图或 mutation 信息，但只用于：

- 调试
- 高级解释
- 运维与分析

它不应成为主视图的默认依赖。

## 8. 存储与异步执行建议

v1 不必过早绑定具体基础设施产品，但需要明确几类存储职责：

- 关系型或等价结构存储：保存 `workspace`、`run`、`intervention`、`artifact` 等元数据
- 图持久化层：保存节点、边、mutation 历史与快照
- 原始内容存储：保存文档、网页抓取结果、资料切片和物化结果
- 事件与异步执行层：承载 runtime 事件、重试、恢复和流式广播
- 缓存或订阅层：服务前端的实时状态同步

关键要求不是某个具体数据库，而是支持：

- 可追溯
- 可重放
- 可恢复
- 可扩展

## 9. v1 验收标准与非目标

### 9.1 验收标准

技术架构至少要支持以下结果：

- 在一个 `workspace` 中持续运行自治探索，而不是只完成一次任务
- 接受中途 `intervention`，并将其转化为后续策略和图变化
- 让 `方向地图` 以流式方式更新
- 让高价值方向稳定追溯到 `evidence`、`decision` 和 `run`
- 让前端只依赖 `projection` 就能完成主控制台体验

### 9.2 非目标

- 不在 v1 做强实时多人协作
- 不在 v1 把前端做成通用图编辑器
- 不在 v1 开放过细的图操作 API 作为主交互
- 不在 v1 建立复杂插件市场或模板系统

## 10. 演进建议

### 阶段一：探索内核定型

优先完成：

- `workspace` 契约
- runtime 事件模型
- 图 mutation / snapshot 机制
- `evidence` 和 `decision` 的持久化路径
- `projection` 基线

### 阶段二：参考工作台验证

优先完成：

- `方向地图` 主投影
- 运行态解释层
- `intervention` 生效路径
- 路径与物化结果下钻

### 阶段三：平台化开放

优先完成：

- API / SDK 稳定化
- strategy 配置能力
- research adapter 扩展
- 更多 `artifact` 形态

## 11. 一句话总结

`Idea Factory` v1 的技术架构应围绕一个中心展开：以 `workspace` 为顶层容器、以 `event-driven state machine` 为自治运行内核、以图为唯一真相层、以 `projection` 为前端主契约，支撑一张持续生长、可治理、可追溯的 `方向地图`。
