import type {
  Edge,
  ExplorationInput,
  ExplorationSession,
  GenerationRun,
  Node,
  WorkbenchView,
} from '../types/exploration'

function slugify(value: string) {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

function makeNode(
  sessionId: string,
  branchId: string | undefined,
  type: Node['type'],
  title: string,
  summary: string,
  depth: number,
  slot?: string,
): Node {
  return {
    id: `${type}-${slugify(title)}-${branchId ?? 'root'}`,
    sessionId,
    type,
    title,
    summary,
    status: 'active',
    score: Math.max(0.45, 0.92 - depth * 0.1),
    depth,
    parentContext: branchId,
    metadata: {
      branchId,
      slot,
    },
    evidenceSummary: branchId
      ? `Derived from branch ${branchId} during structured exploration.`
      : 'Derived from the user topic and launch intent.',
  }
}

function makeEdge(from: string, to: string, type: Edge['type']): Edge {
  return {
    id: `edge-${from}-${to}-${type}`,
    from,
    to,
    type,
  }
}

export function createExplorationSession(input: ExplorationInput): ExplorationSession {
  const topic = input.topic.trim()
  const outputGoal = input.outputGoal.trim()
  const constraints = input.constraints.trim()
  const sessionId = `session-${slugify(topic)}`
  const topicNode = makeNode(
    sessionId,
    undefined,
    'topic',
    topic,
    `Explore ${topic} through problems, hypotheses, and idea branches.`,
    0,
  )

  const nodes: Node[] = [topicNode]
  const edges: Edge[] = []
  const activeOpportunityId = topicNode.id

  const initialRun: GenerationRun = {
    id: 'run-1',
    round: 1,
    focus: activeOpportunityId,
    summary: 'Initialized workspace with topic anchor; waiting for agent-driven graph growth.',
  }

  return {
    id: sessionId,
    topic,
    outputGoal,
    constraints,
    strategy: {
      interval_ms: 4000,
      max_runs: 30,
      expansion_mode: 'active',
    },
    activeOpportunityId,
    nodes,
    edges,
    favorites: [],
    runs: [initialRun],
  }
}

function byBranch(session: ExplorationSession, branchId: string, type: Node['type']) {
  return session.nodes.filter(
    (node) => node.type === type && node.metadata.branchId === branchId,
  )
}

export function buildWorkbenchView(
  session: ExplorationSession,
  activeOpportunityId = session.activeOpportunityId,
): WorkbenchView | null {
  // Support both old 'opportunity' and new 'direction' node types
  const opportunities = session.nodes.filter(
    (node) => node.type === 'opportunity' || node.type === 'direction',
  )
  if (opportunities.length === 0) return null

  const activeOpportunity =
    opportunities.find((node) => node.id === activeOpportunityId) ?? opportunities[0]

  const branchId = activeOpportunity.metadata.branchId ?? activeOpportunity.id

  // Support both old question/hypothesis and new evidence type
  const questionTrail = session.nodes.filter(
    (node) =>
      (node.type === 'question' || node.type === 'evidence') &&
      node.metadata.branchId === branchId,
  )
  const hypothesisTrail = session.nodes.filter(
    (node) => node.type === 'hypothesis' && node.metadata.branchId === branchId,
  )

  // Support both old idea and new claim/decision/artifact types
  const ideaCards = session.nodes.filter(
    (node) =>
      (node.type === 'idea' ||
        node.type === 'claim' ||
        node.type === 'decision' ||
        node.type === 'artifact') &&
      node.metadata.branchId === branchId,
  )

  return {
    opportunities,
    activeOpportunity,
    questionTrail,
    hypothesisTrail,
    ideaCards,
    savedIdeas: ideaCards.filter((idea) => session.favorites.includes(idea.id)),
    runNotes: session.runs,
  }
}

export function materializeOpportunity(
  session: ExplorationSession,
  opportunityId: string,
): ExplorationSession {
  return expandOpportunity(session, opportunityId)
}

export function expandOpportunity(
  session: ExplorationSession,
  opportunityId: string,
): ExplorationSession {
  const opportunity = session.nodes.find(
    (node) => node.id === opportunityId && (node.type === 'opportunity' || node.type === 'direction'),
  )
  if (!opportunity) {
    return session
  }

  const branchId = opportunity.metadata.branchId ?? opportunity.id
  const branchIdeas = byBranch(session, branchId, 'idea')
  // Support both old hypothesis and new evidence node types as the parent link source.
  const sourceParent =
    byBranch(session, branchId, 'hypothesis')[0] ??
    byBranch(session, branchId, 'evidence')[0] ??
    opportunity

  const nextIdea = makeNode(
    session.id,
    branchId,
    'idea',
    `Expansion ${branchIdeas.length + 1}: ${session.topic}`,
    `A deeper materialization for ${opportunity.title.toLowerCase()}.`,
    4,
    `idea-${branchIdeas.length + 1}`,
  )

  return {
    ...session,
    activeOpportunityId: opportunityId,
    nodes: [...session.nodes, nextIdea],
    edges: [...session.edges, makeEdge(sourceParent.id, nextIdea.id, 'expands')],
    runs: [
      ...session.runs,
      {
        id: `run-${session.runs.length + 1}`,
        round: session.runs.length + 1,
        focus: opportunityId,
        summary: `Expanded ${opportunity.title} with one more materialized idea.`,
      },
    ],
  }
}

export function toggleFavoriteIdea(
  session: ExplorationSession,
  ideaId: string,
): ExplorationSession {
  const exists = session.favorites.includes(ideaId)
  return {
    ...session,
    favorites: exists
      ? session.favorites.filter((favoriteId) => favoriteId !== ideaId)
      : [...session.favorites, ideaId],
  }
}
