package main

import (
	"backend/config"
	"backend/domain/exploration"
	"backend/domain/idea"
	"backend/infra"
	"context"

	"github.com/cv70/pkgo/mistake"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	mistake.Unwrap(err)

	// Initialize infrastructure with configuration
	registry, err := infra.NewRegistry(ctx, cfg)
	mistake.Unwrap(err)

	r := gin.Default()
	v1 := r.Group("/api/v1")

	ideaDomain, err := idea.BuildIdeaDomain(registry)
	mistake.Unwrap(err)
	idea.RegisterRoutes(v1, ideaDomain)

	explorationDomain, err := exploration.BuildExplorationDomain(registry)
	mistake.Unwrap(err)
	exploration.RegisterRoutes(v1, explorationDomain)
	go explorationDomain.Start(ctx) // resume active workspaces after restart

	err = r.Run(":8888")
	mistake.Unwrap(err)
}
