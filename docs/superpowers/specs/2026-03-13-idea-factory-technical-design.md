# Idea Factory 技术设计文档

> 版本：v1
> 日期：2026-03-13
> 输入依据：`README.md` + 设计评审结论

## 1. 技术目标

第一版技术设计要服务以下目标：

- 建立 `Exploration Graph` 作为唯一真相层
- 支撑高自治、多轮次的 exploration runtime
- 同时支持纯推理、上下文注入和外部研究增强
- 支撑开源内核 + 云端托管双轨交付
- 为 `reference workbench`、API、SDK 和未来扩展保持稳定边界

第一版的工程重点不是界面复杂度，而是图、运行时、研究增强和可追溯性。

## 2. 总体架构

系统建议拆分为六层：

- `Interface Layer`
- `Application Layer`
- `Exploration Runtime Layer`
- `Graph Intelligence Layer`
- `Research & Context Layer`
- `Storage Layer`

### 2.1 Interface Layer

负责暴露：

- Web App
- REST API
- SDK
- CLI
- Webhook 或事件订阅能力

### 2.2 Application Layer

负责：

- workspace/session 生命周期
- 项目与运行配置
- 任务调度入口
- 配额与权限
- 对外接口聚合

### 2.3 Exploration Runtime Layer

系统核心运行层，负责：

- 解释输入目标和约束
- 选择探索策略
- 编排 phase
- 调用研究与上下文能力
- 生成图变更
- 触发评估、物化与反思

### 2.4 Graph Intelligence Layer

负责：

- 图读写
- 去重、聚类、压缩
- 路径排序
- 空白检测
- 冲突识别
- 快照与投影生成

### 2.5 Research & Context Layer

负责统一接入：

- 用户文档
- 历史 session
- 搜索结果
- 外部知识源
- 第三方数据源

### 2.6 Storage Layer

负责持久化结构化对象、原始上下文、快照、事件与缓存。

## 3. 核心数据流

建议主数据流为：

`Seed Input -> Runtime Plan -> Context Gathering -> Phase Execution -> Graph Mutation -> Evaluation -> Artifact Materialization -> Review / Next Action`

这里最重要的工程约束是：

- Runtime 的主输出不是答案文本，而是 `graph mutation`
- 所有输入必须先进入 graph，再由 materializer 产出可消费结果
- 前端不直接操作底层存储对象，而消费稳定的 projection payload

## 4. 核心实体模型

建议第一版至少包含以下一级实体：

- `Workspace`
- `ExplorationSession`
- `Run`
- `GraphNode`
- `GraphEdge`
- `GraphSnapshot`
- `GraphMutation`
- `Artifact`
- `ContextSource`
- `EvidenceFragment`
- `FeedbackEvent`

### 4.1 Workspace

承载：

- 用户或团队范围
- 默认配置
- 可用 context source
- 多个 exploration session

### 4.2 ExplorationSession

承载：

- 主题
- 目标
- 约束
- 运行模式
- 当前状态
- 活跃 branch
- 当前快照指针

### 4.3 Run

承载一次探索运行实例：

- run spec
- 触发方式
- phase 摘要
- 资源消耗
- 终止原因
- 输出统计

### 4.4 GraphNode

README 中已有节点类型：

- `Topic`
- `Question`
- `Tension`
- `Hypothesis`
- `Opportunity`
- `Idea`
- `Evidence`

建议补充三类：

- `Claim`
- `Decision`
- `Unknown`

这样图不只是内容容器，也能记录判断、中间结论与空白区。

建议最少字段：

- `id`
- `workspace_id`
- `session_id`
- `branch_id`
- `node_type`
- `title`
- `summary`
- `body`
- `status`
- `confidence`
- `novelty_score`
- `importance_score`
- `depth_score`
- `source_mix`
- `metadata`
- `created_by_run_id`
- `superseded_by`
- `created_at`
- `updated_at`

### 4.5 GraphEdge

建议最少字段：

- `id`
- `session_id`
- `branch_id`
- `from_node_id`
- `to_node_id`
- `edge_type`
- `weight`
- `confidence`
- `metadata`
- `created_by_run_id`
- `created_at`

建议第一版支持的边类型：

