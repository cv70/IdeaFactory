import type { ExplorationSession, Node, WorkbenchView } from '../types/exploration'

type WorkbenchColumnsProps = {
  session: ExplorationSession
  view: WorkbenchView
  onSelectOpportunity: (opportunity: Node) => void
  onExpandOpportunity: (opportunity: Node) => void
  onToggleFavorite: (idea: Node) => void
}

export function WorkbenchColumns(props: WorkbenchColumnsProps) {
  const { session, view } = props

  return (
    <section className="workbench">
      <article className="columnPanel">
        <div className="sectionIntro">
          <p className="sectionLabel">Map</p>
          <h2>Direction map</h2>
          <p>{session.topic} reframed as expandable exploration branches.</p>
        </div>

        <div className="stackList">
          {view.opportunities.map((opportunity) => {
            const selected = opportunity.id === session.activeOpportunityId
            return (
              <button
                key={opportunity.id}
                type="button"
                className={selected ? 'branchCard branchCardActive' : 'branchCard'}
                aria-pressed={selected}
                onClick={() => props.onSelectOpportunity(opportunity)}
              >
                <span className="branchTitle">{opportunity.title}</span>
                <span className="branchSummary">{opportunity.summary}</span>
              </button>
            )
          })}
        </div>
      </article>

      <article className="columnPanel">
        <div className="sectionIntro">
          <p className="sectionLabel">Reasoning</p>
          <h2>Question trail</h2>
          <p>{view.activeOpportunity.summary}</p>
        </div>

        <div className="stackList">
          {view.questionTrail.map((question) => (
            <section key={question.id} className="detailCard">
              <p className="detailLabel">Question</p>
              <h3>{question.title.replace(`${session.topic}: `, '')}</h3>
              <p>{question.summary}</p>
            </section>
          ))}
          {view.hypothesisTrail.map((hypothesis) => (
            <section key={hypothesis.id} className="detailCard accentCard">
              <p className="detailLabel">Hypothesis</p>
              <h3>{hypothesis.title}</h3>
              <p>{hypothesis.summary}</p>
            </section>
          ))}
        </div>

        <button
          type="button"
          className="secondaryAction"
          onClick={() => props.onExpandOpportunity(view.activeOpportunity)}
        >
          Expand this branch
        </button>
      </article>

      <article className="columnPanel">
        <div className="sectionIntro">
          <p className="sectionLabel">Materialization</p>
          <h2>Materialized ideas</h2>
          <p>Concrete outputs growing from the currently selected path.</p>
        </div>

        <div className="stackList">
          {view.ideaCards.map((idea) => {
            const favored = session.favorites.includes(idea.id)
            return (
              <section key={idea.id} className="ideaCard">
                <div className="ideaHeader">
                  <h3>{idea.title}</h3>
                  <button
                    type="button"
                    className="miniAction"
                    onClick={() => props.onToggleFavorite(idea)}
                  >
                    {favored ? 'Unsave idea' : 'Save idea'}
                  </button>
                </div>
                <p>{idea.summary}</p>
                <p className="evidenceText">{idea.evidenceSummary}</p>
              </section>
            )
          })}
        </div>
      </article>
    </section>
  )
}
