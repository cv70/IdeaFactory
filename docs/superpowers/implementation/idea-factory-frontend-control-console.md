# Idea Factory 前端控制台实现说明

> 版本：v0.1-implementation-draft
> 日期：2026-04-19
> 状态：从 target specs 到当前前端原型的映射说明

## 1. 文档目的

这份文档回答的问题是：

`新版 specs 已经明确了“方向地图 + 控制台”的产品心智，那么当前 frontend 原型应该怎样逐步演进到这个目标态？`

对应正式目标态来源：

- `IdeaFactory/docs/superpowers/specs/idea-factory-product-design.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-system-architecture.md`
- `IdeaFactory/docs/superpowers/specs/idea-factory-openapi.yaml`

## 2. 当前前端现状摘要

从当前代码结构看，前端已经有一个工作台原型：

- 主入口：`IdeaFactory/frontend/src/App.tsx`
- 工作台布局：`IdeaFactory/frontend/src/components/WorkbenchColumns.tsx`
- 地图视图：`IdeaFactory/frontend/src/components/GraphView.tsx`
- 左侧边栏：`IdeaFactory/frontend/src/components/LeftSidebar.tsx`
- 顶部区：`IdeaFactory/frontend/src/components/WorkspaceHeader.tsx`
- 状态拼装：`IdeaFactory/frontend/src/lib/workbench.ts`
- API 类型：`IdeaFactory/frontend/src/types/api.ts`
- 探索类型：`IdeaFactory/frontend/src/types/exploration.ts`

但当前前端模型仍明显偏旧：

- 中心概念还是 `ExplorationSession + WorkbenchView`
- 还没有正式的 `control_action`、`turn`、`checkpoint`、`memory`、`skill` 视图模型
- 控制面动作更多是原型交互，而不是与新版 OpenAPI 对齐的状态模型

## 3. 新版前端目标：Map-First + Control Console

前端目标态不是“更复杂的看板”，而是一个由两块核心区域组成的工作台：

- `方向地图`：展示 graph truth 的可视投影
- `控制台`：让用户显式驾驭系统当前运行状态和后续行为

这意味着前端需要从“只展示 exploration”升级成“展示 exploration + runtime + control plane + long-term context”。

## 4. 目标态信息架构

建议将页面固定为五块信息层：

### 4.1 地图主画布

当前主要落点：

- `IdeaFactory/frontend/src/components/GraphView.tsx`

目标态职责：

- 展示方向节点和分支关系
- 用 `maturity`、`heat`、`confidence`、`folded` 表达生命周期和热度
- 支持聚焦某个 branch / direction
- 支持从地图直接发起 `redirect`、`review`、`artifact` 等控制动作

### 4.2 运行状态层

当前主要落点：

- `WorkspaceHeader.tsx`
- `LeftSidebar.tsx` 部分摘要

目标态职责：

- 展示当前 `run` 是否活跃
- 展示当前 `turn` 编号与状态
- 展示当前等待原因：`approval / review / external_input / idle`
- 展示最近 checkpoint 和是否可恢复

这是当前前端最缺的一层。

### 4.3 最近变化层

当前主要落点：

- `WorkbenchView.runNotes`
- mutations 相关逻辑

目标态职责：

- 展示最近 graph mutation
- 展示最近 control action 的吸收与反映情况
- 展示最近 run / turn 摘要
- 展示 projection 最近变化原因，而不是只列出新节点

### 4.4 分支详情层

当前主要落点：

- `SidebarPanel.tsx`
- `LeftSidebar.tsx`

目标态职责：

- 展示当前 branch 的 evidence、claim、decision、unknown、artifact
- 展示此 branch 最近几轮 turn 对它做了什么
- 展示与此 branch 绑定的 memory 与 skill
- 展示可执行动作：继续探索、要求 review、转 artifact、折叠、恢复

### 4.5 控制台入口层

当前主要落点：

- `LaunchPanel.tsx`
- `WorkspaceHeader.tsx`

目标态职责：

- 统一发起 control actions
- 管理 `explore / pause / resume / redirect / review / artifact / policy / memory`
- 显示动作状态：`received / absorbed / reflected / rejected`

## 5. 前端状态模型该怎么改

### 5.1 当前类型模型问题

当前 `IdeaFactory/frontend/src/types/exploration.ts` 仍以这些对象为中心：

- `ExplorationSession`
- `WorkbenchView`
- `GenerationRun`
- `ExplorationMutation`

