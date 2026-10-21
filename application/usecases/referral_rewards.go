package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"github.com/google/uuid"
)

type ReferralRewardService struct {
	repo ports.ReferralRewardRepository
}

func NewReferralRewardService(repo ports.ReferralRewardRepository) *ReferralRewardService {
	return &ReferralRewardService{repo: repo}
}
func (s *ReferralRewardService) List(ctx context.Context, p ports.ModeratorPrincipal, reviewQueue bool, status domainuser.ReferralRewardStatus, cursor *types.MatchListCursor, limit int) ([]ports.ReferralRewardView, *types.MatchListCursor, error) {
	if p.ID == uuid.Nil || (reviewQueue && !domainuser.CanReviewReferral(domainuser.AccountRole(p.Role))) {
		return nil, nil, domainuser.ErrReferralReviewForbidden
	}
	if status != "" && status != domainuser.ReferralRewardPending && status != domainuser.ReferralRewardPaid && status != domainuser.ReferralRewardRejected {
		return nil, nil, domainuser.ErrReferralReviewInvalid
	}
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}
	return s.repo.ListReferralRewards(ctx, p.ID, reviewQueue, status, cursor, limit)
}
func (s *ReferralRewardService) Review(ctx context.Context, p ports.ModeratorPrincipal, id uuid.UUID, decision, note, reference string) (ports.ReferralRewardView, error) {
	if p.ID == uuid.Nil || !domainuser.CanReviewReferral(domainuser.AccountRole(p.Role)) {
		return ports.ReferralRewardView{}, domainuser.ErrReferralReviewForbidden
	}
	if id == uuid.Nil {
		return ports.ReferralRewardView{}, domainuser.ErrReferralReviewInvalid
	}
	review, err := domainuser.NewReferralReview(decision, note, reference)
	if err != nil {
		return ports.ReferralRewardView{}, err
	}
	return s.repo.ReviewReferralReward(ctx, p.ID, id, review)
}
