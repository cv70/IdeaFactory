import { createContext, useContext } from 'react'

export type Lang = 'en' | 'zh'

const translations = {
  en: {
    // Header
    'header.title': 'Idea Factory',
    'header.langSwitch': '中文',

    // Nav sidebar
    'nav.newExploration': 'New Exploration',
    'nav.recent': 'Recent',

    // Launch panel
    'launch.label': 'Launch',
    'launch.title': 'Start from a theme, not a template',
    'launch.description':
      'Keep the input lightweight. The system will turn it into opportunities, questions, hypotheses, and materialized ideas.',
    'launch.topic': 'Topic',
    'launch.topicPlaceholder': 'AI education, oncology screening, creator infrastructure',
    'launch.outputGoal': 'Output goal',
    'launch.outputGoalPlaceholder': 'Research directions, venture opportunities, product bets',
    'launch.constraints': 'Constraints',
    'launch.constraintsPlaceholder': 'Low-cost, explainable, easy to validate',
    'launch.startButton': 'Start exploration',
    'launch.startingButton': 'Starting...',

    // Workspaces
    'workspaces.empty': 'No historical workspaces yet.',
    'workspaces.open': 'Open',
    'workspaces.pause': 'Pause',
    'workspaces.resume': 'Resume',
    'workspaces.archive': 'Archive',

    // Workbench map column
    'map.label': 'Map',
    'map.title': 'Direction map',
    'map.question': 'Question',
    'map.hypothesis': 'Hypothesis',
    'map.expandButton': 'Expand this branch',

    // Workbench reasoning column
    'reasoning.label': 'Reasoning',
    'reasoning.title': 'Question trail',

    // Workbench materialization column
    'materialization.label': 'Materialization',
    'materialization.title': 'Materialized ideas',
    'materialization.description': 'Concrete outputs growing from the currently selected path.',

    // Idea actions
    'idea.save': 'Save idea',
    'idea.unsave': 'Unsave idea',

    // Sidebar – intervention
    'sidebar.intervention.label': 'Intervention',
    'sidebar.intervention.title': 'Submit intervention',
    'sidebar.intervention.placeholder': 'Steer the exploration in a new direction...',
    'sidebar.intervention.button': 'Submit',
    'sidebar.intervention.statusLabel': 'Status:',

    // Sidebar – strategy
    'sidebar.strategy.label': 'Governance',
    'sidebar.strategy.title': 'Runtime strategy',
    'sidebar.strategy.livePrefix': 'Live strategy:',
    'sidebar.strategy.balanced': 'Balanced',
    'sidebar.strategy.rapid': 'Rapid Scan',
    'sidebar.strategy.focused': 'Focus Active',
    'sidebar.strategy.intervalMs': 'Interval (ms)',
    'sidebar.strategy.maxRuns': 'Max runs',
    'sidebar.strategy.expansionMode': 'Expansion mode',
    'sidebar.strategy.preferredBranch': 'Preferred branch',
    'sidebar.strategy.apply': 'Apply strategy',
    'sidebar.strategy.applying': 'Updating...',
    'sidebar.strategy.activeBranch': 'Active branch',
    'sidebar.strategy.roundRobin': 'Round robin',
    'sidebar.strategy.none': 'None',

    // Sidebar – history
    'sidebar.history.label': 'History',
    'sidebar.history.title': 'Strategy history',
    'sidebar.history.empty': 'No strategy updates yet.',
    'sidebar.history.rollback': 'Rollback',

    // Sidebar – saved
    'sidebar.saved.label': 'Saved',
    'sidebar.saved.titlePrefix': 'Saved ideas',
    'sidebar.saved.empty': 'No saved ideas yet.',

    // Sidebar – runs
    'sidebar.runs.label': 'Runs',
    'sidebar.runs.title': 'Recent run notes',
    'sidebar.runs.roundPrefix': 'Round',
    'sidebar.agentLog.label': 'Agent log',
    'sidebar.agentLog.title': 'Recent agent events',
    'sidebar.agentLog.empty': 'No agent events yet.',
    'sidebar.agentLog.runPrefix': 'Run',
    'sidebar.agentLog.eventTypeStart': 'Start',
    'sidebar.agentLog.eventTypeDelegate': 'Delegate',
    'sidebar.agentLog.eventTypeTool': 'Tool',
    'sidebar.agentLog.eventTypeSummary': 'Summary',
    'sidebar.agentLog.eventTypeError': 'Error',

    // Graph view
    'graph.expandButton': 'Expand branch',
    'graph.dismiss': 'Dismiss',
    'graph.emptyHint': 'Starting exploration…',
  },

  zh: {
    // Header
    'header.title': '创意工厂',
    'header.langSwitch': 'English',

    // Nav sidebar
    'nav.newExploration': '新建探索',
    'nav.recent': '最近',

    // Launch panel
    'launch.label': '启动',
    'launch.title': '从主题出发，而非模板',
    'launch.description':
      '保持输入简洁。系统将自动演化出机会、问题、假设和具象创意。',
    'launch.topic': '主题',
    'launch.topicPlaceholder': 'AI教育、肿瘤筛查、创作者基础设施',
    'launch.outputGoal': '输出目标',
    'launch.outputGoalPlaceholder': '研究方向、创业机会、产品赌注',
    'launch.constraints': '约束条件',
    'launch.constraintsPlaceholder': '低成本、可解释、易于验证',
    'launch.startButton': '开始探索',
    'launch.startingButton': '启动中...',

    // Workspaces
    'workspaces.empty': '暂无历史工作空间。',
    'workspaces.open': '打开',
    'workspaces.pause': '暂停',
    'workspaces.resume': '继续',
    'workspaces.archive': '归档',

    // Workbench map column
    'map.label': '地图',
    'map.title': '方向图谱',
    'map.question': '问题',
    'map.hypothesis': '假设',
    'map.expandButton': '展开此分支',

    // Workbench reasoning column
    'reasoning.label': '推理',
    'reasoning.title': '问题链',

    // Workbench materialization column
    'materialization.label': '物化',
    'materialization.title': '具象创意',
    'materialization.description': '从当前选定路径生长出的具体成果。',

    // Idea actions
    'idea.save': '收藏',
    'idea.unsave': '取消收藏',

    // Sidebar – intervention
    'sidebar.intervention.label': '干预',
    'sidebar.intervention.title': '提交干预',
    'sidebar.intervention.placeholder': '引导探索向新方向发展...',
    'sidebar.intervention.button': '提交',
    'sidebar.intervention.statusLabel': '状态：',

    // Sidebar – strategy
    'sidebar.strategy.label': '策略',
    'sidebar.strategy.title': '运行策略',
    'sidebar.strategy.livePrefix': '当前策略：',
    'sidebar.strategy.balanced': '均衡',
    'sidebar.strategy.rapid': '快速扫描',
    'sidebar.strategy.focused': '专注当前',
    'sidebar.strategy.intervalMs': '间隔 (ms)',
    'sidebar.strategy.maxRuns': '最大轮次',
    'sidebar.strategy.expansionMode': '扩展模式',
    'sidebar.strategy.preferredBranch': '首选分支',
    'sidebar.strategy.apply': '应用策略',
    'sidebar.strategy.applying': '更新中...',
    'sidebar.strategy.activeBranch': '活跃分支',
    'sidebar.strategy.roundRobin': '轮询',
    'sidebar.strategy.none': '无',

    // Sidebar – history
    'sidebar.history.label': '历史',
    'sidebar.history.title': '策略历史',
    'sidebar.history.empty': '暂无策略更新。',
    'sidebar.history.rollback': '回滚',

    // Sidebar – saved
    'sidebar.saved.label': '收藏',
    'sidebar.saved.titlePrefix': '已收藏创意',
    'sidebar.saved.empty': '暂无已收藏创意。',

    // Sidebar – runs
    'sidebar.runs.label': '运行',
    'sidebar.runs.title': '最近运行记录',
    'sidebar.runs.roundPrefix': '第',
    'sidebar.agentLog.label': 'Agent 日志',
    'sidebar.agentLog.title': '最近 Agent 事件',
    'sidebar.agentLog.empty': '暂无 Agent 事件。',
    'sidebar.agentLog.runPrefix': '运行',
    'sidebar.agentLog.eventTypeStart': '启动',
    'sidebar.agentLog.eventTypeDelegate': '委派',
    'sidebar.agentLog.eventTypeTool': '工具',
    'sidebar.agentLog.eventTypeSummary': '总结',
    'sidebar.agentLog.eventTypeError': '错误',

    // Graph view
    'graph.expandButton': '展开此分支',
    'graph.dismiss': '关闭',
    'graph.emptyHint': '探索启动中…',
  },
} as const

type TranslationKey = keyof typeof translations.en

export type I18nContext = {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: TranslationKey) => string
}

export const LangContext = createContext<I18nContext>({
  lang: 'en',
  setLang: () => {},
  t: (key) => key,
})

export function useTranslation() {
  return useContext(LangContext)
}

export function makeT(lang: Lang): (key: TranslationKey) => string {
  return (key) => (translations[lang] as Record<string, string>)[key] ?? key
}
