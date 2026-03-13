package idea

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestIdeaDomain() *IdeaDomain {
	return &IdeaDomain{}
}

func TestApiRegenerateClusterSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, newTestIdeaDomain())

	body := []byte(`{"topic":"creator economy","cluster_id":"audience","count":4,"angle":"niche-first"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/ideas/regenerate-cluster", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if int(resp["code"].(float64)) != 200 {
		t.Fatalf("unexpected business code: %v", resp["code"])
	}
}

func TestApiFavoriteCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, newTestIdeaDomain())

	saveBody := []byte(`{"card":{"id":"fav-api-1","name":"n","one_liner":"o","target_audience":"a","core_scenario":"s","value_point":"v","business_tags":["tool"],"opportunity_tags":["niche"]}}`)
	saveReq, _ := http.NewRequest(http.MethodPost, "/api/v1/ideas/favorites", bytes.NewBuffer(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveW := httptest.NewRecorder()
	r.ServeHTTP(saveW, saveReq)

	if saveW.Code != http.StatusOK {
		t.Fatalf("save status %d", saveW.Code)
	}

	listReq, _ := http.NewRequest(http.MethodGet, "/api/v1/ideas/favorites", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status %d", listW.Code)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, "/api/v1/ideas/favorites/fav-api-1", nil)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete status %d", delW.Code)
	}
}
