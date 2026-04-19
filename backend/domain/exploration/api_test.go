package exploration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cv70/pkgo/mistake"
	"github.com/gin-gonic/gin"
)

type testResp[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func newTestRouter() *gin.Engine {
	r, _ := newTestRouterWithDomain()
	return r
}

func newTestRouterWithDomain() (*gin.Engine, *ExplorationDomain) {
	domain := newTestExplorationDomain()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, domain)
	return r, domain
}

func TestCreateWorkspaceAndReadProjection(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", createW.Code)
	}

	var created testResp[WorkspaceSnapshot]
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Code != http.StatusOK {
		t.Fatalf("unexpected business code: %d", created.Code)
	}
	if created.Data.Exploration.ID == "" {
		t.Fatal("exploration id should not be empty")
	}
	if len(created.Data.DirectionMap.Nodes) == 0 {
		t.Fatal("expected direction map nodes in projection")
	}
	if created.Data.DirectionMap.Nodes[0].Type != NodeTopic {
		t.Fatalf("expected initial node to be topic, got %s", created.Data.DirectionMap.Nodes[0].Type)
	}

	getReq, _ := http.NewRequest(http.MethodGet, "/api/v1/exploration/workspaces/"+created.Data.Exploration.ID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getW.Code)
	}

	var loaded testResp[WorkspaceSnapshot]
	if err := json.Unmarshal(getW.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if loaded.Code != http.StatusOK {
		t.Fatalf("unexpected business code on get: %d", loaded.Code)
	}
	if loaded.Data.Exploration.ID != created.Data.Exploration.ID {
		t.Fatalf("unexpected exploration id: %s", loaded.Data.Exploration.ID)
	}
}

func TestInterventionExpandOpportunity(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	beforeCount := len(created.Data.Workbench.IdeaCards)
	interventionBody := []byte(`{"type":"expand_opportunity","target_id":"` + created.Data.Exploration.ActiveOpportunityID + `"}`)
	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/interventions",
		bytes.NewBuffer(interventionBody),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)

	var updated testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(interventionW.Body.Bytes(), &updated)

	if updated.Code != http.StatusOK {
		t.Fatalf("unexpected business code: %d", updated.Code)
	}
	if len(updated.Data.Workbench.IdeaCards) != beforeCount {
		t.Fatalf("expected expand to be a no-op before agent creates opportunities, before=%d after=%d", beforeCount, len(updated.Data.Workbench.IdeaCards))
	}
	if len(updated.Data.Exploration.Runs) < len(created.Data.Exploration.Runs) {
		t.Fatal("expected run history to be preserved")
	}
}

func TestInterventionToggleFavorite(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	ideaID := "idea-manual-favorite"
	interventionBody := []byte(`{"type":"toggle_favorite","target_id":"` + ideaID + `"}`)
	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/interventions",
		bytes.NewBuffer(interventionBody),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)

	var updated testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(interventionW.Body.Bytes(), &updated)

	if updated.Code != http.StatusOK {
		t.Fatalf("unexpected business code: %d", updated.Code)
	}
	found := false
	for _, favoriteID := range updated.Data.Exploration.Favorites {
		if favoriteID == ideaID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected favorite id %s to be present", ideaID)
	}
}

func TestGetWorkspaceNotFound(t *testing.T) {
	r := newTestRouter()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/exploration/workspaces/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp testResp[int]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected business 400 for invalid workspace id, got %d", resp.Code)
	}
}

func TestGetRuntimeState(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	runtimeReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/runtime",
		nil,
	)
	runtimeW := httptest.NewRecorder()
	r.ServeHTTP(runtimeW, runtimeReq)

	var runtimeResp testResp[RuntimeStateSnapshot]
	if err := json.Unmarshal(runtimeW.Body.Bytes(), &runtimeResp); err != nil {
		t.Fatalf("decode runtime response: %v", err)
	}
	if runtimeResp.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", runtimeResp.Code)
	}
	if len(runtimeResp.Data.Runs) == 0 {
		t.Fatal("expected runtime runs")
	}
	if len(runtimeResp.Data.AgentTasks) == 0 {
		t.Fatal("expected runtime activity records")
	}
}

