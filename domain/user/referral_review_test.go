package user

import (
	"errors"
	"strings"
	"testing"
)

func TestReferralReviewRequiresEvidenceAndStableIdentity(t *testing.T) {
	for _, test := range []struct{ decision, note, reference string }{
		{"approve", "checked evidence", ""}, {"approve", "short", "trusted-reference"}, {"approve", "checked evidence", strings.Repeat("x", 257)}, {"invalid", "checked evidence", "trusted-reference"}, {"reject", "", ""},
	} {
		if _, err := NewReferralReview(test.decision, test.note, test.reference); !errors.Is(err, ErrReferralReviewInvalid) {
			t.Fatalf("review error %v", err)
		}
	}
	first, err := NewReferralReview("approve", "Checked identity evidence", " Provider:Person-123 ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewReferralReview("approve", "Other review note", "provider:person-123")
	if err != nil {
		t.Fatal(err)
	}
	if first.IdentityHash() == nil || second.IdentityHash() == nil || *first.IdentityHash() != *second.IdentityHash() || len(*first.IdentityHash()) != 64 {
		t.Fatal("identity normalization is unstable")
	}
	copy := first.IdentityHash()
	*copy = "changed"
	if *first.IdentityHash() == "changed" {
		t.Fatal("caller mutated review identity")
	}
	rejected, err := NewReferralReview("reject", "Rejected duplicate account", "")
	if err != nil || rejected.Status() != ReferralRewardRejected || rejected.IdentityHash() != nil {
		t.Fatalf("rejection %+v, %v", rejected, err)
	}
	for _, role := range []AccountRole{AccountRoleUser, AccountRoleModerator, AccountRoleVerified, "unknown"} {
		if CanReviewReferral(role) {
			t.Fatalf("role %q can approve money", role)
		}
	}
	if !CanReviewReferral(AccountRoleAdmin) || !CanReviewReferral(AccountRoleSuperAdmin) {
		t.Fatal("administrator cannot review")
	}
}
