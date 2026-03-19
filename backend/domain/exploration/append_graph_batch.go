package exploration

import (
	"backend/agentools"
	"context"
	"fmt"
	"strings"
)

func (d *ExplorationDomain) AppendGraphBatch(_ context.Context, req agentools.AppendGraphBatchParams) (agentools.AppendGraphBatchResult, error) {
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return agentools.AppendGraphBatchResult{}, fmt.Errorf("workspace_id is required")
	}

	newNodes, err := d.normalizeGraphBatchNodes(workspaceID, req.Nodes)
	if err != nil {
		return agentools.AppendGraphBatchResult{}, err
	}
	newEdges, err := d.normalizeGraphBatchEdges(req.Edges)
	if err != nil {
		return agentools.AppendGraphBatchResult{}, err
	}

	mutations, err := d.appendGraphBatch(workspaceID, newNodes, newEdges, string(MutationSourceRuntime))
	if err != nil {
		return agentools.AppendGraphBatchResult{}, err
	}

	mutationIDs := make([]string, 0, len(mutations))
	for _, mutation := range mutations {
		mutationIDs = append(mutationIDs, mutation.ID)
	}

	return agentools.AppendGraphBatchResult{
		WorkspaceID:  workspaceID,
		AppliedNodes: len(newNodes),
		AppliedEdges: len(newEdges),
		MutationIDs:  mutationIDs,
		Summary:      fmt.Sprintf("%s added %d nodes and %d edges", agentools.ToolAppendGraphBatch, len(newNodes), len(newEdges)),
	}, nil
}