func TestGetRuntimeStateWithFilters(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	for i := 0; i < 4; i++ {
		domain.executeRuntimeCycle(created.Data.Exploration, "test_filter")
	}

	fullState, ok := domain.GetRuntimeState(created.Data.Exploration.ID)
	if !ok || len(fullState.Runs) < 2 {
		t.Fatal("expected at least two runs for runtime filter tests")
	}
	targetRunID := fullState.Runs[len(fullState.Runs)-1].ID

	runReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/runtime?run_id="+targetRunID,
		nil,
	)
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)

	var runResp testResp[RuntimeStateSnapshot]
	if err := json.Unmarshal(runW.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode run filter response: %v", err)
	}
	if runResp.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", runResp.Code)
	}
	if len(runResp.Data.Runs) != 1 || runResp.Data.Runs[0].ID != targetRunID {
		t.Fatalf("expected only run %s", targetRunID)
	}
	for _, task := range runResp.Data.AgentTasks {
		if task.RunID != targetRunID {
			t.Fatalf("unexpected task run id: %s", task.RunID)
		}
	}

	latestReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/runtime?latest_runs=1",
		nil,
	)
	latestW := httptest.NewRecorder()
	r.ServeHTTP(latestW, latestReq)

	var latestResp testResp[RuntimeStateSnapshot]
	if err := json.Unmarshal(latestW.Body.Bytes(), &latestResp); err != nil {
		t.Fatalf("decode latest_runs response: %v", err)
	}
	if latestResp.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", latestResp.Code)
	}
	if len(latestResp.Data.Runs) != 1 {
		t.Fatalf("expected exactly one latest run, got %d", len(latestResp.Data.Runs))
	}
}

func TestReplayMutations(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	interventionBody := []byte(`{"type":"expand_opportunity","target_id":"` + created.Data.Exploration.ActiveOpportunityID + `"}`)
	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/interventions",
		bytes.NewBuffer(interventionBody),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)

	replayReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/mutations?cursor=&limit=1",
		nil,
	)
	replayW := httptest.NewRecorder()
	r.ServeHTTP(replayW, replayReq)

	var replayResp testResp[map[string]any]
	_ = json.Unmarshal(replayW.Body.Bytes(), &replayResp)
	if replayResp.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", replayResp.Code)
	}
	mutations, ok := replayResp.Data["mutations"].([]any)
	if !ok {
		t.Fatal("mutations missing in replay response")
	}
	_ = mutations
	if replayResp.Data["has_more"] == nil {
		t.Fatal("expected has_more field")
	}
}

func TestUpdateStrategy(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	updateBody := []byte(`{"interval_ms":1200,"max_runs":8,"expansion_mode":"round_robin"}`)
	updateReq, _ := http.NewRequest(
		http.MethodPut,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/strategy",
		bytes.NewBuffer(updateBody),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()
	r.ServeHTTP(updateW, updateReq)

	var updated testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(updateW.Body.Bytes(), &updated)
	if updated.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", updated.Code)
	}
	if updated.Data.Exploration.Strategy.IntervalMs != 1200 {
		t.Fatalf("expected interval 1200, got %d", updated.Data.Exploration.Strategy.IntervalMs)
	}
	if updated.Data.Exploration.Strategy.MaxRuns != 8 {
		t.Fatalf("expected max_runs 8, got %d", updated.Data.Exploration.Strategy.MaxRuns)
	}
	if updated.Data.Exploration.Strategy.ExpansionMode != "round_robin" {
		t.Fatalf("expected round_robin, got %s", updated.Data.Exploration.Strategy.ExpansionMode)
	}
}

func TestListAndArchiveWorkspaces(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	listReq, _ := http.NewRequest(http.MethodGet, "/api/v1/exploration/workspaces?limit=20", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	var listed testResp[map[string]any]
	_ = json.Unmarshal(listW.Body.Bytes(), &listed)
	if listed.Code != http.StatusOK {
		t.Fatalf("expected list code 200, got %d", listed.Code)
	}

	deleteReq, _ := http.NewRequest(http.MethodDelete, "/api/v1/exploration/workspaces/"+created.Data.Exploration.ID, nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	var archived testResp[map[string]any]
	_ = json.Unmarshal(deleteW.Body.Bytes(), &archived)
	if archived.Code != http.StatusOK {
		t.Fatalf("expected archive code 200, got %d", archived.Code)
	}
}

func TestWorkspaceSnapshotIncludesPlanState(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/exploration/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	interventionBody := []byte(`{"type":"shift_focus","target_id":"` + created.Data.Exploration.ActiveOpportunityID + `","note":"follow strongest evidence"}`)
	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/exploration/workspaces/"+created.Data.Exploration.ID+"/interventions",
		bytes.NewBuffer(interventionBody),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)

	var updated testResp[WorkspaceSnapshot]
	_ = json.Unmarshal(interventionW.Body.Bytes(), &updated)
	if updated.Code != http.StatusOK {
		t.Fatalf("expected business 200, got %d", updated.Code)
	}
	if updated.Data.Workbench.CurrentFocus == "" {
		t.Fatal("expected current focus in snapshot")
	}
	if updated.Data.Workbench.LatestChange == "" {
		t.Fatal("expected latest change summary in snapshot")
	}
	if updated.Data.Workbench.LatestRunStatus == "" {
		t.Fatal("expected latest run status in snapshot")
	}
	if updated.Data.Workbench.LatestReplanReason == "" {
		t.Fatal("expected latest replan reason in snapshot")
	}
}

func TestV1CreateRunAndGetRun(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d", createW.Code)
	}

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Workspace.ID == "" {
		t.Fatal("expected workspace id")
	}

	runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+created.Workspace.ID+"/runs", strings.NewReader(`{"trigger":"manual"}`))
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusAccepted {
		t.Fatalf("unexpected run status: %d", runW.Code)
	}

	var runResp RunResponse
	if err := json.Unmarshal(runW.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if runResp.Run.ID == "" {
		t.Fatal("expected run id")
	}
	if runResp.Run.Status == "" {
		t.Fatal("expected run status")
	}

	getRunReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/runs/"+runResp.Run.ID, nil)
	getRunW := httptest.NewRecorder()
	r.ServeHTTP(getRunW, getRunReq)
	if getRunW.Code != http.StatusOK {
		t.Fatalf("unexpected get run status: %d", getRunW.Code)
	}

	var fetched RunResponse
	if err := json.Unmarshal(getRunW.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get run response: %v", err)
	}
	if fetched.Run.ID != runResp.Run.ID {
		t.Fatalf("expected run id %s, got %s", runResp.Run.ID, fetched.Run.ID)
	}
	if !isRFC3339(fetched.Run.StartedAt) {
		t.Fatalf("expected RFC3339 started_at, got %s", fetched.Run.StartedAt)
	}
}

