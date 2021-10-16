package dnswarmer

import (
	"time"
)

type scorer interface {
	Score(query string, previous int64, t time.Time) int64
}

type emaScorer struct {
	alpha float64
}

func newEMAScorer(alpha float64) *emaScorer {
	return &emaScorer{alpha: alpha}
}

func (s *emaScorer) Score(query string, previous int64, t time.Time) int64 {
	return int64(float64(previous)*(1-s.alpha) + float64(t.Unix())*s.alpha)
}
