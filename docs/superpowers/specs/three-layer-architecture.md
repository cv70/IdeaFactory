# Idea Factory 三层架构图解

## 整体架构

```mermaid
graph TD
    A[用户] --> B[控制面 L1]
    B --> C[执行协议层 L2]
    C --> D[长期能力层 L3]
    D --> E[核心引擎]
    E --> F[数据存储层]
    
    %% 反向影响
    F --> E
    E --> D
    E --> C
    E --> B
    
    %% 层内部组件
    subgraph L1[控制面 L1 - Claude Code 灵感]
        B1[显式治理入口] --> B2[Start/Pause/Resume]
        B1 --> B3[Redirect Focus]
        B1 --> B4[Review/Artifact Requests]
        B1 --> B5[Policy/Memory Controls]
        B6[运行状态视图] --> B7[当前Run/Turn状态]
        B6 --> B8[等待原因与最近动作]
        B6 --> B9[Checkpoint历史]
    end
    
    subgraph L2[执行协议层 L2 - Codex 灵感]
        C1[Run/Turn/Checkpoint协议] --> C2[运行生命周期管理]
        C1 --> C3[检查点与恢复机制]
        C4[工具风险策略] --> C5[权限控制与审批]
        C4 --> C6[风险分级与审批流程]
        C7[事件溯源系统] --> C8[高影响动作记录]
        C7 --> C9[因果链追溯]
        C7 --> C10[审计与调试能力]
    end
    
    subgraph L3[长期能力层 L3 - Hermes Agent 灵感]
        D1[持久记忆系统] --> D2[Workspace Memory - 历史运行结论]
        D1 --> D3[User Preference Memory - 用户偏好学习]
        D4[技能系统] --> D5[技能发现与自动装载]
        D4 --> D6[基于任务模式的技能绑定]
        D4 --> D7[技能版本与使用追溯]
        D8[跨运行上下文] --> D9[异步预取与预热]
        D8 --> D10[Specialist/Subagent协作支持]
        D8 --> D11[上下文压缩与检索]
    end
    
    subgraph Core[核心引擎]
        E1[MainAgent运行时] --> E2[图谱生长决策引擎]
        E1 --> E3[研究/审查/产出决策]
        E1 --> E4[工具编排与执行]
        E5[平衡引擎] --> E6[发散/收敛轴平衡]
        E5 --> E7[研究/产出轴平衡]
        E5 --> E8[激进/稳健轴平衡]
    end
    
    subgraph Storage[数据存储层]
        F1[图谱存储] --> F2[方向/证据/声明/决策节点]
        F3[元数据存储] --> F4[运行摘要与状态信息]
        F5[事件存储] --> F6[工具调用与控制动作事件]
        F7[内容存储] --> F8[原始资料与物化产物]
    end
    
    %% 样式定义
    classDef layer1 fill:#e3f2fd,stroke:#1565c0,stroke-width:2px;
    classDef layer2 fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px;
    classDef layer3 fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px;
    classDef core fill:#fff3e0,stroke:#ef6c00,stroke-width:2px;
    classDef storage fill:#ffebee,stroke:#c62828,stroke-width:2px;
    
    class L1 layer1
    class L2 layer2
    class L3 layer3
    class Core core
    class Storage storage
```

## 数据流示例

```mermaid
sequenceDiagram
    participant User
    participant Control as 控制面 L1
    participant Protocol as 执行协议层 L2
    participant Capability as 长期能力层 L3
    participant Engine as 核心引擎
    participant Storage as 数据存储层
    
    User->>Control: 发起探索控制动作
    Control->>Protocol: 转换为结构化control_action
    Protocol->>Capability: 请求运行上下文
    Capability->>Engine: 提供memory/skills/preferences
    Engine->>Storage: 读取当前图谱状态
    Engine->>Protocol: 请求工具执行许可
    Protocol->>Capability: 检查工具风险策略
    Protocol->>Engine: 授权工具执行
    Engine->>Storage: 写入图谱变更
    Engine->>Protocol: 报告执行结果
    Protocol->>Control: 更新事件溯源
    Control->>User: 返回控制反馈
    Capability->>Capability: 持久化学习结果
```

## 控制动作生命周期

```mermaid
stateDiagram-v2
    [*] --> 用户输入
    用户输入 --> 控制面解析: 结构化动作?
    控制面解析 --> 执行协议层: 有效control_action
    执行协议层 --> 风险评估: 工具/策略检查
    风险评估 --> 批准: 低风险或已授权
    风险评估 --> 拒绝: 高风险未授权
    批准 --> 长期能力层: 注入运行上下文
    长期能力层 --> 核心引擎: 提供memory/skills
    核心引擎 --> 执行决策: 决定下一步行动
    执行决策 --> 工具执行: 通过受控工具面
    工具执行 --> 图谱更新: 通过集成管道
    图谱更新 --> 事件记录: 不可变追踪
    事件记录 --> 长期能力层: 更新学习记录
    长期能力层 --> 控制面: 更新状态视图
    控制面 --> 用户: 反馈控制动作效果
    用户 --> [*]
```

## 三层职责清晰度原则

1. **控制面 (L1)** 只负责：
   - 用户输入的结构化转换
   - 治理动作的分发与可见性
   - 运行状态的统一呈现
   - 不直接修改图谱或执行工具

2. **执行协议层 (L2)** 只负责：
   - 运行生命周期管理 (run/turn/checkpoint)
   - 工具调用的权限控制与审批
   - 所有高影响动作的不可变追踪
   - 不做图谱生长决策或长期记忆管理

3. **长期能力层 (L3)** 只负责：
   - 持久记忆的存储与检索
   - 技能的发现、绑定与应用
   - 用户偏好的学习与应用
   - 跨运行上下文的维护
   - 不直接控制执行或修改图谱

4. **核心引擎** 只负责：
   - 在三层约束下进行图谱生长决策
   - 研究/审查/产出的执行决策
   - 工具的编排与实际调用
   - 平衡引擎的行为轴调节

5. **数据存储层** 只负责：
   - 持久化存储与检索
   - 数据完整性与一致性
   - 不包含任何业务逻辑