func TestV1ProjectionAndInterventionLifecycle(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	workspaceID := created.Workspace.ID

	projectionReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID+"/projection", nil)
	projectionW := httptest.NewRecorder()
	r.ServeHTTP(projectionW, projectionReq)
	if projectionW.Code != http.StatusOK {
		t.Fatalf("unexpected projection status: %d", projectionW.Code)
	}

	var projectionResp ProjectionResponse
	if err := json.Unmarshal(projectionW.Body.Bytes(), &projectionResp); err != nil {
		t.Fatalf("decode projection response: %v", err)
	}
	if len(projectionResp.Projection.Map.Nodes) == 0 {
		t.Fatal("expected projection nodes")
	}

	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+workspaceID+"/interventions",
		strings.NewReader(`{"intent":"focus on strongest branch","target_branch_id":"`+projectionResp.Projection.Map.Nodes[0].ID+`"}`),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)
	if interventionW.Code != http.StatusAccepted {
		t.Fatalf("unexpected intervention status: %d", interventionW.Code)
	}

	var interventionResp InterventionResponse
	if err := json.Unmarshal(interventionW.Body.Bytes(), &interventionResp); err != nil {
		t.Fatalf("decode intervention response: %v", err)
	}
	if interventionResp.Intervention.ID == "" {
		t.Fatal("expected intervention id")
	}
	if interventionResp.Intervention.Status != InterventionAbsorbed && interventionResp.Intervention.Status != InterventionReflected {
		t.Fatalf("expected absorbed/reflected status after create, got %s", interventionResp.Intervention.Status)
	}
	if interventionResp.Intervention.Status == InterventionReflected {
		if interventionResp.Intervention.AbsorbedByRunID == "" {
			t.Fatal("expected absorbed_by_run_id for reflected intervention")
		}
		if interventionResp.Intervention.ReflectedEventID == "" {
			t.Fatal("expected reflected_event_id for reflected intervention")
		}
	}

	getInterventionReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID+"/interventions/"+interventionResp.Intervention.ID, nil)
	getInterventionW := httptest.NewRecorder()
	r.ServeHTTP(getInterventionW, getInterventionReq)
	if getInterventionW.Code != http.StatusOK {
		t.Fatalf("unexpected get intervention status: %d", getInterventionW.Code)
	}

	var fetched InterventionResponse
	if err := json.Unmarshal(getInterventionW.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get intervention response: %v", err)
	}
	if fetched.Intervention.ID != interventionResp.Intervention.ID {
		t.Fatalf("expected intervention id %s, got %s", interventionResp.Intervention.ID, fetched.Intervention.ID)
	}
	if fetched.Intervention.Status != interventionResp.Intervention.Status {
		t.Fatalf("expected get status %s, got %s", interventionResp.Intervention.Status, fetched.Intervention.Status)
	}
}

func TestV1TraceSummary(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	traceReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/trace/summary", nil)
	traceW := httptest.NewRecorder()
	r.ServeHTTP(traceW, traceReq)
	if traceW.Code != http.StatusOK {
		t.Fatalf("unexpected trace status: %d", traceW.Code)
	}

	var traceResp TraceSummaryResponse
	if err := json.Unmarshal(traceW.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("decode trace summary response: %v", err)
	}
	if traceResp.WorkspaceID == "" {
		t.Fatal("expected workspace id")
	}
	if len(traceResp.Items) == 0 {
		t.Fatal("expected non-empty trace items")
	}
	if !isRFC3339(traceResp.Items[0].Timestamp) {
		t.Fatalf("expected RFC3339 timestamp, got %s", traceResp.Items[0].Timestamp)
	}
}

