import type { ApiResponse, ExplorationPayload } from '../types/api'
import type { ExplorationMutation } from '../types/exploration'

type WsEnvelope = {
  type: 'response' | 'snapshot' | 'mutation'
  request_id?: string
  workspace_id?: string
  code?: number
  msg?: string
  data?: ExplorationPayload
  mutations?: ExplorationMutation[]
  next_cursor?: string
  has_more?: boolean
}

type WsRequest = {
  request_id: string
  action: string
  workspace_id?: string
  payload?: unknown
}

type SnapshotHandler = (payload: ExplorationPayload) => void
type MutationHandler = (mutations: ExplorationMutation[]) => void

const pending = new Map<string, (resp: WsEnvelope) => void>()
const snapshotSubscribers = new Map<string, Set<SnapshotHandler>>()
const mutationSubscribers = new Map<string, Set<MutationHandler>>()
let socket: WebSocket | null = null
let connecting: Promise<WebSocket | null> | null = null

function wsEnabled() {
  return import.meta.env.MODE !== 'test'
}

function toWsUrl() {
  const base = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'
  if (base.startsWith('http://') || base.startsWith('https://')) {
    const url = new URL(base)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    url.pathname = `${url.pathname.replace(/\/$/, '')}/exploration/ws`
    return url.toString()
  }
  if (typeof window === 'undefined') return null
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${base}/exploration/ws`
}

function wsEnvelopeToApiResponse(msg: WsEnvelope): ApiResponse<ExplorationPayload> {
  return {
    code: msg.code ?? 500,
    msg: msg.msg,
    data: msg.data as ExplorationPayload,
  }
}

function handleMessage(event: MessageEvent<string>) {
  let msg: WsEnvelope
  try {
    msg = JSON.parse(event.data) as WsEnvelope
  } catch {
    return
  }

  if (msg.type === 'response' && msg.request_id) {
    const resolve = pending.get(msg.request_id)
    if (!resolve) return
    pending.delete(msg.request_id)
    resolve(msg)
    return
  }

  if (msg.type === 'snapshot' && msg.workspace_id && msg.data) {
    const handlers = snapshotSubscribers.get(msg.workspace_id)
    if (!handlers) return
    handlers.forEach((handler) => handler(msg.data!))
    return
  }

  if (msg.type === 'mutation' && msg.workspace_id && msg.mutations) {
    const handlers = mutationSubscribers.get(msg.workspace_id)
    if (!handlers) return
    handlers.forEach((handler) => handler(msg.mutations!))
  }
}

function replaySubscriptions(conn: WebSocket) {
  const workspaceIDs = new Set<string>([
    ...snapshotSubscribers.keys(),
    ...mutationSubscribers.keys(),
  ])
  workspaceIDs.forEach((workspaceID) => {
    const req: WsRequest = {
      request_id: nextRequestID(),
      action: 'subscribe',
      workspace_id: workspaceID,
    }
    try {
      conn.send(JSON.stringify(req))
    } catch {
      // Ignore and let next reconnect retry.
    }
  })
}

function resetSocket() {
  if (socket) {
    socket.onopen = null
    socket.onclose = null
    socket.onmessage = null
    socket.onerror = null
    socket = null
  }
}

async function ensureSocket(): Promise<WebSocket | null> {
  if (!wsEnabled()) return null
  if (socket && socket.readyState === WebSocket.OPEN) return socket
  if (connecting) return connecting

  const wsUrl = toWsUrl()
  if (!wsUrl || typeof WebSocket === 'undefined') return null

  connecting = new Promise<WebSocket | null>((resolve) => {
    const conn = new WebSocket(wsUrl)
    const timeout = setTimeout(() => {
      resetSocket()
      connecting = null
      resolve(null)
    }, 1200)

    conn.onopen = () => {
      clearTimeout(timeout)
      socket = conn
      conn.onmessage = handleMessage
      replaySubscriptions(conn)
      conn.onclose = () => {
        resetSocket()
      }
      conn.onerror = () => {
        resetSocket()
      }
      connecting = null
      resolve(conn)
    }

    conn.onclose = () => {
      clearTimeout(timeout)
      connecting = null
      resolve(null)
    }
  })

  return connecting
}

function nextRequestID() {
  return `req-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

export async function wsRequest(
  action: string,
  workspaceID?: string,
  payload?: unknown,
): Promise<ApiResponse<ExplorationPayload> | null> {
  const conn = await ensureSocket()
  if (!conn) return null

  const requestID = nextRequestID()
  const req: WsRequest = {
    request_id: requestID,
    action,
  }
  if (workspaceID) req.workspace_id = workspaceID
  if (payload !== undefined) req.payload = payload

  return new Promise<ApiResponse<ExplorationPayload> | null>((resolve) => {
    const timeout = setTimeout(() => {
      pending.delete(requestID)
      resolve(null)
    }, 3000)

    pending.set(requestID, (resp) => {
      clearTimeout(timeout)
      resolve(wsEnvelopeToApiResponse(resp))
    })

    try {
      conn.send(JSON.stringify(req))
    } catch {
      clearTimeout(timeout)
      pending.delete(requestID)
      resolve(null)
    }
  })
}

export async function wsReplayMutations(
  workspaceID: string,
  cursor: string,
  limit = 200,
): Promise<{
  mutations: ExplorationMutation[]
  nextCursor?: string
  hasMore: boolean
} | null> {
  const conn = await ensureSocket()
  if (!conn) return null

  const requestID = nextRequestID()
  const req: WsRequest = {
    request_id: requestID,
    action: 'replay_mutations',
    workspace_id: workspaceID,
    payload: {
      cursor,
      limit,
    },
  }

  return new Promise<{
    mutations: ExplorationMutation[]
    nextCursor?: string
    hasMore: boolean
  } | null>((resolve) => {
    const timeout = setTimeout(() => {
      pending.delete(requestID)
      resolve(null)
    }, 3000)

    pending.set(requestID, (resp) => {
      clearTimeout(timeout)
      if ((resp.code ?? 500) !== 200) {
        resolve(null)
        return
      }
      resolve({
        mutations: resp.mutations ?? [],
        nextCursor: resp.next_cursor,
        hasMore: resp.has_more ?? false,
      })
    })

    try {
      conn.send(JSON.stringify(req))
    } catch {
      clearTimeout(timeout)
      pending.delete(requestID)
      resolve(null)
    }
  })
}

export async function subscribeWorkspace(
  workspaceID: string,
  handlers: {
    onSnapshot?: SnapshotHandler
    onMutation?: MutationHandler
  },
): Promise<() => void> {
  if (handlers.onSnapshot) {
    const set = snapshotSubscribers.get(workspaceID) ?? new Set<SnapshotHandler>()
    set.add(handlers.onSnapshot)
    snapshotSubscribers.set(workspaceID, set)
  }
  if (handlers.onMutation) {
    const set = mutationSubscribers.get(workspaceID) ?? new Set<MutationHandler>()
    set.add(handlers.onMutation)
    mutationSubscribers.set(workspaceID, set)
  }

  await wsRequest('subscribe', workspaceID)

  return () => {
    if (handlers.onSnapshot) {
      const set = snapshotSubscribers.get(workspaceID)
      if (set) {
        set.delete(handlers.onSnapshot)
        if (set.size === 0) snapshotSubscribers.delete(workspaceID)
      }
    }

    if (handlers.onMutation) {
      const set = mutationSubscribers.get(workspaceID)
      if (set) {
        set.delete(handlers.onMutation)
        if (set.size === 0) mutationSubscribers.delete(workspaceID)
      }
    }

    if (!snapshotSubscribers.has(workspaceID) && !mutationSubscribers.has(workspaceID)) {
      void wsRequest('unsubscribe', workspaceID)
    }
  }
}
