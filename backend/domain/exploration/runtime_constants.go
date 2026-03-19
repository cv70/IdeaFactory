package exploration

type MutationSource string

const (
	MutationSourceRuntime   MutationSource = "runtime"
	MutationSourceWorkspace MutationSource = "workspace"
)

type MutationKind string

const (
	MutationKindRunCreated           MutationKind = "run_created"
	MutationKindRunCompleted         MutationKind = "run_completed"
	MutationKindRunFailed            MutationKind = "run_failed"
	MutationKindNodeAdded            MutationKind = "node_added"
	MutationKindEdgeAdded            MutationKind = "edge_added"
	MutationKindInterventionAbsorbed MutationKind = "intervention_absorbed"
	MutationKindBalanceUpdated       MutationKind = "balance_updated"
	MutationKindActiveOpportunitySet MutationKind = "active_opportunity_set"
	MutationKindFavoritesUpdated     MutationKind = "favorites_updated"
	MutationKindStrategyUpdated      MutationKind = "strategy_updated"
)

type RuntimeActor string

const (
	RuntimeActorMainAgent RuntimeActor = "main_agent"
)

type RunSource string

const (
	RunSourceManual          RunSource = "manual"
	RunSourceAuto            RunSource = "auto"
	RunSourceResume          RunSource = "resume"
	RunSourceWorkspaceCreate RunSource = "workspace_create"
	RunSourceWorkspaceLoad   RunSource = "workspace_load"
	RunSourceWorkspaceRead   RunSource = "workspace_read"
)

const MainAgentGraphGoal = "grow graph through append_graph_batch"

func validNodeTypes() []NodeType {
	return []NodeType{
		NodeTopic,
		NodeQuestion,
		NodeTension,
		NodeHypothesis,
		NodeOpportunity,
		NodeIdea,
		NodeEvidence,
		NodeResearch,
		NodeClaim,
		NodeDecision,
		NodeUnknown,
		NodeDirection,
		NodeArtifact,
	}
}

func validEdgeTypes() []EdgeType {
	return []EdgeType{
		EdgeSupports,
		EdgeRefines,
		EdgeLeadsTo,
		EdgeExpands,
		EdgeContradicts,
		EdgeQuestions,
		EdgeExplains,
		EdgeWeakens,
		EdgeJustifies,
		EdgeBranchesFrom,
		EdgeRaises,
		EdgeResolves,
	}
}

func validNodeStatuses() []NodeStatus {
	return []NodeStatus{
		NodeActive,
		NodeArchived,
	}
}