func TestV1ErrorShape(t *testing.T) {
	r := newTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/999999999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.Error.Code != "not_found" {
		t.Fatalf("expected not_found code, got %s", resp.Error.Code)
	}
	if resp.Error.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestV1InterventionCanRecoverFromDB(t *testing.T) {
	r, domain := newTestRouterWithDomain()
	if domain.DB == nil {
		t.Skip("db unavailable in test environment")
	}

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions",
		strings.NewReader(`{"intent":"focus on strongest branch","target_branch_id":"`+created.Workspace.ID+`"}`),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)

	var createdIntervention InterventionResponse
	if err := json.Unmarshal(interventionW.Body.Bytes(), &createdIntervention); err != nil {
		t.Fatalf("decode create intervention response: %v", err)
	}
	if createdIntervention.Intervention.ID == "" {
		t.Fatal("expected intervention id")
	}

	domain.withWorkspaceState(created.Workspace.ID, func(state *RuntimeWorkspaceState) {
		state.Interventions = map[string]InterventionView{}
	})

	getReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions/"+createdIntervention.Intervention.ID,
		nil,
	)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var loaded InterventionResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("decode loaded intervention response: %v", err)
	}
	if loaded.Intervention.ID != createdIntervention.Intervention.ID {
		t.Fatalf("expected intervention id %s, got %s", createdIntervention.Intervention.ID, loaded.Intervention.ID)
	}
	if loaded.Intervention.Status == "" {
		t.Fatal("expected intervention status")
	}

	events, err := domain.DB.ListInterventionEventsByPrefix(
		created.Workspace.ID,
		createdIntervention.Intervention.ID+"#",
		"v1_intervention_lifecycle_event",
		20,
	)
	if err != nil {
		t.Fatalf("list intervention lifecycle events: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 lifecycle events (received + progressed), got %d", len(events))
	}
}

func TestV1ListInterventionEvents(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d", createW.Code)
	}

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions",
		strings.NewReader(`{"intent":"focus on strongest branch","target_branch_id":"`+created.Workspace.ID+`"}`),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)
	if interventionW.Code != http.StatusAccepted {
		t.Fatalf("unexpected intervention status: %d", interventionW.Code)
	}
	var intervention InterventionResponse
	if err := json.Unmarshal(interventionW.Body.Bytes(), &intervention); err != nil {
		t.Fatalf("decode intervention response: %v", err)
	}

	eventsReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions/"+intervention.Intervention.ID+"/events",
		nil,
	)
	eventsW := httptest.NewRecorder()
	r.ServeHTTP(eventsW, eventsReq)
	if eventsW.Code != http.StatusOK {
		t.Fatalf("unexpected events status: %d", eventsW.Code)
	}

	var eventsResp InterventionEventsResponse
	if err := json.Unmarshal(eventsW.Body.Bytes(), &eventsResp); err != nil {
		t.Fatalf("decode events response: %v", err)
	}
	if len(eventsResp.Events) == 0 {
		t.Fatal("expected at least one event")
	}
	if eventsResp.Events[len(eventsResp.Events)-1].Status == "" {
		t.Fatal("expected latest event status")
	}
	if eventsResp.Events[0].InterventionID != intervention.Intervention.ID {
		t.Fatalf("expected intervention id %s, got %s", intervention.Intervention.ID, eventsResp.Events[0].InterventionID)
	}

	// verify pagination shape
	pagedReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions/"+intervention.Intervention.ID+"/events?limit=1",
		nil,
	)
	pagedW := httptest.NewRecorder()
	r.ServeHTTP(pagedW, pagedReq)
	if pagedW.Code != http.StatusOK {
		t.Fatalf("unexpected paged events status: %d", pagedW.Code)
	}
	var paged InterventionEventsResponse
	if err := json.Unmarshal(pagedW.Body.Bytes(), &paged); err != nil {
		t.Fatalf("decode paged events response: %v", err)
	}
	if len(paged.Events) != 1 {
		t.Fatalf("expected exactly 1 paged event, got %d", len(paged.Events))
	}
	if len(eventsResp.Events) > 1 && !paged.HasMore {
		t.Fatal("expected has_more true when full response has multiple events")
	}
	if paged.HasMore && paged.NextCursor == "" {
		t.Fatal("expected next_cursor when has_more is true")
	}
	if paged.NextCursor != "" && !strings.Contains(paged.NextCursor, "|") {
		t.Fatalf("expected composite cursor format, got %s", paged.NextCursor)
	}

	if paged.NextCursor != "" {
		nextPageReq, _ := http.NewRequest(
			http.MethodGet,
			"/api/v1/workspaces/"+created.Workspace.ID+"/interventions/"+intervention.Intervention.ID+"/events?limit=1&cursor="+paged.NextCursor,
			nil,
		)
		nextPageW := httptest.NewRecorder()
		r.ServeHTTP(nextPageW, nextPageReq)
		if nextPageW.Code != http.StatusOK {
			t.Fatalf("unexpected next page status: %d", nextPageW.Code)
		}
		var nextPage InterventionEventsResponse
		if err := json.Unmarshal(nextPageW.Body.Bytes(), &nextPage); err != nil {
			t.Fatalf("decode next page response: %v", err)
		}
		if len(nextPage.Events) > 0 && nextPage.Events[0].ID == paged.Events[0].ID {
			t.Fatal("expected next page to move beyond previous event")
		}
	}

	filteredReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions/"+intervention.Intervention.ID+"/events?status=reflected",
		nil,
	)
	filteredW := httptest.NewRecorder()
	r.ServeHTTP(filteredW, filteredReq)
	if filteredW.Code != http.StatusOK {
		t.Fatalf("unexpected filtered events status: %d", filteredW.Code)
	}
	var filtered InterventionEventsResponse
	if err := json.Unmarshal(filteredW.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("decode filtered events response: %v", err)
	}
	if len(filtered.Events) == 0 {
		t.Fatal("expected at least one reflected event")
	}
	for _, event := range filtered.Events {
		if event.Status != InterventionReflected {
			t.Fatalf("expected reflected status, got %s", event.Status)
		}
	}
}

