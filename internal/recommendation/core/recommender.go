package core

import (
	"cmp"
	"context"
	"slices"
)

type GenreRecommender struct{ catalog Catalog }

func NewGenreRecommender(catalog Catalog) *GenreRecommender {
	return &GenreRecommender{catalog: catalog}
}

func (r *GenreRecommender) Recommend(ctx context.Context, input Input) ([]Series, error) {
	catalog, err := r.catalog.Series(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Series, len(catalog))
	for _, series := range catalog {
		byID[series.ID] = series
	}
	excluded := make(map[string]bool)
	profile := func(ids []string) map[string]float64 {
		counts := make(map[string]float64)
		for _, id := range ids {
			excluded[id] = true
			if s, ok := byID[id]; ok && s.Genre != "" {
				counts[s.Genre] += 1 / float64(len(ids))
			}
		}
		return counts
	}
	owner, partner := profile(input.OwnerIDs), profile(input.PartnerIDs)
	type scored struct {
		series Series
		score  float64
	}
	ranked := make([]scored, 0, len(catalog))
	for _, series := range catalog {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if excluded[series.ID] {
			continue
		}
		score := input.Weight.Value()*owner[series.Genre] + (1-input.Weight.Value())*partner[series.Genre]
		if score > 0 {
			ranked = append(ranked, scored{series, score})
		}
	}
	slices.SortFunc(ranked, func(a, b scored) int {
		if order := cmp.Compare(b.score, a.score); order != 0 {
			return order
		}
		return cmp.Compare(a.series.ID, b.series.ID)
	})
	result := make([]Series, 0, min(10, len(ranked)))
	for _, item := range ranked[:min(10, len(ranked))] {
		result = append(result, item.series)
	}
	return result, nil
}
