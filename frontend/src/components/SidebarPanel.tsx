import type { Node, WorkbenchView } from '../types/exploration'

type SidebarPanelProps = {
  savedIdeas: Node[]
  view: WorkbenchView
  onToggleFavorite: (idea: Node) => void
}

export function SidebarPanel(props: SidebarPanelProps) {
  return (
    <aside className="sidebarPanel">
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