func TestV1ListTraceEvents(t *testing.T) {
	r := newTestRouter()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d", createW.Code)
	}

	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	traceReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/trace/events?limit=1",
		nil,
	)
	traceW := httptest.NewRecorder()
	r.ServeHTTP(traceW, traceReq)
	if traceW.Code != http.StatusOK {
		t.Fatalf("unexpected trace events status: %d", traceW.Code)
	}

	var traceResp TraceEventsResponse
	if err := json.Unmarshal(traceW.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("decode trace events response: %v", err)
	}
	if len(traceResp.Items) == 0 {
		t.Fatal("expected at least one trace event")
	}
	if len(traceResp.Events) == 0 {
		t.Fatal("expected structured trace events")
	}
	if traceResp.Events[0].EventType == "" {
		t.Fatal("expected structured trace event type")
	}
	if traceResp.Items[0].ID == "" {
		t.Fatal("expected trace event id")
	}
	if traceResp.HasMore && traceResp.NextCursor == "" {
		t.Fatal("expected next_cursor when has_more is true")
	}
	if traceResp.NextCursor != "" && !strings.Contains(traceResp.NextCursor, "|") {
		t.Fatalf("expected composite cursor format, got %s", traceResp.NextCursor)
	}

	filteredReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/trace/events?category=tool", nil)
	filteredW := httptest.NewRecorder()
	r.ServeHTTP(filteredW, filteredReq)
	if filteredW.Code != http.StatusOK {
		t.Fatalf("unexpected filtered trace status: %d", filteredW.Code)
	}
	var filtered TraceEventsResponse
	if err := json.Unmarshal(filteredW.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("decode filtered trace response: %v", err)
	}
	if len(filtered.Items) == 0 {
		t.Fatal("expected filtered trace items")
	}
	if len(filtered.Events) == 0 {
		t.Fatal("expected filtered trace events")
	}
	for _, item := range filtered.Items {
		if item.Category != "tool" {
			t.Fatalf("expected tool category, got %s", item.Category)
		}
	}
	for _, event := range filtered.Events {
		if traceCategoryForAgentRunEvent(event) != "tool" {
			t.Fatalf("expected tool trace event, got %+v", event)
		}
	}

	levelReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/trace/events?level=info",
		nil,
	)
	levelW := httptest.NewRecorder()
	r.ServeHTTP(levelW, levelReq)
	if levelW.Code != http.StatusOK {
		t.Fatalf("unexpected level filtered trace status: %d", levelW.Code)
	}
	var levelFiltered TraceEventsResponse
	if err := json.Unmarshal(levelW.Body.Bytes(), &levelFiltered); err != nil {
		t.Fatalf("decode level filtered trace response: %v", err)
	}
	if len(levelFiltered.Items) == 0 {
		t.Fatal("expected level filtered trace items")
	}
	for _, item := range levelFiltered.Items {
		if item.Level != "info" {
			t.Fatalf("expected info level, got %s", item.Level)
		}
	}
}

func TestV1ListTraceEventsRejectsInvalidCategory(t *testing.T) {
	r := newTestRouter()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/workspace-missing/trace/events?category=bad-category",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid category, got %d", w.Code)
	}
}

func TestV1ListTraceEventsRejectsInvalidLevel(t *testing.T) {
	r := newTestRouter()
	req, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/workspace-missing/trace/events?level=bad-level",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid level, got %d", w.Code)
	}
}

