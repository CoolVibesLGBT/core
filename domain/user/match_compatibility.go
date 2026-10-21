package user

import (
	"errors"
	"math"
	"strings"
)

const MatchInterestWeight = 60.0
const MatchLanguageWeight = 20.0
const MatchDistanceWeight = 20.0
const MatchDistanceScaleKM = 50.0

var ErrInvalidMatchCriteria = errors.New("invalid match criteria")

type MatchCriteria struct {
	MinAge        *int
	MaxAge        *int
	MaxDistanceKM *float64
}

func (c MatchCriteria) Validate() error {
	for _, age := range []*int{c.MinAge, c.MaxAge} {
		if age != nil && (*age < 18 || *age > 120) {
			return ErrInvalidMatchCriteria
		}
	}
	if c.MinAge != nil && c.MaxAge != nil && *c.MinAge > *c.MaxAge {
		return ErrInvalidMatchCriteria
	}
	if c.MaxDistanceKM != nil && (math.IsNaN(*c.MaxDistanceKM) || math.IsInf(*c.MaxDistanceKM, 0) || *c.MaxDistanceKM <= 0 || *c.MaxDistanceKM > 20050) {
		return ErrInvalidMatchCriteria
	}
	return nil
}

func MatchCompatibility(interests, otherInterests, languages, otherLanguages []string, distanceKM *float64) float64 {
	setScore := func(first, second []string) float64 {
		a, b := map[string]bool{}, map[string]bool{}
		for _, v := range first {
			if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
				a[v] = true
			}
		}
		for _, v := range second {
			if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
				b[v] = true
			}
		}
		intersection := 0
		for v := range a {
			if b[v] {
				intersection++
			}
		}
		union := len(a) + len(b) - intersection
		if union == 0 {
			return 0
		}
		return float64(intersection) / float64(union)
	}
	score := MatchInterestWeight*setScore(interests, otherInterests) + MatchLanguageWeight*setScore(languages, otherLanguages)
	if distanceKM != nil && !math.IsNaN(*distanceKM) && !math.IsInf(*distanceKM, 0) && *distanceKM >= 0 {
		score += MatchDistanceWeight / (1 + *distanceKM/MatchDistanceScaleKM)
	}
	return score
}
