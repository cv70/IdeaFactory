package idea

type GenerateIdeasReq struct {
	Topic   string `json:"topic" binding:"required,min=2,max=100"`
	Count   int    `json:"count" binding:"omitempty,min=6,max=24"`
	Angle   string `json:"angle" binding:"omitempty,max=64"`
	Market  string `json:"market" binding:"omitempty,max=32"`
	Novelty string `json:"novelty" binding:"omitempty,max=32"`
}

type GenerateIdeasResp struct {
	Topic    string             `json:"topic"`
	Angle    string             `json:"angle"`
	Clusters []OpportunityGroup `json:"clusters"`
	Meta     GenerateMeta       `json:"meta"`
}

type GenerateMeta struct {
	Source        string  `json:"source"`
	Rounds        int     `json:"rounds"`
	QualityScore  float64 `json:"quality_score"`
	DuplicateRate float64 `json:"duplicate_rate"`
}

type ExpandIdeasReq struct {
	Topic      string `json:"topic" binding:"required,min=2,max=100"`
	BaseIdeaID string `json:"base_idea_id" binding:"required,min=2,max=64"`
	BaseName   string `json:"base_name" binding:"required,min=2,max=120"`
	Count      int    `json:"count" binding:"omitempty,min=3,max=8"`
	Angle      string `json:"angle" binding:"omitempty,max=64"`
}

type ExpandIdeasResp struct {
	Topic      string     `json:"topic"`
	BaseIdeaID string     `json:"base_idea_id"`
	BaseName   string     `json:"base_name"`
	Ideas      []IdeaCard `json:"ideas"`
}

type RegenerateClusterReq struct {
	Topic     string `json:"topic" binding:"required,min=2,max=100"`
	ClusterID string `json:"cluster_id" binding:"required,min=2,max=32"`
	Count     int    `json:"count" binding:"omitempty,min=3,max=12"`
	Angle     string `json:"angle" binding:"omitempty,max=64"`
}

type RegenerateClusterResp struct {
	Topic     string     `json:"topic"`
	ClusterID string     `json:"cluster_id"`
	Title     string     `json:"title"`
	Ideas     []IdeaCard `json:"ideas"`
}

type SaveFavoriteReq struct {
	Card IdeaCard `json:"card" binding:"required"`
}

type ListFavoritesResp struct {
	Ideas []IdeaCard `json:"ideas"`
}

type OpportunityGroup struct {
	ClusterID string     `json:"cluster_id"`
	Title     string     `json:"title"`
	Ideas     []IdeaCard `json:"ideas"`
}

type IdeaCard struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	OneLiner        string   `json:"one_liner"`
	TargetAudience  string   `json:"target_audience"`
	CoreScenario    string   `json:"core_scenario"`
	ValuePoint      string   `json:"value_point"`
	BusinessTags    []string `json:"business_tags"`
	OpportunityTags []string `json:"opportunity_tags"`
}
