# Idea Factory

> Autonomous exploration control console for turning ambiguous topics into living direction maps.

`Idea Factory` 是一个 `Graph-First, Research-Augmented Exploration OS`。它的 v1 目标不是生成一批一次性的点子，而是在一个长期存在的 `workspace` 中，让系统持续主驾驶探索，把模糊主题推进成一张可追溯、可干预、可继续扩展的 `方向地图`。

## v1 定义

- `workspace` 是产品的一等对象，承载目标、约束、上下文、运行历史和沉淀结果。
- 系统默认主驾驶，持续推进探索；用户主要负责治理、调策略、改方向和随时介入。
- `方向地图` 是主界面和主交付物，`run`、`artifact` 和局部推演链都是它的投影或下钻视图。
- 外部研究、用户资料和临时输入统一进入图，作为 `evidence`、`claim`、`unknown` 等对象参与后续推理。

## 核心工作方式

系统围绕下面这条主循环工作：

`topic + goal + constraints -> autonomous exploration -> direction map -> intervention -> continued exploration -> artifact`

这意味着：

- 主输出不是一次性文本，而是持续生长的结构化图谱。
- 用户不需要手动搭建整张图，但可以在任何时刻插入 `intervention` 改变后续走向。
- 高价值方向需要能追溯到对应的 `run`、`decision`、`evidence` 和路径变化。

## 文档导航

- 产品设计：[docs/superpowers/specs/idea-factory-product-design.md](docs/superpowers/specs/idea-factory-product-design.md)
- 技术设计：[docs/superpowers/specs/idea-factory-technical-design.md](docs/superpowers/specs/idea-factory-technical-design.md)
- 系统架构：[docs/superpowers/specs/idea-factory-system-architecture.md](docs/superpowers/specs/idea-factory-system-architecture.md)

建议阅读顺序：

1. 先读本页，理解项目定位和当前阶段。
2. 再读产品设计，理解用户角色、核心闭环和信息架构。
3. 再读技术设计，理解 `workspace`、runtime、图模型和前后端边界。
4. 最后读系统架构设计，理解系统分层、组件协作、关键时序和数据职责。

## 当前仓库状态

当前仓库仍处于早期阶段，现有代码主要用于验证参考工作台和探索内核的边界。

- `frontend/` 是参考工作台原型，用于验证 `方向地图` 与控制台体验。
- `backend/` 是早期服务与基础设施骨架，尚未完整实现目标 runtime。
- 本仓库中的设计文档描述的是目标系统边界，不应被简单等同为当前实现现状。

## 本地开发

当前最主要的本地入口是前端原型：

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

## 路线图

### 阶段一：探索内核定型

- 统一 `workspace` 与图模型
- 建立自治 runtime 主循环
- 打通 research/context 到图的沉淀路径
- 建立 `projection` 和可追溯的 `decision` / `evidence` 结构

### 阶段二：参考工作台验证

- 以 `方向地图` 为主界面验证产品心智
- 验证系统主驾驶 + 用户治理是否成立
- 验证用户是否愿意沿高价值路径持续推进

### 阶段三：平台化开放

- 开放 API / SDK
- 开放 strategy 和 adapter 扩展能力
- 支持更多研究源、artifact 形态和集成场景

## 一句话总结

`Idea Factory` 要构建的不是另一个聊天式创意工具，而是一套以图为真相层、以自治 runtime 为增长引擎、以 `方向地图` 为主交互面的探索操作系统。
