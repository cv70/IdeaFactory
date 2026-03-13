# Idea Factory

> Graph-First, Research-Augmented Exploration OS for growing deep, explainable ideas from ambiguous topics.

一个面向多领域创新任务的探索操作系统。

`Idea Factory` 不是点子列表生成器，不是行业模板库，也不是聊天式研究助手。它的目标是把一个模糊主题持续推进为一张可生长、可追溯、可物化的探索图，并从中产出值得继续推进的 idea 资产。

## Idea Factory 是什么

`Idea Factory` 应被理解为一个 `Graph-First, Research-Augmented Exploration OS`：

- `Graph-First`：所有输入先进入统一探索图，所有输出从图中投影，而不是直接生成答案
- `Research-Augmented`：系统同时支持纯推理、上下文注入和外部研究增强
- `Exploration OS`：它不是单次生成工具，而是一套能持续推进问题空间、机会结构和 idea 资产的通用探索底座

它面向但不限于以下场景：

- 科研方向探索
- 创业机会发现
- 产品方向定义
- 内容选题发散
- 新概念与新机制孵化

这些场景共享同一套底层机制，不按领域拆分核心流程。

## 为什么不是普通 Idea 工具

大多数 idea 工具有四个共性问题：

- 结果是一段一次性文本，无法持续演化
- 不同领域依赖不同模板，缺少统一抽象
- 交互以聊天或表单为主，缺少稳定中间结构
- 外部资料、用户上下文和最终结论之间难以形成长期资产

`Idea Factory` 的回答不是“生成更多内容”，而是建立一个统一的探索系统：

- 用 `Exploration Graph` 承载主题、问题、张力、假设、证据、机会与 idea
- 用高自治 runtime 持续推进探索深度，而不是停留在表层 brainstorm
- 用 research/context 注入提升图的密度和可解释性
- 用 artifact 物化把探索结果转成可消费、可导出、可继续扩展的产物

它的核心价值不是“一次给你一批结果”，而是“持续告诉你下一条最值得推进的路径是什么，以及为什么”。

## 核心系统模型

系统建立在三个统一层之上：

- `Exploration Graph`
- `Exploration Runtime`
- `Research Substrate`

核心关系：

`inputs -> runtime -> graph -> artifacts`

### 1. Exploration Graph

`Exploration Graph` 是系统唯一真相层。所有探索内容都写入同一张图中，而不是散落在 prompt、聊天记录或平铺卡片中。

第一版建议支持以下节点类型：

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

这些节点共同表达：

- 当前主题是什么
- 还有哪些关键问题未解决
- 哪些张力值得继续拆解
- 哪些假设正在形成
- 哪些证据支撑或削弱某条路径
- 哪些方向已经被判定值得推进
- 哪些区域仍然未知或证据不足

不同领域的差异，不体现在主流程分支上，而体现在图中的重心不同：

- 科研更偏 `Question -> Hypothesis -> Evidence -> Idea`
- 创业更偏 `Tension -> Opportunity -> Decision -> Idea`
- 产品更偏 `Question -> Tension -> Opportunity -> Artifact`

但它们始终共用同一张图。

### 2. Exploration Runtime

`Exploration Runtime` 负责驱动图的增长、压缩、评估和物化。

它不应被实现为固定人格的 agent 组合，而应被设计为一个可编排的自治系统，围绕以下阶段能力运行：

1. `Interpret`
2. `Explore`
3. `Structure`
4. `Evaluate`
5. `Materialize`
6. `Reflect`

这里最重要的一点是：

**系统的主输出不是文本，而是 `graph mutation`。**

也就是说，runtime 的核心任务不是“回答问题”，而是：

- 增加新的节点与关系
- 压缩重复结构
- 标记高价值路径
- 暴露未知点和证据缺口
- 从图中物化出可消费 artifact

### 3. Research Substrate

系统需要同时支持三类深度来源：

- 纯推理扩展
- 用户上下文注入
- 外部研究增强

因此，研究与上下文不应只是 prompt 附件，而应被统一吸收到系统中，成为图增长的燃料层。

建议采用统一入口：

`ContextSource -> Evidence / Claim / Unknown -> Graph Attachment`

这样搜索结果、笔记、历史 session、文档片段和第三方数据源都能走同一条路径进入系统。

## 系统运行机制

`Idea Factory` 的默认工作方式不应是“输入主题，然后直接吐答案”，而应是：

`seed -> run -> graph mutation -> evaluation -> artifact -> next action`

更具体地说，一次标准探索会经过四段：

