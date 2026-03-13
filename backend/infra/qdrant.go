package infra

import (
	"backend/config"
	"backend/datasource/vectordao"
	"context"

	"github.com/qdrant/go-client/qdrant"
)

func NewQdrant(ctx context.Context, c *config.QdrantConfig) (*vectordao.VectorDB, error) {
	cfg := qdrant.Config{
		Host:   c.Host,
		Port:   c.Port,
		APIKey: c.APIKey,
		UseTLS: true,  // uses default config with minimum TLS version set to 1.3
		// PoolSize: 3,
		// KeepAliveTime: 10,
		// KeepAliveTimeout: 2,
		// TLSConfig: &tls.Config{...},
		// GrpcOptions: []grpc.DialOption{},
	}

	client, err := qdrant.NewClient(&cfg)

	return vectordao.NewVectorDB(client), err
}