这套模型适合原型阶段，但不足以承载新版 specs。

### 5.2 建议新增的核心类型

建议新增或重构为以下前端类型：

- `Workspace`
- `Run`
- `RunTurn`
- `RunCheckpoint`
- `ControlAction`
- `WorkspaceMemory`
- `SkillBinding`
- `Projection`
- `RuntimeState`

建议新增文件：

- `IdeaFactory/frontend/src/types/runtime.ts`
- `IdeaFactory/frontend/src/types/control.ts`
- `IdeaFactory/frontend/src/types/memory.ts`
- `IdeaFactory/frontend/src/types/skill.ts`

### 5.3 与现有类型的过渡关系

| 旧类型 | 新目标 | 处理方式 |
| --- | --- | --- |
| `ExplorationSession` | `Workspace + Projection` | 逐步拆分 |
| `GenerationRun` | `Run` | 扩展字段并重命名语义 |
| `WorkbenchView` | `Projection + UI derived state` | 下放为视图派生层 |
| `ExplorationMutation` | `TraceEvent / ProjectionChange` | 逐步替换 |

## 6. 组件级改造建议

### 6.1 `GraphView.tsx`

保留，但职责升级为：

- 专注地图投影渲染
- 接受 `Projection.map` 作为主要输入
- 从节点点击事件中发出 `control action intent`

### 6.2 `WorkspaceHeader.tsx`

升级为运行状态条：

- 显示 workspace 标题、run 状态、当前 turn、waiting reason
- 提供 `pause / resume / review / artifact` 快捷动作

### 6.3 `LeftSidebar.tsx`

升级为双态侧栏：

- 未选中节点时：显示 runtime 摘要、recent changes、memory / skill 摘要
- 选中节点时：显示 branch detail、related evidence、recent turn impacts、可用 actions

### 6.4 `LaunchPanel.tsx`

逐步从“启动探索表单”升级为“控制台面板”：

- 新建 workspace 时承担创建入口
- 进入 workspace 后承担 control action 发起入口

### 6.5 `WorkbenchColumns.tsx`

继续作为布局器，但要显式支持三列目标态：

- Left: runtime + control
- Center: map
- Right: branch detail + memory/skills

## 7. API 接入迁移建议

### 7.1 当前主要 API 接入点

- `IdeaFactory/frontend/src/lib/explorationApi.ts`
- `IdeaFactory/frontend/src/lib/explorationSocket.ts`

### 7.2 需要新增的接口消费

按新版 OpenAPI，前端要逐步接入：

- `/workspaces/{workspaceId}/runtime`
- `/workspaces/{workspaceId}/runs/{runId}/turns`
- `/workspaces/{workspaceId}/runs/{runId}/checkpoints`
- `/workspaces/{workspaceId}/control-actions`
- `/workspaces/{workspaceId}/memory`
- `/workspaces/{workspaceId}/skills`
- `/workspaces/{workspaceId}/projection`

### 7.3 建议的前端数据流

```text
load workspace
  -> load projection
  -> load runtime state
  -> load memory + skills
  -> derive workbench view
  -> subscribe trace / mutation stream
  -> optimistic control action submit
  -> reconcile with runtime + projection updates
```

## 8. 交互优先级建议

### Phase 1：先补运行状态可见性

优先做：

- 顶部 run 状态条
- 当前 turn 展示
- waiting reason 展示
- recent checkpoint 展示

### Phase 2：再补控制台动作统一入口

优先做：

- `redirect`
- `review`
- `artifact`
- `resume`
- `pause`

### Phase 3：最后补长期能力可视化

优先做：

- workspace memory 列表
- 当前激活 skill 列表
- branch 相关 memory / skill 映射

## 9. 当前不建议做的事

- 不要先做复杂多窗口工作台
- 不要先把所有 trace 原文暴露给用户
- 不要在没有 runtime 状态条的前提下堆更多按钮
- 不要把 memory / skills 做成独立产品，脱离地图和控制台
- 不要让前端先强依赖 specialist 视图，再去补 run / turn 主视图

## 10. 一句话总结

当前前端最现实的演进路径，不是推翻原型，而是保留 `GraphView + WorkbenchColumns` 这套骨架，逐步补上 `运行状态层 + 控制台动作层 + memory/skills 面板`，把它从 exploration demo 升级成真正的 `workspace control console`。
