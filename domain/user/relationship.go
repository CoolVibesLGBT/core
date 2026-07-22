package user

// InteractionStateIntent describes whether a reciprocal social-graph
// relationship must be set to an explicit state or atomically toggled from its
// current state. The desired state is intentionally private so callers must use
// one of the validated constructors below.
type InteractionStateIntent struct {
	interaction InteractionKind
	positive    bool
	desired     *bool
}

// InteractionStateTransition is the logical state transition produced by an
// intent. Changed refers to the actor-facing relationship state; persistence
// adapters may still repair a missing reciprocal row without reporting a new
// logical transition.
type InteractionStateTransition struct {
	Enabled bool
	Changed bool
}

// NewSetInteractionState creates an idempotent command: applying it repeatedly
// leaves the relationship in the requested state.
func NewSetInteractionState(kind InteractionKind, enabled bool) (InteractionStateIntent, error) {
	if _, err := InteractionEngagementPair(kind, true); err != nil {
		return InteractionStateIntent{}, err
	}
	desired := enabled
	return InteractionStateIntent{interaction: kind, positive: true, desired: &desired}, nil
}

// NewToggleInteractionState creates a command that flips the actor-facing
// relationship state exactly once inside the persistence transaction.
func NewToggleInteractionState(kind InteractionKind) (InteractionStateIntent, error) {
	if _, err := InteractionEngagementPair(kind, true); err != nil {
		return InteractionStateIntent{}, err
	}
	return InteractionStateIntent{interaction: kind, positive: true}, nil
}

// NewToggleReactionState creates an atomic toggle for either the positive
// like pair or the negative dislike pair. Both directions remain one domain
// intent and therefore one repository transaction.
func NewToggleReactionState(positive bool) (InteractionStateIntent, error) {
	if _, err := InteractionEngagementPair(InteractionLike, positive); err != nil {
		return InteractionStateIntent{}, err
	}
	return InteractionStateIntent{interaction: InteractionLike, positive: positive}, nil
}

func (i InteractionStateIntent) Interaction() InteractionKind {
	return i.interaction
}

func (i InteractionStateIntent) EngagementPair() (EngagementPair, error) {
	return InteractionEngagementPair(i.interaction, i.positive)
}

func (i InteractionStateIntent) Transition(current bool) (InteractionStateTransition, error) {
	if _, err := i.EngagementPair(); err != nil {
		return InteractionStateTransition{}, err
	}

	next := !current
	if i.desired != nil {
		next = *i.desired
	}
	return InteractionStateTransition{
		Enabled: next,
		Changed: next != current,
	}, nil
}
