# Idea Factory

一个通用的 idea 生产工厂。

它不是按领域切模板的点子生成器，而是一套统一的、可解释的 idea agent 系统：围绕一个主题，先构建问题空间与机会结构，再从中持续长出不同类型的 idea。

当前目标覆盖但不限于：

- 科研 idea
- 创业 idea
- 产品 idea
- 内容 idea

这些场景共享同一套底层机制，不走领域分支。

## 产品定位

Idea Factory 的核心目标不是“直接吐一批结果”，而是帮助用户：

- 先理解一个主题背后的问题空间
- 再识别值得推进的机会路径
- 最后生成可继续扩展的 idea

默认工作方式是：

`主题输入 -> 探索图生成 -> 机会/问题/假设收敛 -> idea 物化 -> 继续展开`

产品强调 4 件事：

- 通用性：同一套系统可覆盖多领域创新场景
- 可解释性：结果能说明自己从哪里长出来
- 可控性：用户可以沿某条路径继续深挖、重做、扩展
- 渐进式探索：先探索，再生成，而不是一次性结束

## 核心设计原则

- 不做科研版、创业版、内容版等领域分支流程
- 不把系统建成纯聊天体验
- 不把输出限制成一组平铺卡片
- 不追求一开始就生成长篇方案

系统必须始终围绕同一个中间模型工作。

## 系统核心模型

系统的底层核心是一个统一的 `Exploration Graph`。

所有领域都在同一张图上表达，节点类型固定为：

- `Topic`
- `Question`
- `Tension`
- `Hypothesis`
- `Opportunity`
- `Idea`
- `Evidence`

含义如下：

- `Topic`：当前探索主题
- `Question`：值得研究、拆解或澄清的问题
- `Tension`：矛盾、缺口、低效、不满足、争议点
- `Hypothesis`：对问题或张力的可探索判断
- `Opportunity`：值得继续展开的方向
- `Idea`：最终可消费的产物
- `Evidence`：节点成立时所依赖的上下文或推理摘要

不同领域的差异，不体现在流程分支上，而体现在探索图中的重心不同。

例如：

- 科研更强调 `Question -> Hypothesis -> Idea`
- 创业更强调 `Tension -> Opportunity -> Idea`

但它们仍然共用同一张图和同一套生成回路。

## Agent 系统

系统建议由统一的 5 个 agent 能力组成：

- `Interpreter`
- `Explorer`
- `Structurer`
- `Evaluator`
- `Materializer`

职责如下：

- `Interpreter`：理解主题、目标、约束和已有上下文
- `Explorer`：扩展问题、张力、假设与机会候选
- `Structurer`：去重、归类、连边、压缩结构
- `Evaluator`：评估哪些路径更值得继续
- `Materializer`：把高价值路径转成用户可读输出

这 5 项是统一职责，不按领域切换。

### 统一生成回路

每次运行都走同一条链路：

1. `Interpret`
2. `Explore`
3. `Structure`
4. `Evaluate`
5. `Materialize`
6. `Reflect`

`Reflect` 用于判断：

- 哪些区域探索过浅
- 哪些节点重复或空泛
- 下一轮该补问题、补假设，还是补 idea

建议采用最多 3 轮的渐进式运行：

- 第 1 轮：快速建图并交付第一版结构
- 第 2 轮：围绕高潜力路径补深度
- 第 3 轮：做收敛、解释增强与空白补齐

## 产品体验

产品形态应是一个“探索驱动的 idea 工作台”，不是简单表单加结果页。

### 默认交互路径

1. 用户输入主题
2. 系统先构建探索摘要
3. 用户看到简化版机会地图
4. 用户沿某条方向继续展开问题和假设
5. 系统基于该路径物化更多 idea

### 输入方式

输入采用渐进式，而不是复杂表单：

- 主题
- 想得到什么类型的 idea
- 可选约束

例如：

- 主题：AI 教育
- 目标：研究方向
- 约束：低成本、可验证

### 页面结构

