# Exploration Core Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the foundational Exploration Graph and Runtime models from the Technical Design.

**Architecture:** Moving from a simple idea generator to a Graph-First Exploration OS. Defines the core GraphNode, GraphEdge, and runtime structs, and implements a basic repository over GORM.

**Tech Stack:** Go, GORM, Gin

---

### Task 1: Create Exploration Domain and Core Schema

**Files:**
- Create: `backend/domain/exploration/schema.go`
- Create: `backend/domain/exploration/domain.go`

- [ ] **Step 1: Write core schema definitions in `backend/domain/exploration/schema.go`**

```go
package exploration

import "time"

type NodeType string
const (
	NodeTopic       NodeType = "Topic"
	NodeQuestion    NodeType = "Question"
	NodeTension     NodeType = "Tension"
	NodeHypothesis  NodeType = "Hypothesis"
	NodeOpportunity NodeType = "Opportunity"
	NodeIdea        NodeType = "Idea"
	NodeEvidence    NodeType = "Evidence"
	NodeClaim       NodeType = "Claim"
	NodeDecision    NodeType = "Decision"
	NodeUnknown     NodeType = "Unknown"
)

type GraphNode struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	SessionID   string    `json:"session_id" gorm:"index"`
	Type        NodeType  `json:"node_type"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	Metadata    string    `json:"metadata"` // JSON string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EdgeType string
const (
	EdgeQuestions   EdgeType = "questions"
	EdgeExplains    EdgeType = "explains"
	EdgeSupports    EdgeType = "supports"
	EdgeWeakens     EdgeType = "weakens"
	EdgeLeadsTo     EdgeType = "leads_to"
)

type GraphEdge struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"index"`
	FromID    string    `json:"from_node_id"`
	ToID      string    `json:"to_node_id"`
	Type      EdgeType  `json:"edge_type"`
	CreatedAt time.Time `json:"created_at"`
}

// ExplorationSession represents a single exploration context
type ExplorationSession struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Topic       string    `json:"topic"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create `backend/domain/exploration/domain.go`**

```go
package exploration

import (
	"backend/datasource/dbdao"
)

type ExplorationDomain struct {
	DB *dbdao.DB
}
```

### Task 2: Implement Repository Layer for Graph

**Files:**
- Create: `backend/domain/exploration/repository.go`

- [ ] **Step 1: Write repository functions**

```go
package exploration

import (
	"context"
	"time"
)

func (d *ExplorationDomain) CreateSession(ctx context.Context, session *ExplorationSession) error {
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	return d.DB.WithContext(ctx).Create(session).Error
}

func (d *ExplorationDomain) CreateNode(ctx context.Context, node *GraphNode) error {
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	return d.DB.WithContext(ctx).Create(node).Error
}

func (d *ExplorationDomain) CreateEdge(ctx context.Context, edge *GraphEdge) error {
	edge.CreatedAt = time.Now()
	return d.DB.WithContext(ctx).Create(edge).Error
}

func (d *ExplorationDomain) GetSessionGraph(ctx context.Context, sessionID string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	if err := d.DB.WithContext(ctx).Where("session_id = ?", sessionID).Find(&nodes).Error; err != nil {
		return nil, nil, err
	}

	var edges []GraphEdge
	if err := d.DB.WithContext(ctx).Where("session_id = ?", sessionID).Find(&edges).Error; err != nil {
		return nil, nil, err
	}

	return nodes, edges, nil
}
```

### Task 3: API & Routes Setup

**Files:**
- Create: `backend/domain/exploration/api.go`
- Create: `backend/domain/exploration/routes.go`
- Modify: `backend/main.go`

- [ ] **Step 1: Write API handlers in `backend/domain/exploration/api.go`**

```go
package exploration

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreateSessionReq struct {
	WorkspaceID string `json:"workspace_id" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
}

func (d *ExplorationDomain) ApiCreateSession(c *gin.Context) {
	var req CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := ExplorationSession{
		ID:          "sess_" + req.Topic[:2], // Mock ID generation
		WorkspaceID: req.WorkspaceID,
		Topic:       req.Topic,
		Status:      "active",
	}

	if err := d.CreateSession(c.Request.Context(), &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}
```

- [ ] **Step 2: Register routes in `backend/domain/exploration/routes.go`**

```go
package exploration

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, domain *ExplorationDomain) {
	group := router.Group("/exploration")
	{
		group.POST("/sessions", domain.ApiCreateSession)
	}
}
```

- [ ] **Step 3: Update `backend/main.go` to include the exploration domain**

Add these to main():
```go
	explorationDomain := exploration.ExplorationDomain{
		DB: registry.DB,
	}
	exploration.RegisterRoutes(v1, &explorationDomain)
```
