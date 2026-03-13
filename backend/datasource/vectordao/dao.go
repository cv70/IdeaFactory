package vectordao

import (
	"github.com/qdrant/go-client/qdrant"
)

type VectorDB struct {
	*qdrant.Client
}

func NewVectorDB(cli *qdrant.Client) *VectorDB {
	return &VectorDB{
		Client: cli,
	}
}
