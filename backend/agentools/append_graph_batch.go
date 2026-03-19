package agentools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolAppendGraphBatch = "append_graph_batch"

type AppendGraphBatchNode struct {
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	Summary         string         `json:"summary"`
	Status          string         `json:"status,omitempty"`
	Score           float64        `json:"score,omitempty"`
	Depth           int            `json:"depth,omitempty"`
	ParentContext   string         `json:"parent_context,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	EvidenceSummary string         `json:"evidence_summary,omitempty"`
}

type AppendGraphBatchEdge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type AppendGraphBatchParams struct {
	WorkspaceID string                 `json:"workspace_id"`
	Nodes       []AppendGraphBatchNode `json:"nodes"`
	Edges       []AppendGraphBatchEdge `json:"edges"`
}

type AppendGraphBatchResult struct {
	WorkspaceID  string   `json:"workspace_id"`
	AppliedNodes int      `json:"applied_nodes"`
	AppliedEdges int      `json:"applied_edges"`
	MutationIDs  []string `json:"mutation_ids,omitempty"`
	Summary      string   `json:"summary"`
}

type GraphBatchAppender interface {
	AppendGraphBatch(ctx context.Context, req AppendGraphBatchParams) (AppendGraphBatchResult, error)
}

var appendGraphBatchToolInfo = &schema.ToolInfo{
	Name: ToolAppendGraphBatch,
	Desc: "Append nodes and edges to an exploration workspace graph without rewriting existing graph state.",
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"workspace_id": {
			Type:     schema.String,
			Desc:     "Workspace identifier that owns the graph.",
			Required: true,
		},
		"nodes": {
			Type: schema.Array,
			Desc: "Append-only node batch. Node ids must be unique in the workspace and in the batch.",
			ElemInfo: &schema.ParameterInfo{
				Type: schema.Object,
				SubParams: map[string]*schema.ParameterInfo{
					"id":               {Type: schema.String, Required: true},
					"type":             {Type: schema.String, Required: true},
					"title":            {Type: schema.String, Required: true},
					"summary":          {Type: schema.String, Required: true},
					"status":           {Type: schema.String},
					"score":            {Type: schema.Number},
					"depth":            {Type: schema.Integer},
					"parent_context":   {Type: schema.String},
					"metadata":         {Type: schema.Object},
					"evidence_summary": {Type: schema.String},
				},
			},
		},
		"edges": {
			Type: schema.Array,
			Desc: "Append-only edge batch. Edge endpoints must already exist or be introduced in the same batch.",
			ElemInfo: &schema.ParameterInfo{
				Type: schema.Object,
				SubParams: map[string]*schema.ParameterInfo{
					"id":   {Type: schema.String, Required: true},
					"from": {Type: schema.String, Required: true},
					"to":   {Type: schema.String, Required: true},
					"type": {Type: schema.String, Required: true},
				},
			},
		},
	}),
}

func NewAppendGraphBatchTool(appender GraphBatchAppender) tool.InvokableTool {
	return &appendGraphBatchTool{appender: appender}
}

type appendGraphBatchTool struct {
	appender GraphBatchAppender
}

func (t *appendGraphBatchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return appendGraphBatchToolInfo, nil
}

func (t *appendGraphBatchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	_ = opts

	req := AppendGraphBatchParams{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return "", err
	}

	result, err := t.appender.AppendGraphBatch(ctx, req)
	if err != nil {
		return "", err
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
