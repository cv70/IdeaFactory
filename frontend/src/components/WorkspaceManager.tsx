type WorkspaceRecord = {
  id: string
  topic: string
  updatedAt: number
}

type WorkspaceManagerProps = {
  workspaces: WorkspaceRecord[]
  activeWorkspaceId?: string
  loading?: boolean
  onSelectWorkspace: (workspaceId: string) => void
  onArchiveWorkspace: (workspaceId: string) => void
}

export function WorkspaceManager(props: WorkspaceManagerProps) {
  return (
    <section className="launchPanel">
      <div className="sectionIntro">
        <p className="sectionLabel">Workspaces</p>
        <h2>Switch and recover</h2>
        <p>Continue from previous exploration workspaces without starting over.</p>
      </div>

      <div className="workspaceList">
        {props.workspaces.length === 0 ? (
          <p className="emptyState">No historical workspaces yet.</p>
        ) : (
          props.workspaces.map((workspace) => {
            const active = workspace.id === props.activeWorkspaceId
            return (
              <article
                key={workspace.id}
                className={active ? 'workspaceCard workspaceCardActive' : 'workspaceCard'}
              >
                <span className="branchTitle">{workspace.topic}</span>
                <span className="branchSummary">
                  {new Date(workspace.updatedAt).toLocaleString()}
                </span>
                <div className="workspaceActions">
                  <button
                    type="button"
                    className="miniAction"
                    aria-label={`Open workspace ${workspace.topic}`}
                    onClick={() => props.onSelectWorkspace(workspace.id)}
                    disabled={props.loading}
                  >
                    Open
                  </button>
                  <button
                    type="button"
                    className="miniAction"
                    onClick={() => props.onArchiveWorkspace(workspace.id)}
                    disabled={props.loading}
                  >
                    Archive
                  </button>
                </div>
              </article>
            )
          })
        )}
      </div>
    </section>
  )
}
