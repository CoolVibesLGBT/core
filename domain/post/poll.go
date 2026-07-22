package post

import (
	"errors"
	"strings"
)

var (
	ErrInvalidPollKind       = errors.New("invalid poll kind")
	ErrPollSelectionLimit    = errors.New("poll selection limit reached")
	ErrInvalidPollWeight     = errors.New("invalid poll vote weight")
	ErrInvalidPollRank       = errors.New("invalid poll vote rank")
	ErrDuplicatePollRank     = errors.New("poll rank is already used")
	ErrInvalidPollChoiceData = errors.New("invalid poll choice data")
	ErrPollQuestionRequired  = errors.New("poll question is required")
	ErrPollOptionsRequired   = errors.New("at least two poll options are required")
	ErrDuplicatePollOption   = errors.New("poll options must be unique")
	ErrInvalidPollMaximum    = errors.New("invalid poll maximum selections")
)

type PollDefinition struct {
	Question      string
	Kind          PollKind
	MaxSelectable int
	Options       []string
}

func NewPollDefinition(question string, kind PollKind, maxSelectable int, options []string) (PollDefinition, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return PollDefinition{}, ErrPollQuestionRequired
	}
	if kind == "" {
		kind = PollSingle
	}
	switch kind {
	case PollSingle, PollMultiple, PollRanked, PollWeighted:
	default:
		return PollDefinition{}, ErrInvalidPollKind
	}

	normalizedOptions := make([]string, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" {
			return PollDefinition{}, ErrInvalidPollChoiceData
		}
		identity := strings.ToLower(option)
		if _, duplicate := seen[identity]; duplicate {
			return PollDefinition{}, ErrDuplicatePollOption
		}
		seen[identity] = struct{}{}
		normalizedOptions = append(normalizedOptions, option)
	}
	if len(normalizedOptions) < 2 {
		return PollDefinition{}, ErrPollOptionsRequired
	}

	if kind == PollSingle {
		if maxSelectable != 1 {
			return PollDefinition{}, ErrInvalidPollMaximum
		}
	} else if maxSelectable < 1 || maxSelectable > len(normalizedOptions) {
		return PollDefinition{}, ErrInvalidPollMaximum
	}

	return PollDefinition{
		Question:      question,
		Kind:          kind,
		MaxSelectable: maxSelectable,
		Options:       normalizedOptions,
	}, nil
}

type PollKind string

const (
	PollSingle   PollKind = "single"
	PollMultiple PollKind = "multiple"
	PollRanked   PollKind = "ranked"
	PollWeighted PollKind = "weighted"
)

// PollVotePolicy is the domain invariant evaluated before a new vote is
// persisted. Removing an existing vote is always allowed and is handled by
// the aggregate command before this validation.
type PollVotePolicy struct {
	Kind          PollKind
	MaxSelectable int
	ChoiceCount   int
}

func (p PollVotePolicy) ValidateNewVote(currentSelections, weight, rank int, rankAlreadyUsed bool) error {
	if p.ChoiceCount <= 0 || currentSelections < 0 {
		return ErrInvalidPollChoiceData
	}

	maxSelections := p.MaxSelectable
	if maxSelections <= 0 || maxSelections > p.ChoiceCount {
		maxSelections = p.ChoiceCount
	}

	switch p.Kind {
	case "", PollSingle:
		if weight != 1 {
			return ErrInvalidPollWeight
		}
		if rank != 0 {
			return ErrInvalidPollRank
		}
		// Selecting another choice replaces the existing single selection.
		return nil

	case PollMultiple:
		if weight != 1 {
			return ErrInvalidPollWeight
		}
		if rank != 0 {
			return ErrInvalidPollRank
		}
		if currentSelections >= maxSelections {
			return ErrPollSelectionLimit
		}
		return nil

	case PollRanked:
		if weight != 1 {
			return ErrInvalidPollWeight
		}
		if rank <= 0 || rank > maxSelections {
			return ErrInvalidPollRank
		}
		if rankAlreadyUsed {
			return ErrDuplicatePollRank
		}
		if currentSelections >= maxSelections {
			return ErrPollSelectionLimit
		}
		return nil

	case PollWeighted:
		if weight <= 0 {
			return ErrInvalidPollWeight
		}
		if rank != 0 {
			return ErrInvalidPollRank
		}
		if currentSelections >= maxSelections {
			return ErrPollSelectionLimit
		}
		return nil

	default:
		return ErrInvalidPollKind
	}
}
