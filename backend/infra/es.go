package infra

import (
	"backend/config"
	"context"

	"github.com/elastic/go-elasticsearch/v9"
)

func NewES(ctx context.Context, c *config.ESConfig) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			c.Host,
		},
		Username: c.User,
		Password: c.Password,
	}
	esClient, err := elasticsearch.NewClient(cfg)
	return esClient, err
}
