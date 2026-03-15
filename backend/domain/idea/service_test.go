package idea

import (
	"strings"
	"testing"
)

func TestGenerateIdeasProducesClustersAndCards(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.GenerateIdeas(GenerateIdeasReq{
		Topic: "AI education",
		Count: 16,
	})
	if err != nil {
		t.Fatalf("GenerateIdeas returned error: %v", err)
	}

	if resp.Topic != "AI education" {
		t.Fatalf("unexpected topic: %s", resp.Topic)
	}
	if len(resp.Clusters) < 3 {
		t.Fatalf("expected at least 3 clusters, got %d", len(resp.Clusters))
	}

	totalCards := 0
	for _, cluster := range resp.Clusters {
		if cluster.ClusterID == "" {
			t.Fatalf("cluster id is empty")
		}
		if cluster.Title == "" {
			t.Fatalf("cluster title is empty")
		}
		if len(cluster.Ideas) == 0 {
			t.Fatalf("cluster %s has no ideas", cluster.ClusterID)
		}
		totalCards += len(cluster.Ideas)
		for _, card := range cluster.Ideas {
			if card.ID == "" || card.Name == "" || card.OneLiner == "" {
				t.Fatalf("invalid card: %+v", card)
			}
			if card.TargetAudience == "" || card.CoreScenario == "" || card.ValuePoint == "" {
				t.Fatalf("card missing required fields: %+v", card)
			}
			if len(card.BusinessTags) == 0 || len(card.OpportunityTags) == 0 {
				t.Fatalf("card missing tags: %+v", card)
			}
		}
	}

	if totalCards < 12 {
		t.Fatalf("expected >= 12 cards, got %d", totalCards)
	}
}

func TestGenerateIdeasAvoidsDuplicateNames(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.GenerateIdeas(GenerateIdeasReq{
		Topic: "pet economy",
		Count: 20,
	})
	if err != nil {
		t.Fatalf("GenerateIdeas returned error: %v", err)
	}

	seen := map[string]bool{}
	for _, cluster := range resp.Clusters {
		for _, card := range cluster.Ideas {
			key := strings.ToLower(strings.TrimSpace(card.Name))
			if seen[key] {
				t.Fatalf("duplicate card name found: %s", card.Name)
			}
			seen[key] = true
		}
	}
}

func TestExpandIdeasProducesVariants(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.ExpandIdeas(ExpandIdeasReq{
		Topic:      "creator economy",
		BaseIdeaID: "abc123",
		BaseName:   "Creator Audience Slice 1",
		Count:      5,
		Angle:      "profit-first",
	})
	if err != nil {
		t.Fatalf("ExpandIdeas returned error: %v", err)
	}

	if len(resp.Ideas) != 5 {
		t.Fatalf("expected 5 expanded ideas, got %d", len(resp.Ideas))
	}
	for _, card := range resp.Ideas {
		if card.ID == "" || card.Name == "" || card.OneLiner == "" {
			t.Fatalf("invalid expanded card: %+v", card)
		}
	}
}

func TestRegenerateClusterProducesCards(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.RegenerateCluster(RegenerateClusterReq{
		Topic:     "creator economy",
		ClusterID: "audience",
		Count:     4,
		Angle:     "niche-first",
	})
	if err != nil {
		t.Fatalf("RegenerateCluster returned error: %v", err)
	}
	if resp.ClusterID != "audience" {
		t.Fatalf("expected cluster audience, got %s", resp.ClusterID)
	}
	if len(resp.Ideas) != 4 {
		t.Fatalf("expected 4 ideas, got %d", len(resp.Ideas))
	}
}

func TestFavoriteCRUD(t *testing.T) {
	d := newTestIdeaDomain()

	card := IdeaCard{
		ID:              "fav-1",
		Name:            "Idea 1",
		OneLiner:        "One line",
		TargetAudience:  "founders",
		CoreScenario:    "planning",
		ValuePoint:      "speed",
		BusinessTags:    []string{"tool"},
		OpportunityTags: []string{"niche"},
	}
	if err := d.SaveFavorite(SaveFavoriteReq{Card: card}); err != nil {
		t.Fatalf("SaveFavorite returned error: %v", err)
	}
	list := d.ListFavorites()
	if len(list.Ideas) == 0 {
		t.Fatalf("expected non-empty favorites")
	}

	if err := d.RemoveFavorite("fav-1"); err != nil {
		t.Fatalf("RemoveFavorite returned error: %v", err)
	}
	after := d.ListFavorites()
	for _, it := range after.Ideas {
		if it.ID == "fav-1" {
			t.Fatalf("favorite was not removed")
		}
	}
}
