import type {
  Edge,
  ExplorationInput,
  ExplorationSession,
  GenerationRun,
  Node,
  WorkbenchView,
} from '../types/exploration'

const BRANCH_BLUEPRINTS = [
  {
    slug: 'friction',
    label: 'Learning friction',
    summary: 'Trace the places where the current experience breaks down.',
    question: 'Where does the current journey stall or become too expensive to sustain?',
    hypothesis: 'Reducing setup cost and feedback delay will unlock more participation.',
    ideas: ['Lightweight pilot loop', 'Feedback checkpoint pack'],
  },
  {
    slug: 'measurement',
    label: 'Measurable outcomes',
    summary: 'Frame the problem around signals that can be tested quickly.',
    question: 'Which outcome can be observed in a short cycle without overfitting?',
    hypothesis: 'Smaller, better-instrumented experiments will surface clearer signals.',
    ideas: ['Outcome-first tracker', 'Fast validation protocol'],
  },
  {
    slug: 'adoption',
    label: 'Adoption wedge',
    summary: 'Find an entry point where the topic becomes easy to try and share.',
    question: 'What narrow entry point would make the topic immediately usable?',
    hypothesis: 'A sharper entry wedge creates stronger pull than broader feature coverage.',
    ideas: ['Single-scenario launch kit', 'Peer distribution loop'],
  },
] as const

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

  BRANCH_BLUEPRINTS.forEach((branch, index) => {
    const branchId = `opportunity-${branch.slug}-${index + 1}`
    const opportunity = makeNode(
      sessionId,
      branchId,
      'opportunity',
      `${branch.label} for ${topic}`,
      `${branch.summary} Optimized for ${outputGoal || 'open-ended exploration'}.`,
      1,
    )
    const question = makeNode(
      sessionId,
      branchId,
      'question',
      `${topic}: ${branch.question}`,
      `Question path focused on ${branch.label.toLowerCase()}.`,
      2,
    )
    const hypothesis = makeNode(
      sessionId,
      branchId,
      'hypothesis',
      branch.hypothesis,
      `Hypothesis shaped by ${constraints || 'default exploration constraints'}.`,
      3,
    )

    nodes.push(opportunity, question, hypothesis)
    edges.push(
      makeEdge(topicNode.id, opportunity.id, 'refines'),
      makeEdge(opportunity.id, question.id, 'supports'),
      makeEdge(question.id, hypothesis.id, 'leads_to'),
    )

    branch.ideas.forEach((idea, ideaIndex) => {
      const ideaNode = makeNode(
        sessionId,
        branchId,
        'idea',
        `${idea}: ${topic}`,
        `A concrete idea generated from ${branch.label.toLowerCase()} for ${outputGoal || 'general exploration'}.`,
        4,
        `idea-${ideaIndex + 1}`,
      )
      nodes.push(ideaNode)
      edges.push(makeEdge(hypothesis.id, ideaNode.id, 'leads_to'))
    })
  })

  const opportunities = nodes.filter((node) => node.type === 'opportunity')
  const activeOpportunityId = opportunities[0]?.id ?? topicNode.id

  const initialRun: GenerationRun = {
    id: 'run-1',
    round: 1,
    focus: activeOpportunityId,
    summary: `Mapped ${topic} into ${opportunities.length} initial opportunity branches.`,
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
): WorkbenchView {
  const opportunities = session.nodes.filter((node) => node.type === 'opportunity')
  const activeOpportunity =
    opportunities.find((node) => node.id === activeOpportunityId) ?? opportunities[0]

  if (!activeOpportunity) {
    throw new Error('No active opportunity found for the session.')
  }

  const branchId = activeOpportunity.metadata.branchId ?? activeOpportunity.id
  const ideaCards = byBranch(session, branchId, 'idea')

  return {
    opportunities,
    activeOpportunity,
    questionTrail: byBranch(session, branchId, 'question'),
    hypothesisTrail: byBranch(session, branchId, 'hypothesis'),
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
  const opportunity = session.nodes.find((node) => node.id === opportunityId && node.type === 'opportunity')
  if (!opportunity) {
    return session
  }

  const branchId = opportunity.metadata.branchId ?? opportunity.id
  const branchIdeas = byBranch(session, branchId, 'idea')
  const sourceHypothesis = byBranch(session, branchId, 'hypothesis')[0]
  if (!sourceHypothesis) {
    return session
  }

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
    edges: [...session.edges, makeEdge(sourceHypothesis.id, nextIdea.id, 'expands')],
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
