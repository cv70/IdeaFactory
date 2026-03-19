import { useEffect, useState } from 'react'
import type { AgentRunEvent, Node, WorkbenchView } from '../types/exploration'
import type { RuntimeStrategy } from '../types/exploration'
import { useTranslation } from '../lib/i18n'

type TranslateFn = ReturnType<typeof useTranslation>['t']

function eventTypeLabel(eventType: string, t: TranslateFn) {
  switch (eventType) {
    case 'agent_start':
      return t('sidebar.agentLog.eventTypeStart')
    case 'agent_delegate':
      return t('sidebar.agentLog.eventTypeDelegate')
    case 'tool_call':
      return t('sidebar.agentLog.eventTypeTool')
    case 'run_summary':
      return t('sidebar.agentLog.eventTypeSummary')
    case 'run_error':
      return t('sidebar.agentLog.eventTypeError')
    default:
      return eventType
  }
}

function eventTypeClassName(eventType: string) {
  switch (eventType) {
    case 'agent_start':
      return 'agentEventTagStart'
    case 'agent_delegate':
      return 'agentEventTagDelegate'
    case 'tool_call':
      return 'agentEventTagTool'
    case 'run_summary':
      return 'agentEventTagSummary'
    case 'run_error':
      return 'agentEventTagError'
    default:
      return ''
  }
}

function eventDetail(event: AgentRunEvent) {
  if (event.event_type !== 'tool_call') return ''
  const argsSummary = event.payload?.args_summary
  return typeof argsSummary === 'string' ? argsSummary : ''
}

function formatEventTime(createdAt: number) {
  return new Date(createdAt).toISOString().slice(11, 19) + 'Z'
}