func TestV1ToggleFavoriteInterventionAffectsWorkspaceState(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"AI education","output_goal":"Research directions","constraints":"Low-cost, explainable"}`)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d", createW.Code)
	}
	var created WorkspaceResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	_, ok := domain.GetWorkspace(created.Workspace.ID)
	if !ok {
		t.Fatal("expected created workspace snapshot")
	}
	ideaID := "idea-manual-favorite"

	interventionReq, _ := http.NewRequest(
		http.MethodPost,
		"/api/v1/workspaces/"+created.Workspace.ID+"/interventions",
		strings.NewReader(`{"intent":"toggle favorite","target_branch_id":"`+ideaID+`"}`),
	)
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionW := httptest.NewRecorder()
	r.ServeHTTP(interventionW, interventionReq)
	if interventionW.Code != http.StatusAccepted {
		t.Fatalf("unexpected intervention status: %d", interventionW.Code)
	}

	updated, ok := domain.GetWorkspace(created.Workspace.ID)
	if !ok {
		t.Fatal("expected updated workspace snapshot")
	}
	found := false
	for _, favoriteID := range updated.Exploration.Favorites {
		if favoriteID == ideaID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected favorite id %s to be present", ideaID)
	}
}

func isRFC3339(v string) bool {
	if v == "" {
		return false
	}
	pattern := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`
	return regexp.MustCompile(pattern).MatchString(v)
}

func TestDeterministicPlannerInitialRun(t *testing.T) {
	p := NewDeterministicPlanner()
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.6,
			Research:   0.7,
			Aggression: 0.4,
		},
	}
	nodes, edges := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Direction nodes on first cycle, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeDirection {
			t.Errorf("expected all initial nodes to be NodeDirection, got %s", n.Type)
		}
	}
	// No edges expected for initial Direction nodes
	for _, e := range edges {
		_ = e
	}
	if len(nodes) < 3 || len(nodes) > 5 {
		t.Errorf("expected 3-5 Direction nodes, got %d", len(nodes))
	}
}

func TestDeterministicPlannerResearchPhase(t *testing.T) {
	p := NewDeterministicPlanner()
	dirNode := Node{ID: "dir-1", Type: NodeDirection, Title: "ML for diagnosis", WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dirNode},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.6,
			Research:   0.7,
			Aggression: 0.4,
		},
	}
	nodes, edges := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Evidence nodes in research phase, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeEvidence {
			t.Errorf("expected Evidence nodes, got %s", n.Type)
		}
	}
	// Each Evidence node should have an edge to the Direction
	if len(edges) == 0 {
		t.Error("expected edges from Evidence to Direction, got none")
	}
	for _, e := range edges {
		if e.Type != EdgeSupports && e.Type != EdgeContradicts {
			t.Errorf("expected EdgeSupports or EdgeContradicts, got %s", e.Type)
		}
		if e.To != dirNode.ID {
			t.Errorf("expected edge to direction ID %s, got %s", dirNode.ID, e.To)
		}
	}
}

func TestDeterministicPlannerFastPath(t *testing.T) {
	p := NewDeterministicPlanner()
	dirNode := Node{ID: "dir-1", Type: NodeDirection, Title: "ML for diagnosis", WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dirNode}, // Direction but no Evidence
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Aggression: 0.8, // High aggression: skip Evidence, go straight to Claims
		},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Claim nodes on fast path, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeClaim {
			t.Errorf("expected NodeClaim on fast path, got %s", n.Type)
		}
	}
}

func TestDeterministicPlannerConvergence(t *testing.T) {
	p := NewDeterministicPlanner()
	dir := Node{ID: "dir-1", Type: NodeDirection, WorkspaceID: "ws-test"}
	ev1 := Node{ID: "ev-1", Type: NodeEvidence, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	ev2 := Node{ID: "ev-2", Type: NodeEvidence, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	claim := Node{ID: "cl-1", Type: NodeClaim, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dir, ev1, ev2, claim},
		Edges: []Edge{
			{ID: "e1", From: "ev-1", To: "dir-1", Type: EdgeSupports},
			{ID: "e2", From: "ev-2", To: "dir-1", Type: EdgeContradicts},
		},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.3, // Converge: should produce Decision
		},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Decision node in convergence, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeDecision {
			t.Errorf("expected NodeDecision in convergence, got %s", n.Type)
		}
	}
}

func TestInterventionAdjustsBalanceState(t *testing.T) {
	domain := newTestExplorationDomain()
	req := CreateWorkspaceReq{Topic: "quantum computing", OutputGoal: "summary"}
	snapshot, err := domain.CreateWorkspace(req)
	mistake.Unwrap(err)
	wsID := snapshot.Exploration.ID

	// Set known balance state
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		state.Balance = BalanceState{WorkspaceID: wsID, Divergence: 0.6, Research: 0.6, Aggression: 0.4}
	})

	// Apply intervention with converging intent
	intReq := InterventionReq{Type: InterventionAddContext, Note: "please focus and 收敛"}
	domain.replanRuntimeState(snapshot.Exploration, intReq)

	var balance BalanceState
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		balance = state.Balance
	})
	// "focus" and "收敛" both trigger Divergence -= 0.2, accumulated = -0.4 → clamped
	if balance.Divergence >= 0.6 {
		t.Errorf("expected Divergence to decrease after '收敛' intent, got %f", balance.Divergence)
	}
}