- `questions`
- `explains`
- `contradicts`
- `supports`
- `weakens`
- `refines`
- `derives`
- `leads_to`
- `evidences`
- `clusters_with`
- `selected_for`
- `supersedes`

### 4.6 Artifact

`Artifact` 是从图中物化出的消费层结果，不应只等同于 idea 卡片。

建议支持：

- 机会地图
- 问题包
- 假设树摘要
- idea 集合
- 研究方向包
- 下一步行动建议

### 4.7 ContextSource 与 EvidenceFragment

`ContextSource` 记录来源对象本身，`EvidenceFragment` 记录可引用的内容片段。

建议 `ContextSource` 字段：

- `id`
- `workspace_id`
- `session_id`
- `source_type`
- `uri`
- `title`
- `raw_content_ref`
- `normalized_content`
- `trust_level`
- `freshness`
- `ingestion_status`
- `metadata`

`EvidenceFragment` 负责把大块来源切成可挂接、可追溯、可重用的片段。

## 5. Runtime 设计

Runtime 不建议实现为固定的五个人格化 agent，而建议设计成“阶段能力 + 策略配置 + 运行策略”系统。

### 5.1 建议抽象

- `RunSpec`
- `PhaseExecutor`
- `StrategySelector`
- `GraphMutator`
- `ReflectionEngine`
- `StopPolicy`

### 5.2 RunSpec

定义本次运行的：

- 目标
- 预算
- 输出偏好
- 上下文范围
- 深度要求
- 模式

### 5.3 PhaseExecutor

负责执行阶段能力：

- `interpret`
- `explore`
- `structure`
- `evaluate`
- `materialize`
- `reflect`

### 5.4 StrategySelector

根据 session 目标和当前图状态选择下一步策略，例如：

- 问题扩展优先
- 假设分叉优先
- 证据补强优先
- 机会聚类优先
- idea 物化优先

### 5.5 GraphMutator

负责把 phase 的输出转成结构化的 mutation：

- 新增节点
- 新增边
- 合并重复节点
- 降权低质量路径
- 替换被 supersede 的结构
- 标记未知点和缺口

### 5.6 ReflectionEngine

负责判断：

- 哪些区域探索过浅
- 哪些结构重复
- 哪些路径证据不足
- 哪些未知点值得补研究
- 下一轮优先级如何调整

### 5.7 StopPolicy

负责终止条件：

- 预算耗尽
- 深度达到阈值
- 新增结构收益下降
- 目标 artifact 已可物化
- 等待用户干预

## 6. Graph Intelligence 设计

图层不只是存储层，而是认知计算层。

建议承担以下职责：

- 节点语义去重
- 路径重要度评分
- 机会簇聚类
- 空白区检测
- 证据密度评估
- 冲突路径识别
- 投影生成
- 分支和快照管理

图必须支持：

- `snapshot`
- `mutation log`
- `branch`
- `projection`

这样系统才能回答：

- 某个 idea 是从哪些问题和假设长出来的
- 这一轮运行相比上一轮新增了什么
- 哪些节点被合并、替换或压制
- 当前结构最大的未知点在哪里

## 7. Research & Context 设计

既然第一版要同时支持纯推理、上下文注入和外部研究增强，就不能为不同来源设计不同主流程。

建议统一采用：

`ContextSource -> ExtractedEvidence -> GraphAttachment`

这意味着：

- 任意来源先登记成 `ContextSource`
- 通过抽取与切片生成 `EvidenceFragment`
- 再把片段映射为 `Evidence / Claim / Unknown` 等节点
- 最终通过语义边挂接到现有图上

这样做的好处：

- 搜索、PDF、笔记、历史 session 都能走同一条处理链
- evaluator 可以统一衡量证据密度
- materializer 可以稳定做来源追溯
- 后续扩展外部研究接口不会破坏核心模型

## 8. API 设计

API 应按探索系统能力设计，而不是按页面动作设计。

### 8.1 Session APIs

- `POST /api/v1/workspaces/:workspaceId/sessions`
- `GET /api/v1/sessions/:sessionId`
- `PATCH /api/v1/sessions/:sessionId`
- `POST /api/v1/sessions/:sessionId/branch`

### 8.2 Run APIs

