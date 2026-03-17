package vectordao

import (
	"github.com/qdrant/go-client/qdrant"
)

// VectorDB 封装了Qdrant客户端，提供向量数据库操作。
type VectorDB struct {
	*qdrant.Client
}

// NewVectorDB 创建一个新的VectorDB实例。
// 参数:
//   - cli: Qdrant客户端实例
//
// 返回:
//   - *VectorDB: VectorDB实例
func NewVectorDB(cli *qdrant.Client) *VectorDB {
	return &VectorDB{
		Client: cli,
	}
}