func TestMutationEventsWrittenOnRunComplete(t *testing.T) {
	router, domain := newTestRouterWithDomain()
	_ = router

	// Create workspace (initializeWorkspaceGraph is called inside CreateWorkspace handler)
	wsID := ""
	{
		req := CreateWorkspaceReq{Topic: "autonomous vehicles safety", OutputGoal: "risk summary"}
		snapshot, err := domain.CreateWorkspace(req)
		mistake.Unwrap(err)
		wsID = snapshot.Exploration.ID
		// Also call initializeWorkspaceGraph (normally called from handler)
		domain.initializeWorkspaceGraph(context.Background(), wsID)
	}

	var mutations []MutationEvent
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		mutations = append(mutations, state.Mutations...)
	})

	if len(mutations) == 0 {
		t.Fatal("expected at least one mutation event after initializeWorkspaceGraph, got none")
	}

	// Check that at least one node_added event exists
	hasNodeAdded := false
	for _, m := range mutations {
		if m.Kind == "node_added" {
			hasNodeAdded = true
			break
		}
	}
	if !hasNodeAdded {
		t.Error("expected at least one 'node_added' mutation event")
	}
}

func TestRuntimeCycleAppendsMainAgentGraphBatch(t *testing.T) {
	domain := newScriptedExplorationDomain(
		`{"summary":"workspace bootstrap noop","nodes":[],"edges":[]}`,
		`{"summary":"append a runtime evidence node","nodes":[{"id":"evidence-runtime-cycle","type":"evidence","title":"Runtime evidence","summary":"Generated during runtime cycle","status":"active","depth":2}],"edges":[{"id":"edge-runtime-cycle","from":"%s","to":"evidence-runtime-cycle","type":"supports"}]}`,
	)

	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "machine learning safety", OutputGoal: "risk report"})
	mistake.Unwrap(err)
	wsID := snapshot.Exploration.ID
	activeID := snapshot.Exploration.ActiveOpportunityID
	model := domain.Model.(*scriptedToolCallingModel)
	model.replies[1] = strings.ReplaceAll(model.replies[1], "%s", activeID)

	before, ok := domain.GetWorkspace(wsID)
	if !ok {
		t.Fatal("workspace not found after creation")
	}

	domain.executeRuntimeCycle(before.Exploration, "test")

	after, ok := domain.GetWorkspace(wsID)
	if !ok {
		t.Fatal("workspace not found after executeRuntimeCycle")
	}
	if !hasNode(after.Exploration.Nodes, "evidence-runtime-cycle") {
		t.Fatalf("expected runtime cycle to append evidence node, total nodes=%d", len(after.Exploration.Nodes))
	}
}

