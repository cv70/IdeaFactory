import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'
import { AppHeader } from './components/AppHeader'
import { LaunchPanel } from './components/LaunchPanel'
import { SidebarPanel } from './components/SidebarPanel'
import { WorkspaceManager } from './components/WorkspaceManager'
import { WorkbenchColumns } from './components/WorkbenchColumns'
import { DEFAULT_CONSTRAINTS, EXAMPLE_TOPICS } from './data/mockExploration'
import {
  archiveWorkspace,
  createExploration,
  expandOpportunity as expandOpportunityRequest,
  getExploration,
  listWorkspaces,
  replayExplorationMutations,
  sendFeedback,
  subscribeExploration,
  updateExplorationStrategy,
} from './lib/explorationApi'
import { applyExplorationMutations } from './lib/mutations'
import { buildWorkbenchView } from './lib/workbench'
import type { ExplorationMutation, ExplorationSession, Node, RuntimeStrategy } from './types/exploration'

type StrategyHistoryEntry = {
  id: string
  createdAt: number
  strategy: RuntimeStrategy
}

type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}

const WORKSPACE_HISTORY_KEY = 'idea-factory.workspace-history.v1'

function App() {
  const [topic, setTopic] = useState('')
  const [outputGoal, setOutputGoal] = useState('')
  const [constraints, setConstraints] = useState(DEFAULT_CONSTRAINTS)
  const [exploration, setExploration] = useState<ExplorationSession | null>(null)
  const [selectedOpportunityId, setSelectedOpportunityId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [strategyUpdating, setStrategyUpdating] = useState(false)
  const [error, setError] = useState('')
  const mutationCursorRef = useRef('')
  const [strategyHistory, setStrategyHistory] = useState<StrategyHistoryEntry[]>([])
  const [workspaceHistory, setWorkspaceHistory] = useState<WorkspaceRecord[]>([])

  function toCursor(createdAt: number, id: string) {
    return `${createdAt}|${id}`
  }

  function mergeStrategyHistory(mutations: ExplorationMutation[]) {
    const entries = mutations
      .filter((mutation) => mutation.kind === 'strategy_updated' && mutation.strategy)
      .map((mutation) => ({
        id: mutation.id,
        createdAt: mutation.created_at,
        strategy: mutation.strategy as RuntimeStrategy,
      }))

    if (entries.length === 0) return

    setStrategyHistory((current) => {
      const seen = new Set(current.map((entry) => entry.id))
      const merged = [...current]
      for (const entry of entries) {
        if (!seen.has(entry.id)) {
          merged.unshift(entry)
          seen.add(entry.id)
        }
      }
      return merged.slice(0, 20)
    })
  }

  const view = useMemo(() => {
    if (!exploration) return null
    return buildWorkbenchView(exploration, selectedOpportunityId ?? undefined)
  }, [exploration, selectedOpportunityId])

  const savedIdeas = useMemo(() => {
    if (!exploration) return []
    return exploration.nodes.filter(
      (node) => node.type === 'idea' && exploration.favorites.includes(node.id),
    )
  }, [exploration])

  useEffect(() => {
    void (async () => {
      const remote = await listWorkspaces(30)
      if (remote && remote.workspaces.length > 0) {
        const parsed = remote.workspaces.map((workspace) => ({
          id: workspace.id,
          topic: workspace.topic,
          updatedAt: workspace.updated_at,
        }))
        setWorkspaceHistory(parsed)
        try {
          localStorage.setItem(WORKSPACE_HISTORY_KEY, JSON.stringify(parsed))
        } catch {
          // Ignore persistence errors.
        }
        return
      }

      try {
        const raw = localStorage.getItem(WORKSPACE_HISTORY_KEY)
        if (!raw) return
        const parsed = JSON.parse(raw) as WorkspaceRecord[]
        if (!Array.isArray(parsed)) return
        setWorkspaceHistory(parsed)
      } catch {
        // Ignore invalid history payload.
      }
    })()
  }, [])

  function upsertWorkspaceHistory(record: WorkspaceRecord) {
    setWorkspaceHistory((current) => {
      const next = [
        record,
        ...current.filter((workspace) => workspace.id !== record.id),
      ].slice(0, 30)
      try {
        localStorage.setItem(WORKSPACE_HISTORY_KEY, JSON.stringify(next))
      } catch {
        // Ignore persistence errors.
      }
      return next
    })
  }

  useEffect(() => {
    if (!exploration?.id) return
    let active = true
    let cleanup: (() => void) | undefined

    subscribeExploration(exploration.id, {
      onSnapshot: (payload) => {
        if (!active) return
        setExploration(payload.exploration)
        setSelectedOpportunityId((current) => {
          if (!current) return payload.exploration.activeOpportunityId
          const exists = payload.exploration.nodes.some((node) => node.id === current)
          return exists ? current : payload.exploration.activeOpportunityId
        })
      },
      onMutation: (mutations) => {
        if (!active) return
        mergeStrategyHistory(mutations)
        for (const mutation of mutations) {
          mutationCursorRef.current = toCursor(mutation.created_at, mutation.id)
        }
        setExploration((current) => {
          if (!current) return current
          return applyExplorationMutations(current, mutations)
        })
      },
    }).then((dispose) => {
      cleanup = dispose
      void (async () => {
        let cursor = mutationCursorRef.current
        for (let page = 0; page < 5; page += 1) {
          const replay = await replayExplorationMutations(exploration.id, cursor, 200)
          if (!active || !replay || replay.mutations.length === 0) return
          mergeStrategyHistory(replay.mutations)
          for (const mutation of replay.mutations) {
            mutationCursorRef.current = toCursor(mutation.created_at, mutation.id)
          }
          setExploration((current) => {
            if (!current) return current
            return applyExplorationMutations(current, replay.mutations)
          })
          if (!replay.hasMore || !replay.nextCursor) return
          cursor = replay.nextCursor
        }
      })()
    })

    return () => {
      active = false
      if (cleanup) cleanup()
    }
  }, [exploration?.id])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!topic.trim() || !outputGoal.trim()) {
      return
    }

    setLoading(true)
    setError('')

    const response = await createExploration({
      topic,
      outputGoal,
      constraints,
    })

    if (response.code !== 200) {
      setLoading(false)
      setError(response.msg ?? 'Failed to start exploration')
      setExploration(null)
      return
    }

    setExploration(response.data.exploration)
    upsertWorkspaceHistory({
      id: response.data.exploration.id,
      topic: response.data.exploration.topic,
      updatedAt: Date.now(),
    })
    setStrategyHistory([])
    mutationCursorRef.current = ''
    setSelectedOpportunityId(response.data.exploration.activeOpportunityId)
    setLoading(false)
  }

  function handleSelectOpportunity(opportunity: Node) {
    setSelectedOpportunityId(opportunity.id)
  }

  async function handleExpandOpportunity(opportunity: Node) {
    if (!exploration) return

    setLoading(true)
    setError('')

    const response = await expandOpportunityRequest(exploration.id, opportunity.id)

    if (response.code !== 200) {
      setLoading(false)
      setError(response.msg ?? 'Failed to expand branch')
      return
    }

    setExploration(response.data.exploration)
    setSelectedOpportunityId(opportunity.id)
    setLoading(false)
  }

  async function handleToggleFavorite(idea: Node) {
    if (!exploration) return

    setError('')

    const response = await sendFeedback(exploration.id, {
      type: 'toggle_favorite',
      nodeId: idea.id,
    })

    if (response.code !== 200) {
      setError(response.msg ?? 'Failed to update favorite')
      return
    }

    setExploration(response.data.exploration)
  }

  async function handleUpdateStrategy(strategy: {
    interval_ms?: number
    max_runs?: number
    expansion_mode?: 'active' | 'round_robin'
    preferred_branch_id?: string
  }) {
    if (!exploration) return

    setStrategyUpdating(true)
    setError('')

    const response = await updateExplorationStrategy(exploration.id, strategy)
    if (response.code !== 200) {
      setError(response.msg ?? 'Failed to update strategy')
      setStrategyUpdating(false)
      return
    }

    setExploration(response.data.exploration)
    if (response.data.exploration.strategy) {
      setStrategyHistory((current) => [
        {
          id: `local-${Date.now()}`,
          createdAt: Date.now(),
          strategy: response.data.exploration.strategy,
        },
        ...current,
      ].slice(0, 20))
    }
    setStrategyUpdating(false)
  }

  async function handleRollbackStrategy(strategy: RuntimeStrategy) {
    await handleUpdateStrategy({
      interval_ms: strategy.interval_ms,
      max_runs: strategy.max_runs,
      expansion_mode: strategy.expansion_mode,
      preferred_branch_id: strategy.preferred_branch_id,
    })
  }

  async function handleSelectWorkspace(workspaceId: string) {
    setLoading(true)
    setError('')

    const response = await getExploration(workspaceId)
    if (response.code !== 200) {
      setError(response.msg ?? 'Failed to load workspace')
      setLoading(false)
      return
    }

    setExploration(response.data.exploration)
    setSelectedOpportunityId(response.data.exploration.activeOpportunityId)
    upsertWorkspaceHistory({
      id: response.data.exploration.id,
      topic: response.data.exploration.topic,
      updatedAt: Date.now(),
    })
    setLoading(false)
  }

  async function handleArchiveWorkspace(workspaceId: string) {
    setLoading(true)
    const ok = await archiveWorkspace(workspaceId)
    if (!ok) {
      setError('Failed to archive workspace')
      setLoading(false)
      return
    }

    setWorkspaceHistory((current) => {
      const next = current.filter((workspace) => workspace.id !== workspaceId)
      try {
        localStorage.setItem(WORKSPACE_HISTORY_KEY, JSON.stringify(next))
      } catch {
        // Ignore persistence errors.
      }
      return next
    })

    if (exploration?.id === workspaceId) {
      setExploration(null)
      setSelectedOpportunityId(null)
      setStrategyHistory([])
    }
    setLoading(false)
  }

  useEffect(() => {
    if (exploration || workspaceHistory.length === 0) return
    const latest = workspaceHistory[0]
    void handleSelectWorkspace(latest.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceHistory, exploration])

  function handleExampleSelect(example: string) {
    setTopic(example)
  }

  return (
    <div className="appShell">
      <AppHeader />

      <main className="appGrid">
        <div className="mainColumn">
          <LaunchPanel
            topic={topic}
            outputGoal={outputGoal}
            constraints={constraints}
            loading={loading}
            examples={EXAMPLE_TOPICS}
            onTopicChange={setTopic}
            onOutputGoalChange={setOutputGoal}
            onConstraintsChange={setConstraints}
            onExampleSelect={handleExampleSelect}
            onSubmit={handleSubmit}
          />
          <WorkspaceManager
            workspaces={workspaceHistory}
            activeWorkspaceId={exploration?.id}
            loading={loading}
            onSelectWorkspace={handleSelectWorkspace}
            onArchiveWorkspace={handleArchiveWorkspace}
          />

          {error ? <p className="errorBanner">{error}</p> : null}

          {exploration && view ? (
            <WorkbenchColumns
              session={{
                ...exploration,
                activeOpportunityId: selectedOpportunityId ?? exploration.activeOpportunityId,
              }}
              view={view}
              onSelectOpportunity={handleSelectOpportunity}
              onExpandOpportunity={handleExpandOpportunity}
              onToggleFavorite={handleToggleFavorite}
            />
          ) : null}
        </div>

        {view ? (
          <SidebarPanel
            savedIdeas={savedIdeas}
            view={view}
            strategy={exploration?.strategy}
            strategyBusy={strategyUpdating}
            onUpdateStrategy={handleUpdateStrategy}
            strategyHistory={strategyHistory}
            onRollbackStrategy={handleRollbackStrategy}
            onToggleFavorite={handleToggleFavorite}
          />
        ) : null}
      </main>
    </div>
  )
}

export default App