MVP 建议围绕一个主工作台组织：

- 左侧：方向/机会簇
- 中间：问题链与假设链
- 右侧：idea 卡片

默认不展示完整复杂图，而是展示其简化投影：

- 当前主题
- 3 到 5 个高价值方向
- 每个方向下的关键问题
- 每个方向下的核心假设
- 从这些路径长出的代表性 idea

### 用户动作

所有场景共用同一组动作：

- 展开这个方向
- 基于这条假设生成更多 idea
- 换个角度重做这一支
- 压缩重复内容
- 收藏某条路径或某个 idea
- 标记某条路径值得继续

## 数据模型

工程层也应围绕统一探索图建模，而不是围绕领域对象建模。

MVP 建议至少包含以下对象：

- `ExplorationSession`
- `Node`
- `Edge`
- `Materialization`
- `FeedbackSignal`
- `GenerationRun`

### 对象说明

- `ExplorationSession`：一次完整探索会话
- `Node`：统一节点对象，承载所有类型节点
- `Edge`：节点关系，如 `supports`、`refines`、`leads_to`
- `Materialization`：面向前台的一次结构化输出
- `FeedbackSignal`：收藏、忽略、展开、认为有价值等用户反馈
- `GenerationRun`：某一轮 agent 运行记录与摘要

### `Node` 建议字段

- `id`
- `session_id`
- `type`
- `title`
- `summary`
- `status`
- `score`
- `depth`
- `parent_context`
- `metadata`
- `evidence_summary`

关键约束：

- `Node` 是统一骨架
- 不为不同领域建立不同核心 schema
- 展示差异由节点类型和 `metadata` 决定

## API 方向

API 也应围绕“探索图操作”设计，而不是围绕单一业务场景设计。

建议的 MVP 接口：

- `POST /api/v1/explorations`
- `GET /api/v1/explorations/:id`
- `POST /api/v1/explorations/:id/expand`
- `POST /api/v1/explorations/:id/materialize`
- `POST /api/v1/explorations/:id/feedback`
- `GET /api/v1/explorations/:id/graph`
- `GET /api/v1/explorations/:id/runs`

后端建议同时返回两层数据：

- `graph snapshot`：给结构操作使用
- `presentation payload`：给页面直接渲染

## MVP 范围

### 必做

- 渐进式主题输入
- 探索会话创建
- 简化版机会地图生成
- 问题/假设/机会/idea 的统一节点建模
- 节点展开与继续生成
- idea 收藏与反馈
- 基础去重与结构压缩

### 不做

- 按领域拆分的专用流程
- 长对话式代理产品形态
- 自动联网研究
- 复杂账号系统
- 多人协作
- 长篇 PRD 或商业计划自动生成
- 重型知识图谱可视化编辑器

## 当前仓库状态

当前仓库仍处于前端原型阶段，代码主要位于 [frontend](frontend)。

现状包括：

- 一个 Vite + React + TypeScript 前端工作台
- 探索图驱动的统一领域模型
- 一个异步、本地内存型 mock API，接口形状已经对齐未来 `explorations` 后端

当前前端通过以下边界工作：

- 组件层只依赖 `explorationApi`
- `explorationApi` 返回统一的 `ApiResponse`
- 本地 repository 负责按 exploration id 存取 session
- `workbench` 纯函数负责领域变换和视图 selector

接下来的重构重点应是：

- 把本地 mock API 替换成真实服务端接口
- 保持 `ExplorationSession + Node + Edge` 作为统一领域模型
- 继续扩展 `materialize`、`feedback`、`graph` 等接口行为

## 本地开发

当前前端工程位于 [frontend/package.json](frontend/package.json)。

常用命令：

```bash
cd frontend
npm install
npm run dev
```

构建与检查：

```bash
cd frontend
npm run build
npm run lint
```

## 一句话总结

Idea Factory 是一个统一探索图驱动的通用 idea agent 系统：先发现问题空间与机会结构，再把高价值路径物化为可解释、可继续扩展的 idea。
