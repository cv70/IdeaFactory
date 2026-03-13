package scylladao

import (
	"backend/utils"
	"time"

	"github.com/gocql/gocql"
)

type ScyllaDB gocql.Session

func NewScyllaDB(sess *gocql.Session) *ScyllaDB {
	return (*ScyllaDB)(sess)
}

func (d *ScyllaDB) DB() *gocql.Session {
	return d.DB()
}

type BaseModel struct {
	ID        int64
	UpdatedAt time.Time
}

func (m *BaseModel) Reset() (err error) {
	m.ID, err = utils.NewIDInt64()
	if err != nil {
		return err
	}
	m.UpdatedAt = utils.IDInt64ToTime(m.ID)
	return nil
}