func (d *ExplorationDomain) normalizeGraphBatchNodes(workspaceID string, inputs []agentools.AppendGraphBatchNode) ([]Node, error) {
	allowedTypes := make(map[NodeType]struct{}, len(validNodeTypes()))
	for _, nodeType := range validNodeTypes() {
		allowedTypes[nodeType] = struct{}{}
	}
	allowedStatuses := make(map[NodeStatus]struct{}, len(validNodeStatuses()))
	for _, status := range validNodeStatuses() {
		allowedStatuses[status] = struct{}{}
	}

	nodes := make([]Node, 0, len(inputs))
	for _, input := range inputs {
		nodeID := strings.TrimSpace(input.ID)
		if nodeID == "" {
			return nil, fmt.Errorf("node id is required")
		}
		nodeType := NodeType(strings.TrimSpace(input.Type))
		if _, ok := allowedTypes[nodeType]; !ok {
			return nil, fmt.Errorf("invalid node type %q", input.Type)
		}
		status := NodeStatus(strings.TrimSpace(input.Status))
		if status == "" {
			status = NodeActive
		}
		if _, ok := allowedStatuses[status]; !ok {
			return nil, fmt.Errorf("invalid node status %q", input.Status)
		}

		node := Node{
			ID:              nodeID,
			WorkspaceID:     workspaceID,
			SessionID:       workspaceID,
			Type:            nodeType,
			Title:           strings.TrimSpace(input.Title),
			Summary:         strings.TrimSpace(input.Summary),
			Status:          status,
			Score:           input.Score,
			Depth:           input.Depth,
			ParentContext:   strings.TrimSpace(input.ParentContext),
			EvidenceSummary: strings.TrimSpace(input.EvidenceSummary),
		}
		if node.Title == "" {
			return nil, fmt.Errorf("node %q title is required", nodeID)
		}
		if node.Summary == "" {
			return nil, fmt.Errorf("node %q summary is required", nodeID)
		}
		if node.Depth < 0 {
			return nil, fmt.Errorf("node %q depth must be >= 0", nodeID)
		}
		if branchID, ok := input.Metadata["branchId"].(string); ok {
			node.Metadata.BranchID = strings.TrimSpace(branchID)
		}
		if slot, ok := input.Metadata["slot"].(string); ok {
			node.Metadata.Slot = strings.TrimSpace(slot)
		}
		if cluster, ok := input.Metadata["cluster"].(string); ok {
			node.Metadata.Cluster = strings.TrimSpace(cluster)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (d *ExplorationDomain) normalizeGraphBatchEdges(inputs []agentools.AppendGraphBatchEdge) ([]Edge, error) {
	allowedTypes := make(map[EdgeType]struct{}, len(validEdgeTypes()))
	for _, edgeType := range validEdgeTypes() {
		allowedTypes[edgeType] = struct{}{}
	}

	edges := make([]Edge, 0, len(inputs))
	for _, input := range inputs {
		edgeID := strings.TrimSpace(input.ID)
		if edgeID == "" {
			return nil, fmt.Errorf("edge id is required")
		}
		edgeType := EdgeType(strings.TrimSpace(input.Type))
		if _, ok := allowedTypes[edgeType]; !ok {
			return nil, fmt.Errorf("invalid edge type %q", input.Type)
		}

		edge := Edge{
			ID:   edgeID,
			From: strings.TrimSpace(input.From),
			To:   strings.TrimSpace(input.To),
			Type: edgeType,
		}
		if edge.From == "" || edge.To == "" {
			return nil, fmt.Errorf("edge %q endpoints are required", edgeID)
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func (d *ExplorationDomain) appendGraphBatch(workspaceID string, newNodes []Node, newEdges []Edge, source string) ([]MutationEvent, error) {
	if len(newNodes) == 0 && len(newEdges) == 0 {
		return nil, nil
	}

	d.store.mu.Lock()
	current, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		return nil, fmt.Errorf("workspace %q not found", workspaceID)
	}

	existingNodeIDs := make(map[string]struct{}, len(current.Nodes))
	for _, node := range current.Nodes {
		existingNodeIDs[node.ID] = struct{}{}
	}
	existingEdgeIDs := make(map[string]struct{}, len(current.Edges))
	for _, edge := range current.Edges {
		existingEdgeIDs[edge.ID] = struct{}{}
	}

	batchNodeIDs := make(map[string]struct{}, len(newNodes))
	for i := range newNodes {
		node := &newNodes[i]
		node.WorkspaceID = workspaceID
		node.SessionID = workspaceID
		if _, ok := existingNodeIDs[node.ID]; ok {
			d.store.mu.Unlock()
			return nil, fmt.Errorf("node %q already exists", node.ID)
		}
		if _, ok := batchNodeIDs[node.ID]; ok {
			d.store.mu.Unlock()
			return nil, fmt.Errorf("duplicate node id %q in batch", node.ID)
		}
		batchNodeIDs[node.ID] = struct{}{}
	}

	for i := range newEdges {
		edge := &newEdges[i]
		edge.WorkspaceID = workspaceID
		if _, ok := existingEdgeIDs[edge.ID]; ok {
			d.store.mu.Unlock()
			return nil, fmt.Errorf("edge %q already exists", edge.ID)
		}
		if _, ok := batchNodeIDs[edge.From]; !ok {
			if _, ok := existingNodeIDs[edge.From]; !ok {
				d.store.mu.Unlock()
				return nil, fmt.Errorf("edge %q references unknown from node %q", edge.ID, edge.From)
			}
		}
		if _, ok := batchNodeIDs[edge.To]; !ok {
			if _, ok := existingNodeIDs[edge.To]; !ok {
				d.store.mu.Unlock()
				return nil, fmt.Errorf("edge %q references unknown to node %q", edge.ID, edge.To)
			}
		}
		if _, ok := existingEdgeIDs[edge.ID]; ok {
			d.store.mu.Unlock()
			return nil, fmt.Errorf("edge %q already exists", edge.ID)
		}
		existingEdgeIDs[edge.ID] = struct{}{}
	}

	prev := *current
	current.Nodes = append(current.Nodes, newNodes...)
	current.Edges = append(current.Edges, newEdges...)
	updatedCopy := *current
	mutations := diffMutations(prev, *current, source)
	d.store.mu.Unlock()

	if err := d.persistWorkspace(updatedCopy); err != nil {
		d.store.mu.Lock()
		if existing, ok := d.store.workspaces[workspaceID]; ok {
			*existing = prev
		}
		d.store.mu.Unlock()
		return nil, err
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Mutations = append(state.Mutations, mutations...)
	})
	d.broadcastMutations(workspaceID, mutations)
	return mutations, nil
}
