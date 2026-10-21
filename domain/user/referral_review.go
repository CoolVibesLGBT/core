package user

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

type ReferralRewardStatus string

const (
	ReferralRewardPending  ReferralRewardStatus = "pending"
	ReferralRewardPaid     ReferralRewardStatus = "paid"
	ReferralRewardRejected ReferralRewardStatus = "rejected"
)

var (
	ErrReferralReviewForbidden = errors.New("referral review requires an independent administrator")
	ErrReferralReviewInvalid   = errors.New("invalid referral review")
	ErrReferralReviewConflict  = errors.New("referral review conflicts with a completed decision")
	ErrReferralIdentityUsed    = errors.New("verified identity has already received a referral reward")
)

func CanReviewReferral(role AccountRole) bool {
	return role == AccountRoleAdmin || role == AccountRoleSuperAdmin
}

type ReferralReview struct {
	status       ReferralRewardStatus
	note         string
	identityHash *string
}

func NewReferralReview(decision, note, identityReference string) (ReferralReview, error) {
	note = strings.TrimSpace(note)
	if len(note) < 10 || len(note) > 2000 {
		return ReferralReview{}, ErrReferralReviewInvalid
	}
	review := ReferralReview{note: note}
	switch decision {
	case "approve":
		reference := strings.ToLower(strings.TrimSpace(identityReference))
		if len(reference) < 8 || len(reference) > 256 {
			return ReferralReview{}, ErrReferralReviewInvalid
		}
		hash := sha256.Sum256([]byte("referral-identity:" + reference))
		value := hex.EncodeToString(hash[:])
		review.identityHash = &value
		review.status = ReferralRewardPaid
	case "reject":
		review.status = ReferralRewardRejected
	default:
		return ReferralReview{}, ErrReferralReviewInvalid
	}
	return review, nil
}
func (r ReferralReview) Status() ReferralRewardStatus { return r.status }
func (r ReferralReview) Note() string                 { return r.note }
func (r ReferralReview) IdentityHash() *string {
	if r.identityHash == nil {
		return nil
	}
	v := *r.identityHash
	return &v
}
