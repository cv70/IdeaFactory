package exploration

import (
	"fmt"
	"slices"
	"time"
)

func mutationID(workspaceID string) string {
	return fmt.Sprintf("mutation-%s-%d", workspaceID, time.Now().UnixNano())
}

func diffMutations(prev ExplorationSession, next ExplorationSession, source string) []MutationEvent {
	out := make([]MutationEvent, 0)

	if prev.ActiveOpportunityID != next.ActiveOpportunityID {
		out = append(out, MutationEvent{
			ID:                  mutationID(next.ID),
			WorkspaceID:         next.ID,
			Kind:                string(MutationKindActiveOpportunitySet),
			Source:              source,
			ActiveOpportunityID: next.ActiveOpportunityID,
			CreatedAt:           time.Now().UnixMilli(),
		})
	}

	prevNodeIDs := map[string]struct{}{}
	for _, node := range prev.Nodes {
		prevNodeIDs[node.ID] = struct{}{}
	}
	for _, node := range next.Nodes {
		if _, ok := prevNodeIDs[node.ID]; ok {
			continue
		}
		copyNode := node
		out = append(out, MutationEvent{
			ID:          mutationID(next.ID),
			WorkspaceID: next.ID,
			Kind:        string(MutationKindNodeAdded),
			Source:      source,
			Node:        &copyNode,
			CreatedAt:   time.Now().UnixMilli(),
		})
	}

	prevEdgeIDs := map[string]struct{}{}
	for _, edge := range prev.Edges {
		prevEdgeIDs[edge.ID] = struct{}{}
	}
	for _, edge := range next.Edges {
		if _, ok := prevEdgeIDs[edge.ID]; ok {
			continue
		}
		copyEdge := edge
		out = append(out, MutationEvent{
			ID:          mutationID(next.ID),
			WorkspaceID: next.ID,
			Kind:        string(MutationKindEdgeAdded),
			Source:      source,
			Edge:        &copyEdge,
			CreatedAt:   time.Now().UnixMilli(),
		})
	}

	if len(next.Runs) > len(prev.Runs) {
		for _, run := range next.Runs[len(prev.Runs):] {
			copyRun := run
			out = append(out, MutationEvent{
				ID:          mutationID(next.ID),
				WorkspaceID: next.ID,
				Kind:        "run_added",
				Source:      source,
				Run:         &copyRun,
				CreatedAt:   time.Now().UnixMilli(),
			})
		}
	}

	if !slices.Equal(prev.Favorites, next.Favorites) {
		out = append(out, MutationEvent{
			ID:          mutationID(next.ID),
			WorkspaceID: next.ID,
			Kind:        string(MutationKindFavoritesUpdated),
			Source:      source,
			Favorites:   append([]string{}, next.Favorites...),
			CreatedAt:   time.Now().UnixMilli(),
		})
	}

	if prev.Strategy != next.Strategy {
		strategy := next.Strategy
		out = append(out, MutationEvent{
			ID:          mutationID(next.ID),
			WorkspaceID: next.ID,
			Kind:        string(MutationKindStrategyUpdated),
			Source:      source,
			Strategy:    &strategy,
			CreatedAt:   time.Now().UnixMilli(),
		})
	}

	return out
}
