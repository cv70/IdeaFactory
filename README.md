# Idea Factory

> Autonomous exploration control console for turning ambiguous topics into living direction maps.

`Idea Factory` 是一个 `Graph-First, Research-Augmented Exploration OS`。它的 v1 目标不是生成一批一次性的点子，而是在一个长期存在的 `workspace` 中，让系统持续主驾驶探索，把模糊主题推进成一张可追溯、可干预、可继续扩展的 `方向地图`。

## v1 定义

- `workspace` 是产品的一等对象，承载目标、约束、上下文、运行历史、长期记忆和技能绑定。
- 系统默认主驾驶，持续推进探索；用户主要通过 `控制台` 做治理、调策略、改方向、暂停、恢复、审查和转产物。
- `方向地图` 是主界面和主交付物，`run`、`turn`、`artifact` 和 trace 都是它的投影或下钻视图。
- 外部研究、用户资料和临时输入统一进入图，作为 `evidence`、`claim`、`unknown` 等对象参与后续推理。

## 核心工作方式

系统围绕下面这条主循环工作：

`topic + goal + constraints -> autonomous exploration -> direction map -> control action -> continued exploration -> artifact`

这意味着：

- 主输出不是一次性文本，而是持续生长的结构化图谱。
- 用户不需要手动搭建整张图，但可以在任何时刻通过 `intervention / review / resume / artifact request / policy adjustment` 改变后续走向。
- 高价值方向需要能追溯到对应的 `run`、`turn`、`decision`、`evidence`、`checkpoint` 和路径变化。

## 文档导航

- 产品设计：[docs/superpowers/specs/idea-factory-product-design.md](docs/superpowers/specs/idea-factory-product-design.md)
- 技术设计：[docs/superpowers/specs/idea-factory-technical-design.md](docs/superpowers/specs/idea-factory-technical-design.md)
- 系统架构：[docs/superpowers/specs/idea-factory-system-architecture.md](docs/superpowers/specs/idea-factory-system-architecture.md)
- 接口契约（OpenAPI）：[docs/superpowers/specs/idea-factory-openapi.yaml](docs/superpowers/specs/idea-factory-openapi.yaml)
- 后端实现映射：[docs/superpowers/implementation/idea-factory-backend-runtime-mapping.md](docs/superpowers/implementation/idea-factory-backend-runtime-mapping.md)
- 前端控制台实现说明：[docs/superpowers/implementation/idea-factory-frontend-control-console.md](docs/superpowers/implementation/idea-factory-frontend-control-console.md)
- 后端开发任务拆解：[docs/superpowers/implementation/idea-factory-backend-development-plan.md](docs/superpowers/implementation/idea-factory-backend-development-plan.md)
- Claude Code / Codex / Hermes 参考文档集：[docs/claudecode-codex-hermes/README.md](docs/claudecode-codex-hermes/README.md)
- Codex / Hermes 旧参考文档集（历史草稿）：[docs/codex-hermes/README.md](docs/codex-hermes/README.md)

建议阅读顺序：

1. 先读本页，理解项目定位和当前阶段。
2. 再读产品设计，理解 `方向地图 + 控制台` 的产品心智与用户治理方式。
3. 再读技术设计，理解 `workspace -> run -> turn -> graph mutation -> projection` 的内核语义和状态流转。
4. 再读系统架构设计，理解 `控制面 + 执行协议层 + 长期能力层` 的职责分工。
5. 最后读 OpenAPI 契约，作为对外接口和状态枚举的唯一字段级来源。
6. 如需理解这些取舍来自哪里，再读 Claude Code / Codex / Hermes 参考文档集。

## 当前仓库状态

当前仓库仍处于早期阶段，现有代码主要用于验证参考工作台和探索内核的边界。

- `frontend/` 是参考工作台原型，用于验证 `方向地图` 与控制台体验。
- `backend/` 其余部分仍然主要是早期服务与基础设施骨架，尚未完整实现目标 exploration runtime。
- 本仓库中的设计文档描述的是目标系统边界，不应被简单等同为当前实现现状。

## 技术实现基线

当前技术实现方向不再把 runtime 只理解成一个抽象状态机，而是把 `Idea Factory` 落成一套 `agent-driven graph runtime`：

- 一个 `MainAgent` 负责解释 `workspace` 目标、读取当前 graph、memory、skills、policy，决定下一步探索、审查、收敛与产出动作，并直接驱动 graph 生长。
- 程序侧保留控制面、调度、最小结构校验、持久化、审批、mutation 广播与恢复，不在后端内核里固化 graph 阶段策略。
- 运行时按 `run -> turn -> checkpoint` 协议治理，所有高影响动作都可追溯、可恢复、可解释。
- 系统内部持续平衡 `发散/收敛`、`研究/产出`、`激进/稳健` 三组行为轴，用户默认不直接调这些轴，但可以通过控制动作改变后续重心。
- 所有执行都通过受控工具面完成，包括 graph 追加、研究、review、artifact、文件访问和外部动作能力。
- `graph`、`projection`、`decision`、`evidence`、`artifact`、`control_action` 和 trace 是 agent 运行后的持久化结果层，而不是模型上下文里的隐式状态。
- 弱方向不会被直接淘汰，而是被降温、折叠、暂停主资源投入，并在新上下文下重新激活。

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

### 阶段一：控制面与执行协议定型

- 统一 `workspace`、`run`、`turn` 与 checkpoint 模型
- 建立 `MainAgent` 直接驱动 graph 生长的自治 runtime 主循环
- 建立 `control_action` 统一治理模型，覆盖 intervention、review、resume、artifact request
- 建立工具风险分级、审批与可追溯事件流
- 建立 `projection` 和可追溯的 `decision / evidence / turn summary` 结构

### 阶段二：长期能力层验证

- 引入 `workspace memory`、`user preference memory`、`skill binding`
- 验证系统是否能跨 run 延续策略和工作上下文
- 验证 `方向地图 + 控制台` 的产品心智是否成立
- 验证用户是否愿意沿高价值路径持续推进

### 阶段三：平台化开放

- 开放 API / SDK
- 开放 skill、policy 和 adapter 扩展能力
- 支持更多研究源、artifact 形态和集成场景
- 在受控边界下支持 specialist、多入口和自动化

## 一句话总结

`Idea Factory` 要构建的不是另一个聊天式创意工具，而是一套以图为真相层、以控制台为治理入口、以 `run-turn` 协议为执行骨架、以长期记忆与技能为进化机制的探索操作系统。
