package exploration

import "backend/datasource/dbdao"

func (d *ExplorationDomain) CreateSession(req *CreateSessionReq) (*dbdao.ExplorationSession, error) {
	session := dbdao.ExplorationSession{
		WorkspaceID: req.WorkspaceID,
		Topic:       req.Topic,
		Status:      dbdao.StatusActive,
	}

	err := d.DB.CreateSession(&session)
	return &session, err
}
