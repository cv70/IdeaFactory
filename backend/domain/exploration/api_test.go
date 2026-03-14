package exploration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

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
	if len(created.Data.Workbench.Opportunities) == 0 {
		t.Fatal("expected opportunities in projection")
	}
	if len(created.Data.DirectionMap.Nodes) == 0 {
		t.Fatal("expected direction map nodes in projection")
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
	if len(updated.Data.Workbench.IdeaCards) <= beforeCount {
		t.Fatalf("expected expanded idea cards > %d, got %d", beforeCount, len(updated.Data.Workbench.IdeaCards))
	}
	if len(updated.Data.Exploration.Runs) <= len(created.Data.Exploration.Runs) {
		t.Fatal("expected runtime run history to grow")
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

	ideaID := created.Data.Workbench.IdeaCards[0].ID
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
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected business 404, got %d", resp.Code)
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
	if len(runtimeResp.Data.Plans) == 0 {
		t.Fatal("expected runtime plans")
	}
	if len(runtimeResp.Data.AgentTasks) == 0 {
		t.Fatal("expected runtime agent tasks")
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
	for _, plan := range runResp.Data.Plans {
		if plan.RunID != targetRunID {
			t.Fatalf("unexpected plan run id: %s", plan.RunID)
		}
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
	if len(mutations) == 0 {
		t.Fatal("expected replay mutations to be non-empty")
	}
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
	if runResp.Run.CurrentPlan == nil {
		t.Fatal("expected current plan")
	}
	if runResp.Run.CurrentPlan.Status == "" {
		t.Fatal("expected current plan status")
	}
	if len(runResp.Run.CurrentPlan.Steps) == 0 {
		t.Fatal("expected current plan steps")
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
	if interventionResp.Intervention.Status != InterventionReflected {
		t.Fatalf("expected reflected status on create after runtime event processing, got %s", interventionResp.Intervention.Status)
	}
	if interventionResp.Intervention.AbsorbedByRunID == "" {
		t.Fatal("expected absorbed_by_run_id")
	}
	if interventionResp.Intervention.ReplannedPlanID == "" {
		t.Fatal("expected replanned_plan_id")
	}
	if interventionResp.Intervention.ReflectedEventID == "" {
		t.Fatal("expected reflected_event_id")
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

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/non-exist", nil)
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

	domain.runtime.mu.Lock()
	delete(domain.runtime.intervention, created.Workspace.ID)
	domain.runtime.mu.Unlock()

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
	if traceResp.Items[0].ID == "" {
		t.Fatal("expected trace event id")
	}
	if traceResp.HasMore && traceResp.NextCursor == "" {
		t.Fatal("expected next_cursor when has_more is true")
	}
	if traceResp.NextCursor != "" && !strings.Contains(traceResp.NextCursor, "|") {
		t.Fatalf("expected composite cursor format, got %s", traceResp.NextCursor)
	}

	filteredReq, _ := http.NewRequest(
		http.MethodGet,
		"/api/v1/workspaces/"+created.Workspace.ID+"/trace/events?category=task",
		nil,
	)
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
	for _, item := range filtered.Items {
		if item.Category != "task" {
			t.Fatalf("expected task category, got %s", item.Category)
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

	snapshot, ok := domain.GetWorkspace(created.Workspace.ID)
	if !ok {
		t.Fatal("expected created workspace snapshot")
	}
	ideaID := ""
	for _, node := range snapshot.Exploration.Nodes {
		if node.Type == NodeIdea {
			ideaID = node.ID
			break
		}
	}
	if ideaID == "" {
		t.Fatal("expected at least one idea node")
	}

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
