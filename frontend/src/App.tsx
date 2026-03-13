import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'
import { AppHeader } from './components/AppHeader'
import { LaunchPanel } from './components/LaunchPanel'
import { SidebarPanel } from './components/SidebarPanel'
import { WorkbenchColumns } from './components/WorkbenchColumns'
import { DEFAULT_CONSTRAINTS, EXAMPLE_TOPICS } from './data/mockExploration'
import {
  createExploration,
  expandOpportunity as expandOpportunityRequest,
  sendFeedback,
} from './lib/explorationApi'
import { buildWorkbenchView } from './lib/workbench'
import type { ExplorationSession, Node } from './types/exploration'

function App() {
  const [topic, setTopic] = useState('')
  const [outputGoal, setOutputGoal] = useState('')
  const [constraints, setConstraints] = useState(DEFAULT_CONSTRAINTS)
  const [exploration, setExploration] = useState<ExplorationSession | null>(null)
  const [selectedOpportunityId, setSelectedOpportunityId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

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
            onToggleFavorite={handleToggleFavorite}
          />
        ) : null}
      </main>
    </div>
  )
}

export default App
