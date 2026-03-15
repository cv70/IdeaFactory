package exploration

import "time"

func (d *ExplorationDomain) ListWorkspaces(limit int) ([]WorkspaceSummary, error) {
	if d.DB == nil {
		d.store.mu.RLock()
		out := make([]WorkspaceSummary, 0, len(d.store.workspaces))
		for _, session := range d.store.workspaces {
			out = append(out, WorkspaceSummary{
				ID:         session.ID,
				Topic:      session.Topic,
				OutputGoal: session.OutputGoal,
				UpdatedAt:  time.Now().UnixMilli(),
			})
		}
		d.store.mu.RUnlock()
		return out, nil
	}

	states, err := d.DB.ListWorkspaceStates(limit, false)
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceSummary, 0, len(states))
	for _, state := range states {
		out = append(out, WorkspaceSummary{
			ID:         state.WorkspaceID,
			Topic:      state.Topic,
			OutputGoal: state.OutputGoal,
			UpdatedAt:  state.UpdatedAt.UnixMilli(),
		})
	}
	return out, nil
}

func (d *ExplorationDomain) ArchiveWorkspace(workspaceID string) bool {
	exists := false
	d.store.mu.RLock()
	_, exists = d.store.workspaces[workspaceID]
	d.store.mu.RUnlock()
	if !exists {
		if loaded, ok := d.loadWorkspace(workspaceID); ok && loaded != nil {
			exists = true
		}
	}
	if !exists {
		return false
	}

	if d.DB != nil {
		if err := d.DB.ArchiveWorkspaceState(workspaceID); err != nil {
			return false
		}
	}

	d.store.mu.Lock()
	delete(d.store.workspaces, workspaceID)
	d.store.mu.Unlock()

	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Running = false
		state.Cursor = 0
		state.Interventions = map[string]InterventionView{}
	})

	return true
}
