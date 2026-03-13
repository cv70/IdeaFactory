package user

import (
	"backend/datasource/dbdao"

	"github.com/redis/go-redis/v9"
)

type UserDomain struct {
	DB    *dbdao.DB
	Redis *redis.Client
}