1. `Seed`
   输入主题、目标、约束、上下文源、输出偏好与探索预算
2. `Run`
   系统启动高自治探索，解释主题、扩展问题、识别证据缺口、构建初始结构
3. `Review`
   系统先交付当前探索状态，而不是直接交付答案
4. `Materialize`
   系统从高价值路径中产出 artifact，例如方向地图、问题包、机会说明或 idea 集合

`Reflect` 贯穿整个过程，用于回答：

- 哪些区域探索过浅
- 哪些路径重复或空泛
- 哪些未知点值得补研究
- 下一轮更应该补问题、补证据还是补 idea

## 参考产品形态

`Idea Factory` 不应被限定为一个前端工作台，但需要一个标准 `reference workbench` 作为最佳实践界面。

这个 workbench 的职责不是暴露底层图数据库，而是展示同一张图的不同投影。

建议至少支持四种核心视图：

- `Map View`
  看方向、机会簇、结构空白和未知区
- `Path View`
  看某条从问题到 hypothesis 到 idea 的推演链
- `Run View`
  看某次运行做了什么、为什么这样做
- `Artifact View`
  看已物化产物及其来源追溯

这意味着 reference workbench 是标准消费者，而不是唯一产品形态。未来同一内核可以服务 Web、API、SDK、CLI 或其他行业化前端。

## 技术架构概览

系统建议拆为六层：

- `Interface Layer`
- `Application Layer`
- `Exploration Runtime Layer`
- `Graph Intelligence Layer`
- `Research & Context Layer`
- `Storage Layer`

### Interface Layer

对外提供：

- Web App
- REST API
- SDK
- CLI
- Event/Webhook 接口

### Application Layer

负责：

- workspace 与 session 生命周期
- 任务调度入口
- 配置、权限、配额
- 对外接口聚合

### Exploration Runtime Layer

负责：

- run spec 解释
- phase 编排
- 策略选择
- 图变更生成
- 物化与反思

### Graph Intelligence Layer

负责：

- 节点与边管理
- 去重、聚类、压缩
- 路径评分
- 空白区检测
- 快照、分支与投影

### Research & Context Layer

负责统一接入：

- 用户文档
- 历史 session
- 搜索结果
- 外部知识源
- 第三方研究或数据服务

### Storage Layer

负责持久化：

- workspaces
- sessions
- runs
- nodes
- edges
- snapshots
- mutations
- artifacts
- context sources
- feedback events

## 对外接口方向

API 应围绕“探索系统能力”设计，而不是围绕单一页面动作设计。

建议的核心能力包括：

- session 创建与管理
- run 启动、查看、取消与重跑
- graph 读取与投影读取
- 节点局部干预，例如 expand / prioritize / suppress
- context source 注入与重处理
- artifact 生成与追溯
- feedback 写回与洞察聚合

关键原则：

- `run` 表示一次自治探索
- `node action` 表示一次局部图干预
- `projection payload` 面向前端或 SDK 消费
- `graph snapshot` 面向结构理解与系统计算

## 当前仓库状态

本仓库当前仍处于早期实现阶段。

当前代码主要位于 [frontend](frontend)，现有实现更接近一个前端原型和本地 mock API，用于验证探索工作台的基本交互方向。

本 README 描述的是项目的目标系统定义与长期架构方向，而不是对当前实现状态的逐项映射。当前代码结构不应被视为最终架构边界。

## 本地开发

当前前端工程位于 [frontend/package.json](frontend/package.json)。

安装依赖：

```bash
cd frontend
npm install
```

启动开发环境：

```bash
cd frontend
npm run dev
```

构建与检查：

```bash
cd frontend
npm run build
npm run lint
```

## 路线图

建议分三阶段推进：

### 阶段一：内核成型

优先建立：

- 统一 graph model
- runtime orchestration
- context ingestion
- snapshot / mutation / projection
- artifact model

### 阶段二：Reference Workbench 验证

优先验证：

- map/path/run/artifact 四视图
- 高自治探索是否真的带来深度
- 用户是否愿意沿高价值路径持续推进

### 阶段三：平台化扩展

逐步开放：

- API / SDK
- strategy 配置能力
- research adapter
- artifact 模板化
- 开源内核 + 托管平台双轨交付

## 一句话总结

`Idea Factory` 应被构建为一个以 `Exploration Graph` 为唯一真相层、以高自治 runtime 为增长引擎、以 research/context 为深度燃料、以 artifact 为对外交付形态的通用探索操作系统。
