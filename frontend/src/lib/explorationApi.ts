import type {
  ApiResponse,
  CreateExplorationRequest,
  ExplorationPayload,
  FeedbackRequest,
} from '../types/api'
import { buildWorkbenchView, createExplorationSession, expandOpportunity, materializeOpportunity, toggleFavoriteIdea } from './workbench'
import { createMockExplorationRepository } from './mockExplorationRepository'

const repository = createMockExplorationRepository()

function success(data: ExplorationPayload): ApiResponse<ExplorationPayload> {
  return {
    code: 200,
    data,
  }
}

function notFound(): ApiResponse<ExplorationPayload> {
  return {
    code: 404,
    msg: 'Exploration not found',
    data: {
      exploration: {
        id: '',
        topic: '',
        outputGoal: '',
        constraints: '',
        activeOpportunityId: '',
        nodes: [],
        edges: [],
        favorites: [],
        runs: [],
      },
      presentation: {
        opportunities: [],
        activeOpportunity: undefined as never,
        questionTrail: [],
        hypothesisTrail: [],
        ideaCards: [],
        savedIdeas: [],
        runNotes: [],
      },
    },
  }
}

function payloadFor(explorationId: string) {
  const exploration = repository.get(explorationId)
  if (!exploration) return null
  return {
    exploration,
    presentation: buildWorkbenchView(exploration),
  }
}

function delay<T>(value: T) {
  return new Promise<T>((resolve) => {
    setTimeout(() => resolve(value), 10)
  })
}

export async function createExploration(input: CreateExplorationRequest) {
  const exploration = repository.set(createExplorationSession(input))
  return delay(success({ exploration, presentation: buildWorkbenchView(exploration) }))
}

export async function getExploration(explorationId: string) {
  const payload = payloadFor(explorationId)
  return delay(payload ? success(payload) : notFound())
}

export async function expandOpportunityRequest(explorationId: string, opportunityId: string) {
  const next = repository.update(explorationId, (session) => expandOpportunity(session, opportunityId))
  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export async function materializeOpportunityRequest(explorationId: string, opportunityId: string) {
  const next = repository.update(explorationId, (session) =>
    materializeOpportunity(session, opportunityId),
  )
  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export async function sendFeedback(explorationId: string, request: FeedbackRequest) {
  const next = repository.update(explorationId, (session) => {
    if (request.type === 'toggle_favorite') {
      return toggleFavoriteIdea(session, request.nodeId)
    }
    return session
  })

  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export { expandOpportunityRequest as expandOpportunity, materializeOpportunityRequest as materializeOpportunity }
