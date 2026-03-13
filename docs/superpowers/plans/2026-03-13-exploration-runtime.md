# Exploration Runtime Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Exploration Runtime skeleton, Graph Intelligence stub, and ensure the new schema is migrated in the database.

**Architecture:** Introduces `Run` schema, `runtime.go` for executing exploration phases, and updates `infra/db.go` to auto-migrate the new `exploration` tables.

**Tech Stack:** Go, GORM, Eino (LLM framework)

---

### Task 1: Update Database Migration

**Files:**
- Modify: `backend/infra/db.go`

- [ ] **Step 1: Add exploration models to AutoMigrate**

```go
// Add import for "backend/domain/exploration"
// In InitDB function, update AutoMigrate call:
// db.AutoMigrate(&exploration.ExplorationSession{}, &exploration.GraphNode{}, &exploration.GraphEdge{}, &exploration.Run{})
```

### Task 2: Define Run Schema and Repository

**Files:**
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/repository.go`

- [ ] **Step 1: Add `Run` and `RunSpec` to schema.go**

```go
type RunStatus string
const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
)

type Run struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"index"`
	Status    RunStatus `json:"status"`
	Spec      string    `json:"spec"` // JSON of RunSpec
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Add CreateRun and UpdateRun to repository.go**

```go
func (d *ExplorationDomain) CreateRun(ctx context.Context, run *Run) error {
	run.CreatedAt = time.Now()
	run.UpdatedAt = time.Now()
	return d.DB.WithContext(ctx).Create(run).Error
}

func (d *ExplorationDomain) UpdateRunStatus(ctx context.Context, runID string, status RunStatus, errMsg string) error {
	return d.DB.WithContext(ctx).Model(&Run{}).Where("id = ?", runID).Updates(map[string]interface{}{
		"status":     status,
		"error":      errMsg,
		"updated_at": time.Now(),
	}).Error
}
```

### Task 3: Implement Runtime Skeleton

**Files:**
- Create: `backend/domain/exploration/runtime.go`
- Modify: `backend/domain/exploration/domain.go`

- [ ] **Step 1: Add LLM dependency to Domain**

```go
// In domain.go
import "github.com/cloudwego/eino/components/model"

type ExplorationDomain struct {
	DB  *dbdao.DB
	LLM model.ToolCallingChatModel
}
```

- [ ] **Step 2: Create runtime skeleton in `runtime.go`**

```go
package exploration

import (
	"context"
	"fmt"
)

// StartRun kicks off an exploration run asynchronously
func (d *ExplorationDomain) StartRun(ctx context.Context, sessionID string, specJSON string) (*Run, error) {
	run := &Run{
		ID:        "run_" + sessionID[:8] + "_temp", // Mock ID
		SessionID: sessionID,
		Status:    RunPending,
		Spec:      specJSON,
	}

	if err := d.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// Start execution in background
	go d.executeRun(context.Background(), run)

	return run, nil
}

func (d *ExplorationDomain) executeRun(ctx context.Context, run *Run) {
	_ = d.UpdateRunStatus(ctx, run.ID, RunRunning, "")

	err := d.runPhases(ctx, run)

	if err != nil {
		_ = d.UpdateRunStatus(ctx, run.ID, RunFailed, err.Error())
	} else {
		_ = d.UpdateRunStatus(ctx, run.ID, RunCompleted, "")
	}
}

func (d *ExplorationDomain) runPhases(ctx context.Context, run *Run) error {
	// Stub: In the future, this will do Interpret -> Explore -> Structure -> Evaluate -> Materialize -> Reflect
	fmt.Printf("Executing run %s for session %s\n", run.ID, run.SessionID)
	return nil
}
```

### Task 4: Add Run API

**Files:**
- Modify: `backend/domain/exploration/api.go`
- Modify: `backend/domain/exploration/routes.go`

- [ ] **Step 1: Add ApiStartRun**

```go
// In api.go
type StartRunReq struct {
	Spec string `json:"spec"` // e.g. target, focus areas
}

func (d *ExplorationDomain) ApiStartRun(c *gin.Context) {
	sessionID := c.Param("sessionId")
	
	var req StartRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, err := d.StartRun(c.Request.Context(), sessionID, req.Spec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, run)
}
```

- [ ] **Step 2: Register in routes.go**

```go
// In routes.go
group.POST("/sessions/:sessionId/runs", domain.ApiStartRun)
```