- `POST /api/v1/sessions/:sessionId/runs`
- `GET /api/v1/runs/:runId`
- `POST /api/v1/runs/:runId/cancel`
- `GET /api/v1/sessions/:sessionId/runs`

### 8.3 Graph APIs

- `GET /api/v1/sessions/:sessionId/graph`
- `GET /api/v1/sessions/:sessionId/projections/map`
- `GET /api/v1/sessions/:sessionId/projections/path`
- `GET /api/v1/sessions/:sessionId/projections/run`
- `GET /api/v1/sessions/:sessionId/projections/artifacts`
- `POST /api/v1/sessions/:sessionId/nodes/:nodeId/expand`
- `POST /api/v1/sessions/:sessionId/nodes/:nodeId/prioritize`
- `POST /api/v1/sessions/:sessionId/nodes/:nodeId/suppress`

### 8.4 Context APIs

- `POST /api/v1/sessions/:sessionId/context-sources`
- `GET /api/v1/sessions/:sessionId/context-sources`
- `POST /api/v1/context-sources/:sourceId/reingest`
- `GET /api/v1/context-sources/:sourceId/fragments`

### 8.5 Artifact / Feedback APIs

- `POST /api/v1/sessions/:sessionId/artifacts`
- `GET /api/v1/artifacts/:artifactId`
- `POST /api/v1/sessions/:sessionId/feedback-events`
- `GET /api/v1/sessions/:sessionId/insights`

关键边界：

- `run` 表示一次自治探索
- `node action` 表示对图进行局部干预
- `projection API` 提供稳定视图，而不是暴露原始表结构

## 9. 事件与异步执行

系统天然适合异步执行。建议第一版支持以下事件：

- `run.started`
- `run.phase.completed`
- `graph.updated`
- `artifact.created`
- `context.ingested`
- `feedback.recorded`
- `run.completed`
- `run.failed`

前端同步方式建议：

- 第一版优先 `SSE` 或轮询
- 后续按需要扩展 websocket

理由是第一版重点在于可追踪，而不是强实时。

## 10. 存储与基础设施建议

### 10.1 推荐技术栈

后端建议：

- `Go`
- `Gin`
- `PostgreSQL`
- `Gorm`
- `Redis`
- `OpenTelemetry`

前端建议：

- 继续 `React + TypeScript + Vite`
- 面向 projection 设计 UI，而不是面向原始 graph 表结构设计

### 10.2 为什么不把 agent framework 作为核心

不建议第一版把 LangChain、LlamaIndex 之类框架作为 runtime 核心对象模型。

原因：

- 它们更适合快速拼装 agent 流程，不适合承载长期稳定的 graph-first 内核
- 容易让系统抽象被第三方框架绑架
- 一旦图、artifact、research 模型复杂起来，框架对象会成为限制

更稳妥的方式是：

- 自研轻量 orchestration 层
- 把 LLM provider、research source、ranking logic 都做成 adapter

## 11. 可观测性与非功能要求

第一版必须满足：

- 可追溯：每个 artifact 可追到 path、run、evidence
- 可重放：同一 session 可回看关键 run
- 可扩展：strategy 和 context adapter 可插拔
- 可观测：能量化深度、重复率、证据密度、成本
- 可降级：没有外部研究源时仍能以纯推理模式运行

建议核心观测指标：

- 每轮新增节点数与合并节点数
- 节点平均深度分布
- 证据密度
- 路径评分分布
- artifact 生成成功率
- run 平均耗时与成本

## 12. 演进建议

建议按以下顺序推进：

### 阶段一：内核定型

优先完成：

- graph model
- run model
- mutation/snapshot
- context ingestion
- projection payload

### 阶段二：工作台验证

优先完成：

- 四视图 workbench
- run 可视化
- path 交互
- artifact 追溯

### 阶段三：平台化开放

优先完成：

- SDK
- strategy 配置
- external research adapter
- artifact 模板化
- 托管服务能力

## 13. 一句话总结

第一版技术设计应围绕一个中心展开：把 `Idea Factory` 构造成一个以 `Exploration Graph` 为唯一真相层、以高自治 runtime 为增长引擎、以统一 research substrate 为深度燃料、以 projection 和 artifact 为对外接口的探索操作系统。
