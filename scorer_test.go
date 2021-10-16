package dnswarmer

import (
	"fmt"
	"testing"
	"time"
)

func TestEMAScorer(t *testing.T) {
	s := newEMAScorer(0.25)

	testCases := []struct {
		current  int64
		query    string
		t        time.Time
		expected int64
	}{
		{0, "a.com", time.Unix(1600354415, 0), 400088603},
		{400088603, "a.com", time.Unix(1600354427, 0), 700155059},
		{0, "b.com", time.Unix(1600354427, 0), 400088606},
		{400088606, "b.com", time.Unix(1600354427, 0), 700155061},
		{700155061, "b.com", time.Unix(1600354430, 0), 925204903},
	}
	for i, tC := range testCases {
		t.Run(fmt.Sprintf("%s-%d", tC.query, i), func(t *testing.T) {
			if got, want := s.Score(tC.query, tC.current, tC.t), tC.expected; got != want {
				t.Errorf("unexpected score: got %d, want %d", got, want)
			}
		})
	}
}
