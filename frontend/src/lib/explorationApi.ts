import type {
  ApiResponse,
  CreateExplorationRequest,
  ExplorationPayload,
  FeedbackRequest,
} from '../types/api'
import type { ExplorationMutation } from '../types/exploration'
import { buildWorkbenchView, createExplorationSession, expandOpportunity, materializeOpportunity, toggleFavoriteIdea } from './workbench'
import { createMockExplorationRepository } from './mockExplorationRepository'
import { subscribeWorkspace, wsReplayMutations, wsRequest } from './explorationSocket'

const repository = createMockExplorationRepository()
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

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

type WorkspaceListPayload = {
  workspaces: Array<{
    id: string
    topic: string
    output_goal: string
    updated_at: number
  }>
}

async function requestSnapshot(
  path: string,
  init: RequestInit,
): Promise<ApiResponse<ExplorationPayload> | null> {
  try {
    const response = await fetch(`${API_BASE}${path}`, {
      headers: {
        'Content-Type': 'application/json',
      },
      ...init,
    })
    const payload = (await response.json()) as ApiResponse<ExplorationPayload>
    if (payload && typeof payload.code === 'number') {
      return payload
    }
    return null
  } catch {
    return null
  }
}

export async function createExploration(input: CreateExplorationRequest) {
  const wsResp = await wsRequest('create_workspace', undefined, {
    topic: input.topic,
    output_goal: input.outputGoal,
    constraints: input.constraints,
  })
  if (wsResp) return wsResp

  const remote = await requestSnapshot('/exploration/workspaces', {
    method: 'POST',
    body: JSON.stringify({
      topic: input.topic,
      output_goal: input.outputGoal,
      constraints: input.constraints,
    }),
  })
  if (remote) return remote

  const exploration = repository.set(createExplorationSession(input))
  return delay(success({ exploration, presentation: buildWorkbenchView(exploration) }))
}

export async function getExploration(explorationId: string) {
  const wsResp = await wsRequest('get_workspace', explorationId)
  if (wsResp) return wsResp

  const remote = await requestSnapshot(`/exploration/workspaces/${explorationId}`, {
    method: 'GET',
  })
  if (remote) return remote

  const payload = payloadFor(explorationId)
  return delay(payload ? success(payload) : notFound())
}

export async function expandOpportunityRequest(explorationId: string, opportunityId: string) {
  const wsResp = await wsRequest('intervention', explorationId, {
    type: 'expand_opportunity',
    target_id: opportunityId,
  })
  if (wsResp) return wsResp

  const remote = await requestSnapshot(`/exploration/workspaces/${explorationId}/interventions`, {
    method: 'POST',
    body: JSON.stringify({
      type: 'expand_opportunity',
      target_id: opportunityId,
    }),
  })
  if (remote) return remote

  const next = repository.update(explorationId, (session) => expandOpportunity(session, opportunityId))
  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export async function materializeOpportunityRequest(explorationId: string, opportunityId: string) {
  const wsResp = await wsRequest('intervention', explorationId, {
    type: 'expand_opportunity',
    target_id: opportunityId,
  })
  if (wsResp) return wsResp

  const remote = await requestSnapshot(`/exploration/workspaces/${explorationId}/interventions`, {
    method: 'POST',
    body: JSON.stringify({
      type: 'expand_opportunity',
      target_id: opportunityId,
    }),
  })
  if (remote) return remote

  const next = repository.update(explorationId, (session) =>
    materializeOpportunity(session, opportunityId),
  )
  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export async function sendFeedback(explorationId: string, request: FeedbackRequest) {
  const wsResp = await wsRequest('intervention', explorationId, {
    type: request.type === 'toggle_favorite' ? 'toggle_favorite' : request.type,
    target_id: request.nodeId,
  })
  if (wsResp) return wsResp

  const remote = await requestSnapshot(`/exploration/workspaces/${explorationId}/interventions`, {
    method: 'POST',
    body: JSON.stringify({
      type: request.type === 'toggle_favorite' ? 'toggle_favorite' : request.type,
      target_id: request.nodeId,
    }),
  })
  if (remote) return remote

  const next = repository.update(explorationId, (session) => {
    if (request.type === 'toggle_favorite') {
      return toggleFavoriteIdea(session, request.nodeId)
    }
    return session
  })

  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export async function updateExplorationStrategy(
  explorationId: string,
  strategy: {
    interval_ms?: number
    max_runs?: number
    expansion_mode?: 'active' | 'round_robin'
    preferred_branch_id?: string
  },
) {
  const wsResp = await wsRequest('update_strategy', explorationId, strategy)
  if (wsResp) return wsResp

  const remote = await requestSnapshot(`/exploration/workspaces/${explorationId}/strategy`, {
    method: 'PUT',
    body: JSON.stringify(strategy),
  })
  if (remote) return remote

  const next = repository.update(explorationId, (session) => ({
    ...session,
    strategy: {
      interval_ms: strategy.interval_ms ?? session.strategy?.interval_ms ?? 4000,
      max_runs: strategy.max_runs ?? session.strategy?.max_runs ?? 30,
      expansion_mode: strategy.expansion_mode ?? session.strategy?.expansion_mode ?? 'active',
      preferred_branch_id: strategy.preferred_branch_id ?? session.strategy?.preferred_branch_id,
    },
  }))
  return delay(next ? success({ exploration: next, presentation: buildWorkbenchView(next) }) : notFound())
}

export { expandOpportunityRequest as expandOpportunity, materializeOpportunityRequest as materializeOpportunity }

export async function subscribeExploration(
  explorationId: string,
  handlers: {
    onSnapshot?: (payload: ExplorationPayload) => void
    onMutation?: (mutations: ExplorationMutation[]) => void
  },
) {
  return subscribeWorkspace(explorationId, handlers)
}

export async function replayExplorationMutations(
  explorationId: string,
  cursor: string,
  limit?: number,
) {
  const wsResult = await wsReplayMutations(explorationId, cursor, limit)
  if (wsResult) return wsResult

  try {
    const params = new URLSearchParams()
    if (cursor) params.set('cursor', cursor)
    if (limit) params.set('limit', String(limit))
    const response = await fetch(`${API_BASE}/exploration/workspaces/${explorationId}/mutations?${params.toString()}`)
    const payload = (await response.json()) as ApiResponse<{
      mutations: ExplorationMutation[]
      next_cursor?: string
      has_more: boolean
    }>
    if (payload.code === 200) {
      return {
        mutations: payload.data.mutations ?? [],
        nextCursor: payload.data.next_cursor,
        hasMore: payload.data.has_more ?? false,
      }
    }
  } catch {
    // Ignore and fallback to null.
  }

  return null
}

export async function listWorkspaces(limit = 30): Promise<WorkspaceListPayload | null> {
  try {
    const response = await fetch(`${API_BASE}/exploration/workspaces?limit=${limit}`)
    const payload = (await response.json()) as ApiResponse<WorkspaceListPayload>
    if (payload.code === 200) {
      return payload.data
    }
  } catch {
    // Ignore and fallback.
  }
  return null
}

export async function archiveWorkspace(workspaceId: string): Promise<boolean> {
  const wsResp = await wsRequest('archive_workspace', workspaceId)
  if (wsResp && wsResp.code === 200) return true

  try {
    const response = await fetch(`${API_BASE}/exploration/workspaces/${workspaceId}`, {
      method: 'DELETE',
    })
    const payload = (await response.json()) as ApiResponse<{ archived: boolean }>
    return payload.code === 200
  } catch {
    return false
  }
}
