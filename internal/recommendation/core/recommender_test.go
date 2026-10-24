package core

import (
	"context"
	"fmt"
	"math"
	"testing"
)

type catalogStub []Series

func (c catalogStub) Series(context.Context) ([]Series, error) { return c, nil }

func TestGenreRankingWeightAndExclusion(t *testing.T) {
	catalog := catalogStub{
		{ID: "own", Genre: "drama"}, {ID: "partner", Genre: "comedy"},
		{ID: "d1", Genre: "drama"}, {ID: "d2", Genre: "drama"}, {ID: "c1", Genre: "comedy"},
		{ID: "unknown", Genre: ""},
	}
	for _, tc := range []struct {
		weight float64
		first  string
		count  int
	}{{1, "d1", 2}, {0, "c1", 1}, {0.5, "c1", 3}} {
		w, _ := NewWeight(tc.weight)
		got, err := NewGenreRecommender(catalog).Recommend(t.Context(), Input{OwnerIDs: []string{"own"}, PartnerIDs: []string{"partner"}, Weight: w})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != tc.count || got[0].ID != tc.first {
			t.Fatalf("weight %v: %+v", tc.weight, got)
		}
	}
}
func TestRecommendationLimitAndStableTieOrder(t *testing.T) {
	catalog := catalogStub{{ID: "seed", Genre: "g"}}
	for i := range 20 {
		catalog = append(catalog, Series{ID: fmt.Sprintf("%02d", i), Genre: "g"})
	}
	w, _ := NewWeight(0.5)
	got, err := NewGenreRecommender(catalog).Recommend(t.Context(), Input{OwnerIDs: []string{"seed"}, PartnerIDs: []string{"seed"}, Weight: w})
	if err != nil || len(got) != 10 || got[0].ID != "00" || got[9].ID != "09" {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestValueObjects(t *testing.T) {
	rounded, err := NewWeight(0.123456)
	if err != nil || rounded.Value() != 0.1235 {
		t.Fatalf("rounded weight = %v, %v", rounded.Value(), err)
	}
	for _, w := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		if _, err := NewWeight(w); err == nil {
			t.Fatalf("accepted %v", w)
		}
	}
	for _, ids := range [][]string{nil, {""}, {"s", "s"}, {" "}} {
		if _, err := NewSelection(ids); err == nil {
			t.Fatalf("accepted %+v", ids)
		}
	}
	input := []string{"b", "a"}
	s, err := NewSelection(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = "changed"
	got := s.IDs()
	got[0] = "changed"
	if s.IDs()[0] != "a" || s.IDs()[1] != "b" {
		t.Fatal("selection aliases caller slice")
	}
}
