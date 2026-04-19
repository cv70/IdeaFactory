# Idea Factory × Claude Code × Codex × Hermes 文档索引

> 版本：v0.2-draft
> 日期：2026-04-19
> 状态：已与 `superpowers/specs` 对齐的参考文档集

这组文档用于回答一个更完整的问题：如果 `Idea Factory` 要继续沿着 `graph-first exploration runtime` 演进，那么 `Claude Code`、`Codex`、`Hermes Agent` 这三个参考系里，分别有哪些模式值得吸收，哪些不该照搬，哪些应该被改写成适合 `workspace -> run -> turn -> graph -> projection` 的产品与技术语义。

## 文档边界

本文档集基于本地仓库中可以直接读取到的材料整理，包括：

- `IdeaFactory/README.md`
- `IdeaFactory/docs/superpowers/specs/*`
- `claudecode/README.md`
- `codex/README.md`
- `codex/AGENTS.md`
- `codex/codex-rs/docs/protocol_v1.md`
- `hermes-agent/README.md`
- `hermes-agent/docs/honcho-integration-spec.md`

说明：

- 这些文档不是对参考项目的逐字转述，而是站在 `Idea Factory` 的目标态上做的架构映射。
- `claudecode` 在本地材料里更像一个“可观察样本”，这里重点参考其公开整理出来的结构与模式，而不是把它当作 `Idea Factory` 的实现模板。
- 这组参考文档现在已经与 `docs/superpowers/specs/` 的新版主线对齐，重点围绕 `控制面 + 执行协议层 + 长期能力层` 展开。
- 真正需要落回产品与工程决策时，应以 `Idea Factory` 自身的 graph-first 语义为最终判断标准。

## 建议阅读顺序

1. 先读 `IdeaFactory/README.md`，理解项目定位与当前阶段。
2. 再读 `docs/superpowers/specs/*`，理解产品、技术、架构与 API 的正式目标态。
3. 最后读本目录，理解这些目标态分别从 `Claude Code`、`Codex`、`Hermes` 吸收了什么。

本目录内建议顺序：

1. [定位与系统分工](./idea-factory-reference-positioning.md)
2. [运行时与长期状态映射](./idea-factory-runtime-and-memory.md)
3. [控制面、命令面与治理机制](./idea-factory-control-surface-and-governance.md)
4. [实施路线图](./idea-factory-implementation-roadmap.md)

## 三个参考系分别提供什么

- `Claude Code`：更像强交互控制面的 agent shell，强调 `command surface`、`mode switch`、`permission checks`、`bridge/plugin/task`。
- `Codex`：更像受控执行协议，强调 `task / turn`、`sandbox / approval`、`tool discipline`、`UI-decoupled runtime`。
- `Hermes Agent`：更像长期运行的 agent operating shell，强调 `session`、`memory`、`skills`、`subagents`、`gateway`。

## 与新版 specs 的对齐点

这组参考文档当前对应到 `Idea Factory` 的正式目标态时，重点落在五个对象上：

- `control_action`
- `run`
- `turn`
- `workspace_memory`
- `skill_binding`

以及三条主线：

- `Claude Code -> control plane`
- `Codex -> run-turn-checkpoint protocol`
- `Hermes -> memory-skill-session continuity`

## 对 Idea Factory 的核心结论

- `Idea Factory` 不应该复制任何一个参考项目的主产品形态。
- `Idea Factory` 最该吸收的是三类能力：
  - 从 `Claude Code` 吸收“控制面”与“用户如何驾驭自治系统”的设计。
  - 从 `Codex` 吸收“受控执行协议”与“高影响动作必须可审计”的设计。
  - 从 `Hermes` 吸收“长期记忆、技能沉淀与多 session 连续性”的设计。
- `Idea Factory` 自己必须保留的核心则只有一个：`graph` 才是真相层，`workspace` 才是长期探索容器，`方向地图` 与 `控制台` 才是产品主界面。

## 推荐影响范围

如果要把这组文档映射到当前仓库，建议优先影响：

- `IdeaFactory/backend/domain/exploration/`
- `IdeaFactory/backend/agents/`
- `IdeaFactory/backend/agentools/`
- `IdeaFactory/docs/superpowers/specs/`
- `IdeaFactory/frontend/src/types/`
- `IdeaFactory/frontend/src/lib/`

一句话说：这组文档服务的是 `Idea Factory` 的 runtime、治理、长期能力和控制面，而不是前端视觉层本身。
