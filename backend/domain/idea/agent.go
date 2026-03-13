package idea

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type IdeaGenerationAgent struct {
	Model model.ToolCallingChatModel
}

type agentPlan struct {
	Round          int            `json:"round"`
	Focuses        []string       `json:"focuses"`
	ClusterTargets map[string]int `json:"cluster_targets"`
}

type agentExecutorOutput struct {
	Clusters []OpportunityGroup `json:"clusters"`
}

type agentCriticOutput struct {
	RejectNames   []string `json:"reject_names"`
	NextFocuses   []string `json:"next_focuses"`
	QualityScore  float64  `json:"quality_score"`
	DuplicateRate float64  `json:"duplicate_rate"`
}

func (a *IdeaGenerationAgent) RunGenerate(ctx context.Context, req GenerateIdeasReq) (GenerateIdeasResp, error) {
	if a.Model == nil {
		return GenerateIdeasResp{}, fmt.Errorf("model is nil")
	}

	topic := strings.TrimSpace(req.Topic)
	angle := strings.TrimSpace(req.Angle)
	if angle == "" {
		angle = "balanced"
	}
	target := req.Count
	if target == 0 {
		target = 16
	}
	if target < 6 {
		target = 6
	}
	if target > 24 {
		target = 24
	}

	focuses := []string{"audience", "scenario", "motivation", "trend"}
	collected := map[string]OpportunityGroup{}
	seenName := map[string]bool{}
	roundsExecuted := 0
	lastQuality := 0.0
	lastDupRate := 1.0

	for round := 1; round <= 3; round++ {
		roundsExecuted = round
		plan, err := a.planRound(ctx, topic, angle, round, target, focuses, collected)
		if err != nil {
			plan = a.defaultPlan(round, target, focuses)
		}

		executed, err := a.executeRound(ctx, topic, angle, plan, seenName)
		if err != nil {
			continue
		}

		for _, cluster := range executed.Clusters {
			if cluster.ClusterID == "" || len(cluster.Ideas) == 0 {
				continue
			}
			existing := collected[cluster.ClusterID]
			if existing.ClusterID == "" {
				existing.ClusterID = cluster.ClusterID
				existing.Title = cluster.Title
			}
			for _, card := range cluster.Ideas {
				key := strings.ToLower(strings.TrimSpace(card.Name))
				if key == "" || seenName[key] {
					continue
				}
				seenName[key] = true
				existing.Ideas = append(existing.Ideas, sanitizeCard(card, cluster.ClusterID, topic, angle))
			}
			collected[cluster.ClusterID] = existing
		}

		critic, err := a.criticRound(ctx, topic, angle, round, target, collected)
		if err == nil {
			lastQuality = critic.QualityScore
			lastDupRate = critic.DuplicateRate
			if len(critic.RejectNames) > 0 {
				a.applyRejects(collected, seenName, critic.RejectNames)
			}
			if len(critic.NextFocuses) > 0 {
				focuses = critic.NextFocuses
			}
			if reachedStop(target, duplicateRate(collected), collected) && critic.QualityScore >= 3.6 {
				break
			}
		} else if reachedStop(target, duplicateRate(collected), collected) {
			break
		}
	}

	finalClusters := normalizeClusters(collected, target)
	if len(finalClusters) == 0 {
		return GenerateIdeasResp{}, fmt.Errorf("no ideas generated")
	}

	return GenerateIdeasResp{
		Topic:    topic,
		Angle:    angle,
		Clusters: finalClusters,
		Meta: GenerateMeta{
			Source:        "agent",
			Rounds:        roundsExecuted,
			QualityScore:  lastQuality,
			DuplicateRate: lastDupRate,
		},
	}, nil
}

func (a *IdeaGenerationAgent) planRound(ctx context.Context, topic, angle string, round, target int, focuses []string, collected map[string]OpportunityGroup) (agentPlan, error) {
	type plannerReq struct {
		Topic      string             `json:"topic"`
		Angle      string             `json:"angle"`
		Round      int                `json:"round"`
		Target     int                `json:"target"`
		Focuses    []string           `json:"focuses"`
		Collected  []OpportunityGroup `json:"collected"`
		Objective  string             `json:"objective"`
		Constraint string             `json:"constraint"`
	}
	in := plannerReq{
		Topic:      topic,
		Angle:      angle,
		Round:      round,
		Target:     target,
		Focuses:    focuses,
		Collected:  mapToClusters(collected),
		Objective:  "maximize novelty and coverage while keeping ideas commercially discussable",
		Constraint: "return strict json only",
	}

	var out agentPlan
	err := a.callJSON(ctx,
		plannerSystemPrompt,
		plannerUserPrompt,
		in,
		&out,
	)
	if err != nil {
		return agentPlan{}, err
	}
	if len(out.ClusterTargets) == 0 {
		return agentPlan{}, fmt.Errorf("empty plan")
	}
	return out, nil
}

func (a *IdeaGenerationAgent) executeRound(ctx context.Context, topic, angle string, plan agentPlan, seen map[string]bool) (agentExecutorOutput, error) {
	type executorReq struct {
		Topic       string            `json:"topic"`
		Angle       string            `json:"angle"`
		Plan        agentPlan         `json:"plan"`
		SeenNames   []string          `json:"seen_names"`
		Constraints map[string]string `json:"constraints"`
	}
	seenNames := make([]string, 0, len(seen))
	for k := range seen {
		seenNames = append(seenNames, k)
	}
	sort.Strings(seenNames)

	in := executorReq{
		Topic:     topic,
		Angle:     angle,
		Plan:      plan,
		SeenNames: seenNames,
		Constraints: map[string]string{
			"format": "json only",
			"shape":  "clusters[].{cluster_id,title,ideas[]}",
		},
	}
	var out agentExecutorOutput
	err := a.callJSON(ctx,
		executorSystemPrompt,
		executorUserPrompt,
		in,
		&out,
	)
	if err != nil {
		return agentExecutorOutput{}, err
	}
	return out, nil
}

