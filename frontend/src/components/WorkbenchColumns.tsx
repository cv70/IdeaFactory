import type { ExplorationSession, Node, WorkbenchView } from '../types/exploration'
import { useTranslation } from '../lib/i18n'

type WorkbenchColumnsProps = {
  session: ExplorationSession
  view: WorkbenchView
  onSelectOpportunity: (opportunity: Node) => void
  onExpandOpportunity: (opportunity: Node) => void
  onToggleFavorite: (idea: Node) => void
}

export function WorkbenchColumns(props: WorkbenchColumnsProps) {
  const { session, view } = props
  const { t } = useTranslation()

  return (
    <section className="workbench">
      <article className="columnPanel">
        <div className="sectionIntro">
          <p className="sectionLabel">{t('map.label')}</p>
          <h2>{t('map.title')}</h2>
          <p>{session.topic}</p>
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
          <p className="sectionLabel">{t('reasoning.label')}</p>
          <h2>{t('reasoning.title')}</h2>
          <p>{view.activeOpportunity.summary}</p>
        </div>

        <div className="stackList">
          {view.questionTrail.map((question) => (
            <section key={question.id} className="detailCard">
              <p className="detailLabel">{t('map.question')}</p>
              <h3>{question.title.replace(`${session.topic}: `, '')}</h3>
              <p>{question.summary}</p>
            </section>
          ))}
          {view.hypothesisTrail.map((hypothesis) => (
            <section key={hypothesis.id} className="detailCard accentCard">
              <p className="detailLabel">{t('map.hypothesis')}</p>
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
          {t('map.expandButton')}
        </button>
      </article>

      <article className="columnPanel">
        <div className="sectionIntro">
          <p className="sectionLabel">{t('materialization.label')}</p>
          <h2>{t('materialization.title')}</h2>
          <p>{t('materialization.description')}</p>
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
                    {favored ? t('idea.unsave') : t('idea.save')}
                  </button>
                </div>
                <p>{idea.summary}</p>
                {idea.evidenceSummary ? (
                  <p className="evidenceText">{idea.evidenceSummary}</p>
                ) : null}
              </section>
            )
          })}
        </div>
      </article>
    </section>
  )
}
