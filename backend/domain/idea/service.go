package idea

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	favMu   sync.RWMutex
	favData = map[string]IdeaCard{}
)

func (d *IdeaDomain) GenerateIdeas(req GenerateIdeasReq) (GenerateIdeasResp, error) {
	if d.LLM != nil {
		agent := IdeaGenerationAgent{Model: d.LLM}
		resp, err := agent.RunGenerate(context.Background(), req)
		if err == nil && len(resp.Clusters) > 0 {
			return resp, nil
		}
	}

	resp, err := d.generateIdeasDeterministic(req)
	if err != nil {
		return GenerateIdeasResp{}, err
	}
	resp.Meta = GenerateMeta{
		Source:        "fallback",
		Rounds:        0,
		QualityScore:  0,
		DuplicateRate: duplicateRate(sliceToMap(resp.Clusters)),
	}
	return resp, nil
}

func (d *IdeaDomain) generateIdeasDeterministic(req GenerateIdeasReq) (GenerateIdeasResp, error) {
	topic := strings.TrimSpace(req.Topic)
	angle := strings.TrimSpace(req.Angle)
	if angle == "" {
		angle = "balanced"
	}

	count := req.Count
	if count == 0 {
		count = 16
	}
	if count < 6 {
		count = 6
	}
	if count > 24 {
		count = 24
	}

	clusters := []OpportunityGroup{
		{ClusterID: "audience", Title: "Audience Slice"},
		{ClusterID: "scenario", Title: "Scenario Slice"},
		{ClusterID: "motivation", Title: "Motivation Slice"},
		{ClusterID: "trend", Title: "Trend Cross Slice"},
	}

	segments := []string{
		"solo founders",
		"growth operators",
		"small teams",
		"cross-border sellers",
		"creator economy operators",
		"niche service providers",
	}
	scenarios := []string{
		"weekly planning",
		"campaign launch",
		"customer retention",
		"offer validation",
		"content repurposing",
		"funnel optimization",
	}
	values := []string{
		"faster idea throughput with lower noise",
		"clearer niche positioning without research fatigue",
		"new monetization angles from existing assets",
		"higher campaign freshness with less manual brainstorming",
		"better opportunity discovery from weak market signals",
		"repeatable creative output for recurring growth tasks",
	}
	businessTags := []string{"tool", "content", "service", "community", "subscription", "transaction"}
	opportunityTags := []string{"high-frequency", "efficiency", "info-gap", "emotional-value", "niche", "ai-augmented"}

	targetPerCluster := count / len(clusters)
	if targetPerCluster == 0 {
		targetPerCluster = 1
	}

	seenNames := map[string]bool{}
	serial := 0
	for i := range clusters {
		for len(clusters[i].Ideas) < targetPerCluster {
			name := fmt.Sprintf("%s %s %d", titleToken(topic), strings.Title(clusters[i].ClusterID), serial+1)
			nameKey := strings.ToLower(strings.TrimSpace(name))
			if seenNames[nameKey] {
				serial++
				continue
			}
			seenNames[nameKey] = true

			seg := segments[(serial+i)%len(segments)]
			scn := scenarios[(serial+i)%len(scenarios)]
			val := values[(serial+i)%len(values)]

			card := IdeaCard{
				ID:             hashID(topic, clusters[i].ClusterID, serial),
				Name:           name,
				OneLiner:       fmt.Sprintf("A %s idea for %s around %s.", clusters[i].ClusterID, seg, topic),
				TargetAudience: seg,
				CoreScenario:   scn,
				ValuePoint:     val,
				BusinessTags: []string{
					businessTags[(serial+i)%len(businessTags)],
					businessTags[(serial+i+2)%len(businessTags)],
				},
				OpportunityTags: []string{
					opportunityTags[(serial+i)%len(opportunityTags)],
					opportunityTags[(serial+i+3)%len(opportunityTags)],
					angle,
				},
			}
			clusters[i].Ideas = append(clusters[i].Ideas, card)
			serial++
		}
	}

	// Fill remainder to reach requested count.
	total := 0
	for _, cluster := range clusters {
		total += len(cluster.Ideas)
	}
	for total < count {
		idx := total % len(clusters)
		name := fmt.Sprintf("%s Variant %d", titleToken(topic), total+1)
		nameKey := strings.ToLower(name)
		if seenNames[nameKey] {
			total++
			continue
		}
		seenNames[nameKey] = true
		card := IdeaCard{
			ID:             hashID(topic, "extra", total),
			Name:           name,
			OneLiner:       fmt.Sprintf("A variant opportunity for %s.", topic),
			TargetAudience: segments[total%len(segments)],
			CoreScenario:   scenarios[total%len(scenarios)],
			ValuePoint:     values[total%len(values)],
			BusinessTags:   []string{businessTags[total%len(businessTags)]},
			OpportunityTags: []string{
				opportunityTags[total%len(opportunityTags)],
				angle,
			},
		}
		clusters[idx].Ideas = append(clusters[idx].Ideas, card)
		total++
	}

	return GenerateIdeasResp{
		Topic:    topic,
		Angle:    angle,
		Clusters: clusters,
	}, nil
}

