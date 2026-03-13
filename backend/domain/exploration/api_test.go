package exploration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type testResp[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, NewExplorationDomain(nil))
	return r
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
	if len(created.Data.Presentation.Opportunities) == 0 {
		t.Fatal("expected opportunities in projection")
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

	beforeCount := len(created.Data.Presentation.IdeaCards)
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
	if len(updated.Data.Presentation.IdeaCards) <= beforeCount {
		t.Fatalf("expected expanded idea cards > %d, got %d", beforeCount, len(updated.Data.Presentation.IdeaCards))
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

	ideaID := created.Data.Presentation.IdeaCards[0].ID
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
