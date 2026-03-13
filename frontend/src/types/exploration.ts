export type NodeType =
  | 'topic'
  | 'question'
  | 'tension'
  | 'hypothesis'
  | 'opportunity'
  | 'idea'
  | 'evidence'

export type EdgeType = 'supports' | 'refines' | 'leads_to' | 'expands'

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

export type ExplorationSession = {
  id: string
  topic: string
  outputGoal: string
  constraints: string
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

export type ExplorationInput = {
  topic: string
  outputGoal: string
  constraints: string
}