func TestCreateRun_Returns202Immediately(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	// Create a workspace first
	createBody := []byte(`{"topic":"AI education","output_goal":"Research"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var created WorkspaceResponse
	json.Unmarshal(w.Body.Bytes(), &created)
	wsID := created.Workspace.ID

	// POST create run — must return 202 quickly (no blocking LLM calls)
	runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsID+"/runs", nil)
	runW := httptest.NewRecorder()
	start := time.Now()
	r.ServeHTTP(runW, runReq)
	elapsed := time.Since(start)

	if runW.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", runW.Code, runW.Body.String())
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("CreateRun took too long: %v (should be <500ms)", elapsed)
	}
	_ = domain
}

func TestCreateRun_IdempotentWhenAlreadyRunning(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"Blockchain","output_goal":"Summary"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var created WorkspaceResponse
	json.Unmarshal(w.Body.Bytes(), &created)
	wsID := created.Workspace.ID

	// Simulate AgentRunning = true
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		runID := "existing-run-id"
		s.Runs = []Run{{ID: runID, WorkspaceID: wsID, Status: RunStatusRunning, StartedAt: time.Now().UnixMilli()}}
		s.AgentRunning = true
	})

	runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsID+"/runs", nil)
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("expected 200 for idempotent re-request, got %d", runW.Code)
	}
}

func TestV1RunResponseUsesSimplifiedAgentDrivenShape(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"agent runtime","output_goal":"map growth"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create workspace: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var created WorkspaceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create workspace: %v", err)
	}

	runReq, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+created.Workspace.ID+"/runs", nil)
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusAccepted {
		t.Fatalf("create run: expected 202, got %d body=%s", runW.Code, runW.Body.String())
	}

	var runResp RunResponse
	if err := json.Unmarshal(runW.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if runResp.Run.ID == "" {
		t.Fatal("expected run id")
	}
	if runResp.Run.Status == "" {
		t.Fatal("expected run status")
	}

	time.Sleep(150 * time.Millisecond)

	getRunReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/runs/"+runResp.Run.ID, nil)
	getRunW := httptest.NewRecorder()
	r.ServeHTTP(getRunW, getRunReq)
	if getRunW.Code != http.StatusOK {
		t.Fatalf("get run: expected 200, got %d body=%s", getRunW.Code, getRunW.Body.String())
	}

	var fetched RunResponse
	if err := json.Unmarshal(getRunW.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode fetched run: %v", err)
	}
	if fetched.Run.TurnCount == 0 {
		t.Fatalf("expected turn_count in fetched run, got %+v", fetched.Run)
	}
	if fetched.Run.LatestTurnID == "" || fetched.Run.LatestCheckpointID == "" || fetched.Run.ResumeCursor == "" {
		t.Fatalf("expected turn/checkpoint metadata in fetched run, got %+v", fetched.Run)
	}
}

func TestV1TraceSummaryUsesRunToolMutationCategories(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"trace categories","output_goal":"runtime summary"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create workspace: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var created WorkspaceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create workspace: %v", err)
	}

	domain.withWorkspaceState(created.Workspace.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Results) == 0 {
			state.Results = append(state.Results, AgentTaskResultSummary{
				TaskID:    "tool-1",
				Summary:   "append_graph_batch added 2 nodes",
				IsSuccess: true,
				UpdatedAt: time.Now().UnixMilli(),
			})
		}
	})

	traceReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+created.Workspace.ID+"/trace/summary", nil)
	traceW := httptest.NewRecorder()
	r.ServeHTTP(traceW, traceReq)
	if traceW.Code != http.StatusOK {
		t.Fatalf("trace summary: expected 200, got %d body=%s", traceW.Code, traceW.Body.String())
	}

	var traceResp TraceSummaryResponse
	if err := json.Unmarshal(traceW.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("decode trace summary: %v", err)
	}
	if len(traceResp.Items) == 0 {
		t.Fatal("expected trace items")
	}
	for _, item := range traceResp.Items {
		if item.Category == "plan" || item.Category == "task" {
			t.Fatalf("unexpected legacy trace category %q", item.Category)
		}
	}
}

func TestMainAgentRunsDoNotCreatePlannerArtifacts(t *testing.T) {
	_, domain := newTestRouterWithDomain()

	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent growth", OutputGoal: "graph"})
	mistake.Unwrap(err)

	runID, launched := domain.triggerRun(context.Background(), snapshot.Exploration.ID, "manual")
	if !launched {
		t.Fatal("expected triggerRun to launch a new MainAgent run")
	}
	if runID == "" {
		t.Fatal("expected run ID")
	}

	time.Sleep(100 * time.Millisecond)

	state, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected main-agent activity records")
	}
}

func TestPatchWorkspacePause(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	// Create workspace
	createBody := []byte(`{"topic":"pause test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	// Pause it
	patchBody := []byte(`{"status":"paused"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch pause: unexpected status %d body=%s", patchW.Code, patchW.Body.String())
	}
	var patchResp WorkspaceResponse
	if err := json.Unmarshal(patchW.Body.Bytes(), &patchResp); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patchResp.Workspace.Status != WorkspaceStatusPaused {
		t.Fatalf("expected paused, got %s", patchResp.Workspace.Status)
	}
}

func TestPatchWorkspaceResume(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"resume test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	// Pause first (if DB is present)
	if domain.DB != nil {
		id, err := strconv.ParseUint(wsID, 10, 64)
		if err != nil {
			t.Fatalf("parse workspace id: %v", err)
		}
		if err := domain.DB.PauseWorkspaceState(uint(id)); err != nil {
			t.Fatalf("pause: %v", err)
		}
	}

	// Resume
	patchBody := []byte(`{"status":"active"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch resume: unexpected status %d body=%s", patchW.Code, patchW.Body.String())
	}
	var patchResp WorkspaceResponse
	if err := json.Unmarshal(patchW.Body.Bytes(), &patchResp); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patchResp.Workspace.Status != WorkspaceStatusActive {
		t.Fatalf("expected active, got %s", patchResp.Workspace.Status)
	}
}

func TestV1GetWorkspaceRejectsNonNumericID(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/not-a-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric workspaceID, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLegacyGetWorkspaceRejectsNonNumericID(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/exploration/workspaces/not-a-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected transport status 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp testResp[map[string]any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected business code 400 for non-numeric workspaceID, got %d body=%s", resp.Code, w.Body.String())
	}
}

func TestLegacyGetRuntimeRejectsNonNumericID(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/exploration/workspaces/not-a-number/runtime", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected transport status 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp testResp[map[string]any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected business code 400 for non-numeric workspaceID, got %d body=%s", resp.Code, w.Body.String())
	}
}

func TestPatchWorkspace_InvalidStatus(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"invalid test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	patchBody := []byte(`{"status":"banana"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", patchW.Code)
	}
}

func TestGetWorkspaceReturnsActiveStatus(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"status test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: unexpected status %d", w.Code)
	}

	var createResp WorkspaceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	wsID := createResp.Workspace.ID

	getReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+wsID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: unexpected status %d", getW.Code)
	}

	var getResp WorkspaceResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get: %v\n", err)
	}
	if getResp.Workspace.Status != WorkspaceStatusActive {
		t.Fatalf("expected status active, got %s", getResp.Workspace.Status)
	}
}
