package repositories

import (
	"context"
	"core/constants"
	domainpost "core/domain/post"
	"core/models"
	postmodel "core/models/post"
	postpayloads "core/models/post/payloads"
	modelutils "core/models/utils"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestPollVotePolicyAndCountersCommitTogetherIntegration(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if err := db.AutoMigrate(&postpayloads.Poll{}, &postpayloads.PollChoice{}, &postpayloads.PollVote{}); err != nil {
		t.Fatalf("migrate poll vote schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	author := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "poll-author-" + uuid.NewString(), DisplayName: "Poll Author", UserRole: constants.UserRoleUser,
	}
	voter := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "poll-voter-" + uuid.NewString(), DisplayName: "Poll Voter", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{author, voter}).Error; err != nil {
		t.Fatalf("create poll users: %v", err)
	}
	contentableType := postpayloads.ContentablePollPost
	audience := "public"
	publicPost := postmodel.Post{
		ID: uuid.New(), PublicID: basePublicID + 2, AuthorID: author.ID, Domain: models.CoolVibes,
		PostKind: postmodel.PostKindPost, ContentableType: &contentableType, Audience: &audience, Published: true,
	}
	if err := db.Omit(clause.Associations).Create(&publicPost).Error; err != nil {
		t.Fatalf("create poll post: %v", err)
	}

	poll := postpayloads.Poll{
		ID: uuid.New(), ContentableID: publicPost.ID, ContentableType: postpayloads.ContentablePollPost,
		Question: *modelutils.MakeLocalizedString("en", "Choose"), Kind: postpayloads.PollKindMultiple, MaxSelectable: 2,
	}
	choices := []postpayloads.PollChoice{
		{ID: uuid.New(), PollID: poll.ID, Label: *modelutils.MakeLocalizedString("en", "A")},
		{ID: uuid.New(), PollID: poll.ID, Label: *modelutils.MakeLocalizedString("en", "B")},
		{ID: uuid.New(), PollID: poll.ID, Label: *modelutils.MakeLocalizedString("en", "C")},
	}
	if err := db.Omit(clause.Associations).Create(&poll).Error; err != nil {
		t.Fatalf("create poll: %v", err)
	}
	if err := db.Omit(clause.Associations).Create(&choices).Error; err != nil {
		t.Fatalf("create choices: %v", err)
	}

	repository := &PostRepository{db: db}
	for _, choice := range choices[:2] {
		if err := repository.Vote(context.Background(), choice.ID, 1, 0, voter.ID); err != nil {
			t.Fatalf("vote for %s: %v", choice.ID, err)
		}
	}
	if err := repository.Vote(context.Background(), choices[2].ID, 1, 0, voter.ID); !errors.Is(err, domainpost.ErrPollSelectionLimit) {
		t.Fatalf("third selection error = %v; want selection limit", err)
	}

	var voteCount int64
	if err := db.Model(&postpayloads.PollVote{}).
		Joins("JOIN poll_choices ON poll_choices.id = poll_votes.choice_id").
		Where("poll_choices.poll_id = ? AND poll_votes.user_id = ?", poll.ID, voter.ID).
		Count(&voteCount).Error; err != nil {
		t.Fatal(err)
	}
	if voteCount != 2 {
		t.Fatalf("persisted votes = %d; want 2", voteCount)
	}
	assertPollChoiceVoteCount(t, db, choices[0].ID, 1)
	assertPollChoiceVoteCount(t, db, choices[1].ID, 1)
	assertPollChoiceVoteCount(t, db, choices[2].ID, 0)
}

func TestSinglePollVoteReplacesSelectionAndRepairsCounterIntegration(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if err := db.AutoMigrate(&postpayloads.Poll{}, &postpayloads.PollChoice{}, &postpayloads.PollVote{}); err != nil {
		t.Fatalf("migrate poll vote schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	users := []models.User{
		{ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes, UserName: "single-author-" + uuid.NewString(), DisplayName: "Author", UserRole: constants.UserRoleUser},
		{ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes, UserName: "single-voter-" + uuid.NewString(), DisplayName: "Voter", UserRole: constants.UserRoleUser},
	}
	if err := db.Omit(clause.Associations).Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	contentableType, audience := postpayloads.ContentablePollPost, "public"
	publicPost := postmodel.Post{ID: uuid.New(), PublicID: basePublicID + 2, AuthorID: users[0].ID, Domain: models.CoolVibes, PostKind: postmodel.PostKindPost, ContentableType: &contentableType, Audience: &audience, Published: true}
	if err := db.Omit(clause.Associations).Create(&publicPost).Error; err != nil {
		t.Fatal(err)
	}
	poll := postpayloads.Poll{ID: uuid.New(), ContentableID: publicPost.ID, ContentableType: postpayloads.ContentablePollPost, Question: *modelutils.MakeLocalizedString("en", "One"), Kind: postpayloads.PollKindSingle, MaxSelectable: 1}
	choices := []postpayloads.PollChoice{
		{ID: uuid.New(), PollID: poll.ID, Label: *modelutils.MakeLocalizedString("en", "A")},
		{ID: uuid.New(), PollID: poll.ID, Label: *modelutils.MakeLocalizedString("en", "B")},
	}
	if err := db.Omit(clause.Associations).Create(&poll).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Omit(clause.Associations).Create(&choices).Error; err != nil {
		t.Fatal(err)
	}

	repository := &PostRepository{db: db}
	if err := repository.Vote(context.Background(), choices[0].ID, 1, 0, users[1].ID); err != nil {
		t.Fatal(err)
	}
	if err := repository.Vote(context.Background(), choices[1].ID, 1, 0, users[1].ID); err != nil {
		t.Fatal(err)
	}
	assertPollChoiceVoteCount(t, db, choices[0].ID, 0)
	assertPollChoiceVoteCount(t, db, choices[1].ID, 1)

	var selected postpayloads.PollVote
	if err := db.Joins("JOIN poll_choices ON poll_choices.id = poll_votes.choice_id").
		Where("poll_choices.poll_id = ? AND poll_votes.user_id = ?", poll.ID, users[1].ID).
		Take(&selected).Error; err != nil {
		t.Fatal(err)
	}
	if selected.ChoiceID != choices[1].ID {
		t.Fatalf("selected choice = %s; want %s", selected.ChoiceID, choices[1].ID)
	}
}

func assertPollChoiceVoteCount(t *testing.T, db *gorm.DB, choiceID uuid.UUID, expected int) {
	t.Helper()
	var choice postpayloads.PollChoice
	if err := db.Select("id", "vote_count").First(&choice, "id = ?", choiceID).Error; err != nil {
		t.Fatal(err)
	}
	if choice.VoteCount != expected {
		t.Fatalf("choice %s vote_count = %d; want %d", choiceID, choice.VoteCount, expected)
	}
}
