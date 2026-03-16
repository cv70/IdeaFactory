import type { FormEvent } from 'react'
import { useTranslation } from '../lib/i18n'

type LaunchPanelProps = {
  topic: string
  outputGoal: string
  constraints: string
  loading?: boolean
  examples: string[]
  onTopicChange: (value: string) => void
  onOutputGoalChange: (value: string) => void
  onConstraintsChange: (value: string) => void
  onExampleSelect: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function LaunchPanel(props: LaunchPanelProps) {
  const { t } = useTranslation()

  return (
    <section className="launchPanel">
      <div className="sectionIntro">
        <p className="sectionLabel">{t('launch.label')}</p>
        <h2>{t('launch.title')}</h2>
        <p>{t('launch.description')}</p>
      </div>

      <form className="launchForm" onSubmit={props.onSubmit}>
        <label className="field">
          <span>{t('launch.topic')}</span>
          <input
            value={props.topic}
            onChange={(event) => props.onTopicChange(event.target.value)}
            placeholder={t('launch.topicPlaceholder')}
          />
        </label>

        <label className="field">
          <span>{t('launch.outputGoal')}</span>
          <input
            value={props.outputGoal}
            onChange={(event) => props.onOutputGoalChange(event.target.value)}
            placeholder={t('launch.outputGoalPlaceholder')}
          />
        </label>

        <label className="field">
          <span>{t('launch.constraints')}</span>
          <textarea
            value={props.constraints}
            onChange={(event) => props.onConstraintsChange(event.target.value)}
            rows={3}
            placeholder={t('launch.constraintsPlaceholder')}
          />
        </label>

        <div className="launchActions">
          <button type="submit" className="primaryAction" disabled={props.loading}>
            {props.loading ? t('launch.startingButton') : t('launch.startButton')}
          </button>
          <div className="exampleRow" aria-label="Example topics">
            {props.examples.map((example) => (
              <button
                key={example}
                type="button"
                className="chipButton"
                onClick={() => props.onExampleSelect(example)}
              >
                {example}
              </button>
            ))}
          </div>
        </div>
      </form>
    </section>
  )
}
