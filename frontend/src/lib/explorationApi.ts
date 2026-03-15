import type {
  ApiResponse,
  CreateExplorationRequest,
  ExplorationPayload,
  FeedbackRequest,
} from '../types/api'
import type { Edge, ExplorationMutation, ExplorationSession, Node } from '../types/exploration'
import { buildWorkbenchView, createExplorationSession, expandOpportunity, materializeOpportunity, toggleFavoriteIdea } from './workbench'
import { createMockExplorationRepository } from './mockExplorationRepository'
import { subscribeWorkspace, wsReplayMutations, wsRequest } from './explorationSocket'

const repository = createMockExplorationRepository()
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'
const v1Favorites = new Map<string, string[]>()

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
      presentation: null,
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

type V1WorkspaceResponse = {
  workspace: {
    id: string
    topic: string
    goal: string
    constraints?: string[]
  }
}

type V1ProjectionResponse = {
  projection: {
    workspace_id: string
    map: {
      nodes: Array<Record<string, unknown>>
      edges: Array<Record<string, unknown>>
    }
    run_summary?: {
      run_id?: string
      focus?: string
      status?: string
    }
    recent_changes?: Array<{ summary?: string }>
  }
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

async function requestV1<T>(path: string, init?: RequestInit): Promise<{ status: number; data: T } | null> {
  try {
    const response = await fetch(`${API_BASE}${path}`, init)
    const data = (await response.json()) as T
    return { status: response.status, data }
  } catch {
    return null
  }
}

function normalizeNode(raw: Record<string, unknown>, workspaceId: string): Node {
  const metadata = (raw.metadata ?? {}) as { branchId?: string; slot?: string }
  return {
    id: String(raw.id ?? ''),
    sessionId: String(raw.sessionId ?? workspaceId),
    type: String(raw.type ?? 'idea') as Node['type'],
    title: String(raw.title ?? ''),
    summary: String(raw.summary ?? ''),
    status: (String(raw.status ?? 'active') as Node['status']),
    score: Number(raw.score ?? 0),
    depth: Number(raw.depth ?? 0),
    parentContext: raw.parentContext ? String(raw.parentContext) : undefined,
    metadata: {
      branchId: metadata.branchId,
      slot: metadata.slot,
    },
    evidenceSummary: String(raw.evidenceSummary ?? ''),
  }
}

function normalizeEdge(raw: Record<string, unknown>): Edge {
  return {
    id: String(raw.id ?? ''),
    from: String(raw.from ?? ''),
    to: String(raw.to ?? ''),
    type: String(raw.type ?? 'supports') as Edge['type'],
  }
}

function toPayloadFromV1(workspace: V1WorkspaceResponse['workspace'], projection: V1ProjectionResponse['projection']): ExplorationPayload {
  const nodes = (projection.map.nodes ?? []).map((node) => normalizeNode(node, workspace.id))
  const edges = (projection.map.edges ?? []).map(normalizeEdge)
  const opportunities = nodes.filter(
    (node) => node.type === 'opportunity' || node.type === 'direction',
  )
  const activeOpportunityId =
    projection.run_summary?.focus ??
    opportunities[0]?.id ??
    nodes[0]?.id ??
    ''
  const recentSummary = projection.recent_changes?.[projection.recent_changes.length - 1]?.summary
  const session: ExplorationSession = {
    id: workspace.id,
    topic: workspace.topic,
    outputGoal: workspace.goal,
    constraints: (workspace.constraints ?? []).join(', '),
    activeOpportunityId,
    nodes,
    edges,
    favorites: [...(v1Favorites.get(workspace.id) ?? [])],
    runs: projection.run_summary?.run_id
      ? [{
          id: projection.run_summary.run_id,
          round: 1,
          focus: activeOpportunityId,
          summary: recentSummary ?? `Run status: ${projection.run_summary.status ?? 'completed'}`,
        }]
      : [],
  }
  return {
    exploration: session,
    presentation: buildWorkbenchView(session),
  }
}

function toggleFavorite(ids: string[], nodeId: string) {
  return ids.includes(nodeId)
    ? ids.filter((id) => id !== nodeId)
    : [...ids, nodeId]
}

async function loadV1Payload(workspaceId: string): Promise<ApiResponse<ExplorationPayload> | null> {
  const workspaceResp = await requestV1<V1WorkspaceResponse>(`/workspaces/${workspaceId}`, { method: 'GET' })
  if (!workspaceResp || workspaceResp.status !== 200) return null
  const projectionResp = await requestV1<V1ProjectionResponse>(`/workspaces/${workspaceId}/projection`, { method: 'GET' })
  if (!projectionResp || projectionResp.status !== 200) return null
  return success(toPayloadFromV1(workspaceResp.data.workspace, projectionResp.data.projection))
}

export async function createExploration(input: CreateExplorationRequest) {
  const v1Created = await requestV1<V1WorkspaceResponse>('/workspaces', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      topic: input.topic,
      goal: input.outputGoal,
      output_goal: input.outputGoal,
      constraints: input.constraints ? [input.constraints] : [],
    }),
  })
  if (v1Created && v1Created.status === 201) {
    const payload = await loadV1Payload(v1Created.data.workspace.id)
    if (payload) return payload
  }

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
  const v1Payload = await loadV1Payload(explorationId)
  if (v1Payload) return v1Payload

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
  const v1Intervention = await requestV1<{ intervention: { id: string } }>(
    `/workspaces/${explorationId}/interventions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        intent: 'expand this opportunity',
        target_branch_id: opportunityId,
      }),
    },
  )
  if (v1Intervention && v1Intervention.status === 202) {
    const payload = await loadV1Payload(explorationId)
    if (payload) return payload
  }

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
  const v1Intervention = await requestV1<{ intervention: { id: string } }>(
    `/workspaces/${explorationId}/interventions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        intent: 'materialize this opportunity into artifacts',
        target_branch_id: opportunityId,
      }),
    },
  )
  if (v1Intervention && v1Intervention.status === 202) {
    const payload = await loadV1Payload(explorationId)
    if (payload) return payload
  }

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
  if (request.type === 'toggle_favorite') {
    const v1Intervention = await requestV1<{ intervention: { id: string } }>(
      `/workspaces/${explorationId}/interventions`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          intent: 'toggle favorite',
          target_branch_id: request.nodeId,
        }),
      },
    )
    if (v1Intervention && v1Intervention.status === 202) {
      const payload = await loadV1Payload(explorationId)
      if (payload) {
        const nextFavorites = toggleFavorite(payload.data.exploration.favorites, request.nodeId)
        v1Favorites.set(explorationId, nextFavorites)
        const patched: ExplorationPayload = {
          exploration: {
            ...payload.data.exploration,
            favorites: nextFavorites,
          },
          presentation: buildWorkbenchView({
            ...payload.data.exploration,
            favorites: nextFavorites,
          }),
        }
        return success(patched)
      }
    }
  }

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
