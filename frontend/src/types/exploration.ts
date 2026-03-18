import type { WorkspaceStatus } from './workspace'

export type NodeType =
  | 'topic'
  | 'question'
  | 'tension'
  | 'hypothesis'
  | 'opportunity'
  | 'idea'
  | 'evidence'
  | 'direction'
  | 'claim'
  | 'decision'
  | 'unknown'
  | 'artifact'

export type EdgeType =
  | 'supports'
  | 'refines'
  | 'leads_to'
  | 'expands'
  | 'contradicts'
  | 'justifies'
  | 'branches_from'
  | 'raises'
  | 'resolves'

export type NodeStatus = 'active' | 'draft'

export type Node = {
  id: string
  sessionId: string
  type: NodeType
  title: string
  summary: string
  status: NodeStatus
  score: number
  depth: number
  parentContext?: string
  metadata: {
    branchId?: string
    slot?: string
  }
  evidenceSummary: string
}

export type Edge = {
  id: string
  from: string
  to: string
  type: EdgeType
}

export type GenerationRun = {
  id: string
  round: number
  focus: string
  summary: string
}

export type RuntimeStrategy = {
  interval_ms: number
  max_runs: number
  expansion_mode: 'active' | 'round_robin'
  preferred_branch_id?: string
}

export type ExplorationSession = {
  id: string
  topic: string
  outputGoal: string
  constraints: string
  workspaceStatus?: WorkspaceStatus
  strategy?: RuntimeStrategy
  activeOpportunityId: string
  nodes: Node[]
  edges: Edge[]
  favorites: string[]
  runs: GenerationRun[]
}

export type WorkbenchView = {
  opportunities: Node[]
  activeOpportunity: Node
  questionTrail: Node[]
  hypothesisTrail: Node[]
  ideaCards: Node[]
  savedIdeas: Node[]
  runNotes: GenerationRun[]
}

export type ExplorationMutation = {
  id: string
  workspace_id: string
  kind:
    | 'node_added'
    | 'edge_added'
    | 'run_added'
    | 'favorites_updated'
    | 'active_opportunity_set'
    | 'strategy_updated'
  source: 'runtime' | 'intervention'
  node?: Node
  edge?: Edge
  run?: GenerationRun
  strategy?: RuntimeStrategy
  favorites?: string[]
  active_opportunity_id?: string
  created_at: number
}

export type ExplorationInput = {
  topic: string
  outputGoal: string
  constraints: string
}
