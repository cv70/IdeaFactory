package main

import (
	"backend/config"
	"backend/domain/exploration"
	"backend/domain/idea"
	"backend/domain/user"
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

	// Register user routes
	userDomain := user.UserDomain{
		DB:    registry.DB,
		Redis: registry.Redis,
	}
	user.RegisterRoutes(v1, &userDomain)

	ideaDomain := idea.IdeaDomain{
		
	}
	idea.RegisterRoutes(v1, &ideaDomain)

	explorationDomain := exploration.NewExplorationDomain(registry.DB)
	exploration.RegisterRoutes(v1, explorationDomain)

	err = r.Run(":8888")
	mistake.Unwrap(err)
}