function groupEventsByRun(events: AgentRunEvent[]) {
  return events.reduce<Array<{ runId: string; events: AgentRunEvent[] }>>((groups, event) => {
    const runId = event.run_id || 'unknown'
    const currentGroup = groups[groups.length - 1]
    if (currentGroup && currentGroup.runId === runId) {
      currentGroup.events.push(event)
      return groups
    }
    groups.push({ runId, events: [event] })
    return groups
  }, [])
}

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
  agentEvents: AgentRunEvent[]
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
  const { t } = useTranslation()
  const [intervalMs, setIntervalMs] = useState(String(props.strategy?.interval_ms ?? 4000))
  const [maxRuns, setMaxRuns] = useState(String(props.strategy?.max_runs ?? 30))
  const [expansionMode, setExpansionMode] = useState<'active' | 'round_robin'>(
    props.strategy?.expansion_mode ?? 'active',
  )
  const [preferredBranchId, setPreferredBranchId] = useState(props.strategy?.preferred_branch_id ?? '')
  const [interventionText, setInterventionText] = useState('')

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect */
    setIntervalMs(String(props.strategy?.interval_ms ?? 4000))
    setMaxRuns(String(props.strategy?.max_runs ?? 30))
    setExpansionMode(props.strategy?.expansion_mode ?? 'active')
    setPreferredBranchId(props.strategy?.preferred_branch_id ?? '')
    /* eslint-enable react-hooks/set-state-in-effect */
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

  const groupedAgentEvents = groupEventsByRun(props.agentEvents)

  return (
    <aside className="sidebarPanel">
      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.intervention.label')}</p>
        <h2>{t('sidebar.intervention.title')}</h2>
        <div className="strategyPanel">
          <label className="field">
            <span>{t('sidebar.intervention.label')}</span>
            <textarea
              value={interventionText}
              onChange={(event) => setInterventionText(event.target.value)}
              placeholder={t('sidebar.intervention.placeholder')}
              rows={3}
            />
          </label>
          <button
            type="button"
            className="primaryAction"
            onClick={() => {
              props.onSubmitIntervention(interventionText)
              setInterventionText('')
            }}
            disabled={!interventionText.trim()}
          >
            {t('sidebar.intervention.button')}
          </button>
          {props.lastInterventionStatus ? (
            <p className="statusBadge">
              <span className="statusDot" />
              {t('sidebar.intervention.statusLabel')} {props.lastInterventionStatus}
            </p>
          ) : null}
          {props.lastInterventionIntent ? (
            <p className="interventionEcho">{props.lastInterventionIntent}</p>
          ) : null}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.strategy.label')}</p>
        <h2>{t('sidebar.strategy.title')}</h2>
        <div className="strategyPanel">
          <p className="strategySummary">
            {t('sidebar.strategy.livePrefix')}
            {' '}
            <strong>{props.strategy?.expansion_mode ?? 'active'}</strong>
            {' / '}
            <strong>{props.strategy?.interval_ms ?? 4000}ms</strong>
            {' / '}
            <strong>{props.strategy?.max_runs ?? 30}</strong>
          </p>

          <div className="strategyPresetRow">
            <button type="button" className="chipButton" onClick={() => applyPreset('balanced')} disabled={props.strategyBusy}>
              {t('sidebar.strategy.balanced')}
            </button>
            <button type="button" className="chipButton" onClick={() => applyPreset('rapid')} disabled={props.strategyBusy}>
              {t('sidebar.strategy.rapid')}
            </button>
            <button type="button" className="chipButton" onClick={() => applyPreset('focused')} disabled={props.strategyBusy}>
              {t('sidebar.strategy.focused')}
            </button>
          </div>

          <label className="field">
            <span>{t('sidebar.strategy.intervalMs')}</span>
            <input
              type="number"
              min={500}
              step={100}
              value={intervalMs}
              onChange={(event) => setIntervalMs(event.target.value)}
            />
          </label>

          <label className="field">
            <span>{t('sidebar.strategy.maxRuns')}</span>
            <input
              type="number"
              min={1}
              step={1}
              value={maxRuns}
              onChange={(event) => setMaxRuns(event.target.value)}
            />
          </label>

          <label className="field">
            <span>{t('sidebar.strategy.expansionMode')}</span>
            <select
              className="strategySelect"
              value={expansionMode}
              onChange={(event) => setExpansionMode(event.target.value as 'active' | 'round_robin')}
            >
              <option value="active">{t('sidebar.strategy.activeBranch')}</option>
              <option value="round_robin">{t('sidebar.strategy.roundRobin')}</option>
            </select>
          </label>

          <label className="field">
            <span>{t('sidebar.strategy.preferredBranch')}</span>
            <select
              className="strategySelect"
              value={preferredBranchId}
              onChange={(event) => setPreferredBranchId(event.target.value)}
            >
              <option value="">{t('sidebar.strategy.none')}</option>
              {props.view.opportunities.map((opportunity) => (
                <option key={opportunity.id} value={opportunity.id}>
                  {opportunity.title}
                </option>
              ))}
            </select>
          </label>

          <button
            type="button"
            className="secondaryAction"
            onClick={applyCurrentStrategy}
            disabled={props.strategyBusy}
          >
            {props.strategyBusy ? t('sidebar.strategy.applying') : t('sidebar.strategy.apply')}
          </button>
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.history.label')}</p>
        <h2>{t('sidebar.history.title')}</h2>
        <div className="stackList">
          {props.strategyHistory.length === 0 ? (
            <p className="emptyState">{t('sidebar.history.empty')}</p>
          ) : (
            props.strategyHistory.map((entry) => (
              <article key={entry.id} className="runCard">
                <p className="detailLabel">
                  {new Date(entry.createdAt).toLocaleTimeString()}
                </p>
                <p>
                  {entry.strategy.expansion_mode}
                  {' / '}
                  {entry.strategy.interval_ms}ms
                  {' / '}
                  {entry.strategy.max_runs}
                </p>
                <button
                  type="button"
                  className="miniAction"
                  onClick={() => props.onRollbackStrategy(entry.strategy)}
                  disabled={props.strategyBusy}
                >
                  {t('sidebar.history.rollback')}
                </button>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.saved.label')}</p>
        <h2>{t('sidebar.saved.titlePrefix')} ({props.savedIdeas.length})</h2>
        <div className="stackList">
          {props.savedIdeas.length === 0 ? (
            <p className="emptyState">{t('sidebar.saved.empty')}</p>
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
                  {t('idea.unsave')}
                </button>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.runs.label')}</p>
        <h2>{t('sidebar.runs.title')}</h2>
        <div className="stackList">
          {props.view.runNotes.map((run) => (
            <article key={run.id} className="runCard">
              <p className="detailLabel">{t('sidebar.runs.roundPrefix')} {run.round}</p>
              <p>{run.summary}</p>
              {run.timeline && run.timeline.length > 0 ? (
                <div className="timelineRow" aria-label="Runtime timeline">
                  {run.timeline.map((step) => (
                    <span key={`${run.id}-${step}`} className="timelineChip">
                      {step}
                    </span>
                  ))}
                </div>
              ) : null}
            </article>
          ))}
        </div>
      </section>

      <section className="sidebarSection">
        <p className="sectionLabel">{t('sidebar.agentLog.label')}</p>
        <h2>{t('sidebar.agentLog.title')}</h2>
        <div className="stackList">
          {props.agentEvents.length === 0 ? (
            <p className="emptyState">{t('sidebar.agentLog.empty')}</p>
          ) : (
            groupedAgentEvents.map((group) => (
              <div key={group.runId} className="agentEventGroup">
                <p className="detailLabel agentEventGroupLabel">
                  {t('sidebar.agentLog.runPrefix')} {group.runId}
                </p>
                <div className="stackList">
                  {group.events.map((event) => (
                    <article key={event.id} className="runCard agentEventCard">
                      <div className="agentEventHeader">
                        <span className={`agentEventTag ${eventTypeClassName(event.event_type)}`}>
                          {eventTypeLabel(event.event_type, t)}
                        </span>
                        <p className="detailLabel">
                          {event.actor}
                          {event.target ? ` -> ${event.target}` : ''}
                        </p>
                        <span className="agentEventTime">{formatEventTime(event.created_at)}</span>
                      </div>
                      <p>{event.summary}</p>
                      {eventDetail(event) ? (
                        <p className="agentEventDetail">{eventDetail(event)}</p>
                      ) : null}
                    </article>
                  ))}
                </div>
              </div>
            ))
          )}
        </div>
      </section>
    </aside>
  )
}