func (a *IdeaGenerationAgent) criticRound(ctx context.Context, topic, angle string, round, target int, collected map[string]OpportunityGroup) (agentCriticOutput, error) {
	type criticReq struct {
		Topic     string             `json:"topic"`
		Angle     string             `json:"angle"`
		Round     int                `json:"round"`
		Target    int                `json:"target"`
		Collected []OpportunityGroup `json:"collected"`
	}
	in := criticReq{
		Topic:     topic,
		Angle:     angle,
		Round:     round,
		Target:    target,
		Collected: mapToClusters(collected),
	}
	var out agentCriticOutput
	err := a.callJSON(ctx,
		criticSystemPrompt,
		criticUserPrompt,
		in,
		&out,
	)
	return out, err
}

func (a *IdeaGenerationAgent) callJSON(ctx context.Context, systemPrompt, userPrompt string, input any, out any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	msg, err := a.Model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt + "\nInput JSON:\n" + string(raw)),
	}, model.WithTemperature(0.6))
	if err != nil {
		return err
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return fmt.Errorf("empty model response")
	}
	content = extractJSONObject(content)
	if content == "" {
		return fmt.Errorf("no json object found")
	}
	return json.Unmarshal([]byte(content), out)
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return s[start : end+1]
}

func (a *IdeaGenerationAgent) defaultPlan(round, target int, focuses []string) agentPlan {
	t := max(1, target/4)
	return agentPlan{
		Round:   round,
		Focuses: focuses,
		ClusterTargets: map[string]int{
			"audience":   t,
			"scenario":   t,
			"motivation": t,
			"trend":      t,
		},
	}
}

func (a *IdeaGenerationAgent) applyRejects(collected map[string]OpportunityGroup, seen map[string]bool, rejectNames []string) {
	reject := map[string]bool{}
	for _, n := range rejectNames {
		reject[strings.ToLower(strings.TrimSpace(n))] = true
	}
	for key, cluster := range collected {
		next := make([]IdeaCard, 0, len(cluster.Ideas))
		for _, card := range cluster.Ideas {
			name := strings.ToLower(strings.TrimSpace(card.Name))
			if reject[name] {
				delete(seen, name)
				continue
			}
			next = append(next, card)
		}
		cluster.Ideas = next
		collected[key] = cluster
	}
}

func mapToClusters(collected map[string]OpportunityGroup) []OpportunityGroup {
	keys := make([]string, 0, len(collected))
	for k := range collected {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]OpportunityGroup, 0, len(keys))
	for _, k := range keys {
		out = append(out, collected[k])
	}
	return out
}

func reachedStop(target int, dupRatio float64, collected map[string]OpportunityGroup) bool {
	total := 0
	okCluster := 0
	for _, c := range collected {
		total += len(c.Ideas)
		if len(c.Ideas) >= 3 {
			okCluster++
		}
	}
	return total >= target && dupRatio < 0.15 && okCluster >= 2
}

func duplicateRate(collected map[string]OpportunityGroup) float64 {
	total := 0
	seen := map[string]int{}
	dup := 0
	for _, c := range collected {
		for _, idea := range c.Ideas {
			total++
			key := strings.ToLower(strings.TrimSpace(idea.Name + "|" + idea.TargetAudience + "|" + idea.CoreScenario))
			seen[key]++
			if seen[key] > 1 {
				dup++
			}
		}
	}
	if total == 0 {
		return 1
	}
	return float64(dup) / float64(total)
}

func normalizeClusters(collected map[string]OpportunityGroup, target int) []OpportunityGroup {
	clusters := mapToClusters(collected)
	total := 0
	for i := range clusters {
		total += len(clusters[i].Ideas)
	}
	if total <= target {
		return clusters
	}
	overflow := total - target
	for overflow > 0 {
		for i := range clusters {
			if overflow == 0 {
				break
			}
			if len(clusters[i].Ideas) > 1 {
				clusters[i].Ideas = clusters[i].Ideas[:len(clusters[i].Ideas)-1]
				overflow--
			}
		}
		if overflow > 0 {
			break
		}
	}
	return clusters
}

func sanitizeCard(card IdeaCard, clusterID, topic, angle string) IdeaCard {
	if strings.TrimSpace(card.ID) == "" {
		card.ID = hashID(topic, clusterID, card.Name)
	}
	if strings.TrimSpace(card.Name) == "" {
		card.Name = fmt.Sprintf("%s %s", titleToken(topic), strings.Title(clusterID))
	}
	if strings.TrimSpace(card.OneLiner) == "" {
		card.OneLiner = fmt.Sprintf("An opportunity idea around %s.", topic)
	}
	if strings.TrimSpace(card.TargetAudience) == "" {
		card.TargetAudience = "builders"
	}
	if strings.TrimSpace(card.CoreScenario) == "" {
		card.CoreScenario = "weekly planning"
	}
	if strings.TrimSpace(card.ValuePoint) == "" {
		card.ValuePoint = "higher idea throughput"
	}
	if len(card.BusinessTags) == 0 {
		card.BusinessTags = []string{"tool"}
	}
	if len(card.OpportunityTags) == 0 {
		card.OpportunityTags = []string{"ai-augmented", angle}
	}
	return card
}

func max(a, b int) int {
	return int(math.Max(float64(a), float64(b)))
}
