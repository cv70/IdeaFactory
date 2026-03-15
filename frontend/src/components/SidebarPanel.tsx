import { useEffect, useState } from 'react'
import type { Node, WorkbenchView } from '../types/exploration'
import type { RuntimeStrategy } from '../types/exploration'

type SidebarPanelProps = {
  savedIdeas: Node[]
  view: WorkbenchView
  strategy?: RuntimeStrategy
  strategyBusy?: boolean
  strategyHistory: Array<{
    id: string
    createdAt: number
    strategy: RuntimeStrategy
  }>
  onUpdateStrategy: (strategy: {
    interval_ms?: number
    max_runs?: number
    expansion_mode?: 'active' | 'round_robin'
    preferred_branch_id?: string
  }) => void
  onRollbackStrategy: (strategy: RuntimeStrategy) => void
  onToggleFavorite: (idea: Node) => void
  onSubmitIntervention: (intent: string) => void
  lastInterventionIntent?: string
  lastInterventionStatus?: string
}

export function SidebarPanel(props: SidebarPanelProps) {
  const [intervalMs, setIntervalMs] = useState(String(props.strategy?.interval_ms ?? 4000))
  const [maxRuns, setMaxRuns] = useState(String(props.strategy?.max_runs ?? 30))
  const [expansionMode, setExpansionMode] = useState<'active' | 'round_robin'>(
    props.strategy?.expansion_mode ?? 'active',
  )
  const [preferredBranchId, setPreferredBranchId] = useState(props.strategy?.preferred_branch_id ?? '')
  const [interventionText, setInterventionText] = useState('')

  useEffect(() => {
    setIntervalMs(String(props.strategy?.interval_ms ?? 4000))
    setMaxRuns(String(props.strategy?.max_runs ?? 30))
    setExpansionMode(props.strategy?.expansion_mode ?? 'active')
    setPreferredBranchId(props.strategy?.preferred_branch_id ?? '')
  }, [props.strategy?.interval_ms, props.strategy?.max_runs, props.strategy?.expansion_mode, props.strategy?.preferred_branch_id])

  function applyCurrentStrategy() {
    props.onUpdateStrategy({
      interval_ms: Number(intervalMs),
      max_runs: Number(maxRuns),
      expansion_mode: expansionMode,
      preferred_branch_id: preferredBranchId || undefined,
    })
  }

  function applyPreset(preset: 'balanced' | 'rapid' | 'focused') {
    if (preset === 'rapid') {
      props.onUpdateStrategy({
        interval_ms: 1200,
        max_runs: 60,
        expansion_mode: 'round_robin',
        preferred_branch_id: undefined,
      })
      return
    }
    if (preset === 'focused') {
      props.onUpdateStrategy({
        interval_ms: 2200,
        max_runs: 50,
        expansion_mode: 'active',
        preferred_branch_id: props.view.activeOpportunity.id,
      })
      return
    }
    props.onUpdateStrategy({
      interval_ms: 4000,
      max_runs: 30,
      expansion_mode: 'active',
      preferred_branch_id: undefined,
    })
  }

  return (
    <aside className="sidebarPanel">
      <section className="sidebarSection">
        <p className="sectionLabel">Intervention</p>
        <h2>Submit intervention</h2>
        <div className="strategyPanel">
          <label htmlFor="intervention">Intervention</label>
          <textarea
            id="intervention"
            value={interventionText}
            onChange={(event) => setInterventionText(event.target.value)}
          />
          <button
            type="button"
            onClick={() => {
              props.onSubmitIntervention(interventionText)
              setInterventionText('')
            }}
          >
            Submit intervention
          </button>
          {props.lastInterventionStatus ? (
            <p>Status: {props.lastInterventionStatus}</p>
          ) : null}
          {props.lastInterventionIntent ? (
            <p>{props.lastInterventionIntent}</p>
          ) : null}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">Governance</p>
        <h2>Runtime strategy</h2>
        <div className="strategyPanel">
          <p className="strategySummary">
            Live strategy:
            {' '}
            {props.strategy?.expansion_mode ?? 'active'}
            {' '}
            /
            {' '}
            {props.strategy?.interval_ms ?? 4000}
            ms
            {' '}
            /
            {' '}
            {props.strategy?.max_runs ?? 30}
            runs
          </p>

          <div className="strategyPresetRow">
            <button type="button" className="chipButton" onClick={() => applyPreset('balanced')} disabled={props.strategyBusy}>
              Balanced
            </button>
            <button type="button" className="chipButton" onClick={() => applyPreset('rapid')} disabled={props.strategyBusy}>
              Rapid Scan
            </button>
            <button type="button" className="chipButton" onClick={() => applyPreset('focused')} disabled={props.strategyBusy}>
              Focus Active
            </button>
          </div>

          <label className="field">
            <span>Interval (ms)</span>
            <input
              type="number"
              min={500}
              step={100}
              value={intervalMs}
              onChange={(event) => setIntervalMs(event.target.value)}
            />
          </label>

          <label className="field">
            <span>Max runs</span>
            <input
              type="number"
              min={1}
              step={1}
              value={maxRuns}
              onChange={(event) => setMaxRuns(event.target.value)}
            />
          </label>

          <label className="field">
            <span>Expansion mode</span>
            <select
              className="strategySelect"
              value={expansionMode}
              onChange={(event) => setExpansionMode(event.target.value as 'active' | 'round_robin')}
            >
              <option value="active">Active branch</option>
              <option value="round_robin">Round robin</option>
            </select>
          </label>

          <label className="field">
            <span>Preferred branch</span>
            <select
              className="strategySelect"
              value={preferredBranchId}
              onChange={(event) => setPreferredBranchId(event.target.value)}
            >
              <option value="">None</option>
              {props.view.opportunities.map((opportunity) => (
                <option key={opportunity.id} value={opportunity.id}>
                  {opportunity.title}
                </option>
              ))}
            </select>
          </label>

          <button type="button" className="secondaryAction" onClick={applyCurrentStrategy} disabled={props.strategyBusy}>
            {props.strategyBusy ? 'Updating strategy...' : 'Apply strategy'}
          </button>
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">History</p>
        <h2>Strategy history</h2>
        <div className="stackList">
          {props.strategyHistory.length === 0 ? (
            <p className="emptyState">No strategy updates yet.</p>
          ) : (
            props.strategyHistory.map((entry) => (
              <article key={entry.id} className="runCard">
                <p className="detailLabel">
                  {new Date(entry.createdAt).toLocaleTimeString()}
                </p>
                <p>
                  {entry.strategy.expansion_mode}
                  {' '}
                  /
                  {' '}
                  {entry.strategy.interval_ms}
                  ms
                  {' '}
                  /
                  {' '}
                  {entry.strategy.max_runs}
                  runs
                </p>
                <button
                  type="button"
                  className="miniAction"
                  onClick={() => props.onRollbackStrategy(entry.strategy)}
                  disabled={props.strategyBusy}
                >
                  Rollback
                </button>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">Saved</p>
        <h2>Saved ideas ({props.savedIdeas.length})</h2>
        <div className="stackList">
          {props.savedIdeas.length === 0 ? (
            <p className="emptyState">No saved ideas yet.</p>
          ) : (
            props.savedIdeas.map((idea) => (
              <article key={idea.id} className="savedCard">
                <h3>{idea.title}</h3>
                <p>{idea.summary}</p>
                <button
                  type="button"
                  className="miniAction"
                  onClick={() => props.onToggleFavorite(idea)}
                >
                  Unsave idea
                </button>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">Runs</p>
        <h2>Recent run notes</h2>
        <div className="stackList">
          {props.view.runNotes.map((run) => (
            <article key={run.id} className="runCard">
              <p className="detailLabel">Round {run.round}</p>
              <p>{run.summary}</p>
            </article>
          ))}
        </div>
      </section>
    </aside>
  )
}