func sliceToMap(clusters []OpportunityGroup) map[string]OpportunityGroup {
	out := make(map[string]OpportunityGroup, len(clusters))
	for _, c := range clusters {
		out[c.ClusterID] = c
	}
	return out
}

func (d *IdeaDomain) ExpandIdeas(req ExpandIdeasReq) (ExpandIdeasResp, error) {
	count := req.Count
	if count == 0 {
		count = 5
	}
	if count < 3 {
		count = 3
	}
	if count > 8 {
		count = 8
	}

	angle := strings.TrimSpace(req.Angle)
	if angle == "" {
		angle = "balanced"
	}

	ideas := make([]IdeaCard, 0, count)
	for i := 0; i < count; i++ {
		ideas = append(ideas, IdeaCard{
			ID:             hashID(req.Topic, req.BaseIdeaID, i, angle),
			Name:           fmt.Sprintf("%s Variant %d", strings.TrimSpace(req.BaseName), i+1),
			OneLiner:       fmt.Sprintf("A %s variant derived from %s for %s.", angle, req.BaseName, req.Topic),
			TargetAudience: []string{"solo founders", "operators", "small teams", "creators"}[i%4],
			CoreScenario:   []string{"idea validation", "weekly planning", "campaign refresh", "offer iteration"}[i%4],
			ValuePoint:     []string{"higher idea diversity", "faster iteration", "new monetization angle", "fresh audience entry"}[i%4],
			BusinessTags: []string{
				[]string{"tool", "content", "service", "subscription"}[i%4],
			},
			OpportunityTags: []string{
				[]string{"niche", "efficiency", "info-gap", "ai-augmented"}[i%4],
				angle,
			},
		})
	}

	return ExpandIdeasResp{
		Topic:      strings.TrimSpace(req.Topic),
		BaseIdeaID: req.BaseIdeaID,
		BaseName:   strings.TrimSpace(req.BaseName),
		Ideas:      ideas,
	}, nil
}

func (d *IdeaDomain) RegenerateCluster(req RegenerateClusterReq) (RegenerateClusterResp, error) {
	count := req.Count
	if count == 0 {
		count = 6
	}
	if count < 3 {
		count = 3
	}
	if count > 12 {
		count = 12
	}
	angle := strings.TrimSpace(req.Angle)
	if angle == "" {
		angle = "balanced"
	}

	seedIdeas, err := d.ExpandIdeas(ExpandIdeasReq{
		Topic:      req.Topic,
		BaseIdeaID: req.ClusterID,
		BaseName:   fmt.Sprintf("%s %s", titleToken(req.Topic), strings.Title(req.ClusterID)),
		Count:      count,
		Angle:      angle,
	})
	if err != nil {
		return RegenerateClusterResp{}, err
	}

	titleMap := map[string]string{
		"audience":   "Audience Slice",
		"scenario":   "Scenario Slice",
		"motivation": "Motivation Slice",
		"trend":      "Trend Cross Slice",
	}

	return RegenerateClusterResp{
		Topic:     strings.TrimSpace(req.Topic),
		ClusterID: strings.TrimSpace(req.ClusterID),
		Title:     titleMap[strings.TrimSpace(req.ClusterID)],
		Ideas:     seedIdeas.Ideas,
	}, nil
}

func (d *IdeaDomain) SaveFavorite(req SaveFavoriteReq) error {
	if strings.TrimSpace(req.Card.ID) == "" {
		return errors.New("empty card id")
	}
	favMu.Lock()
	defer favMu.Unlock()
	favData[req.Card.ID] = req.Card
	return nil
}

func (d *IdeaDomain) ListFavorites() ListFavoritesResp {
	favMu.RLock()
	defer favMu.RUnlock()
	out := make([]IdeaCard, 0, len(favData))
	for _, card := range favData {
		out = append(out, card)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return ListFavoritesResp{Ideas: out}
}

func (d *IdeaDomain) RemoveFavorite(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("empty card id")
	}
	favMu.Lock()
	defer favMu.Unlock()
	delete(favData, id)
	return nil
}

func hashID(parts ...interface{}) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(fmt.Sprint(part)))
		_, _ = h.Write([]byte("|"))
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func titleToken(topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "Idea"
	}
	if len(topic) > 24 {
		topic = topic[:24]
	}
	return strings.Title(topic)
}
