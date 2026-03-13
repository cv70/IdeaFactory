import type { FormEvent } from 'react'

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
  return (
    <section className="launchPanel">
      <div className="sectionIntro">
        <p className="sectionLabel">Launch</p>
        <h2>Start from a theme, not a template</h2>
        <p>
          Keep the input lightweight. The system will turn it into opportunities, questions,
          hypotheses, and materialized ideas.
        </p>
      </div>

      <form className="launchForm" onSubmit={props.onSubmit}>
        <label className="field">
          <span>Topic</span>
          <input
            value={props.topic}
            onChange={(event) => props.onTopicChange(event.target.value)}
            placeholder="AI education, oncology screening, creator infrastructure"
          />
        </label>

        <label className="field">
          <span>Output goal</span>
          <input
            value={props.outputGoal}
            onChange={(event) => props.onOutputGoalChange(event.target.value)}
            placeholder="Research directions, venture opportunities, product bets"
          />
        </label>

        <label className="field">
          <span>Constraints</span>
          <textarea
            value={props.constraints}
            onChange={(event) => props.onConstraintsChange(event.target.value)}
            rows={3}
            placeholder="Low-cost, explainable, easy to validate"
          />
        </label>

        <div className="launchActions">
          <button type="submit" className="primaryAction" disabled={props.loading}>
            {props.loading ? 'Starting...' : 'Start exploration'}
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
