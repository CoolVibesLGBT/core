package usecases

import (
	"context"
	"core/models"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"
	"core/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPostServiceReadMethodsDelegateFilters(t *testing.T) {
	slug := "hello-world"
	cursor := "cursor-token"
	postRepo := &fakePostRepository{
		getPostBySlug:     &post.Post{ID: uuid.New(), PublicID: 10, Slug: &slug},
		getPostByPublicID: &post.Post{ID: uuid.New(), PublicID: 99},
		timeline:          types.TimelineResult{Posts: []post.Post{{PublicID: 1}}, Cursor: &cursor},
		search:            types.PostsResult{Posts: []post.Post{{PostKind: post.PostKindVideo}}},
		timelineVibes:     types.TimelineResult{Posts: []post.Post{{PostKind: post.PostKindStatus}}},
		recentHashtags:    []types.HashtagStats{{Tag: "go", Count: 2}},
		pillars:           []taxonomy.Pillar{{ID: uuid.New(), Slug: "culture"}},
	}
	userRepo := &fakeUserRepository{}
	service := NewPostService(userRepo, postRepo, &fakeMediaRepository{})

	gotBySlug, err := service.GetPostBySlug(types.Filter{Slug: &slug})
	if err != nil || gotBySlug.Slug == nil || *gotBySlug.Slug != slug {
		t.Fatalf("GetPostBySlug() = %#v, %v", gotBySlug, err)
	}
	if postRepo.getPostBySlugFilter.Slug == nil || *postRepo.getPostBySlugFilter.Slug != slug {
		t.Fatalf("expected slug filter to pass through, got %#v", postRepo.getPostBySlugFilter)
	}

	gotByPublicID, err := service.GetPostByPublicID(99)
	if err != nil || gotByPublicID.PublicID != 99 || postRepo.getPostByPublicIDArg != 99 {
		t.Fatalf("GetPostByPublicID() = %#v, arg=%d, err=%v", gotByPublicID, postRepo.getPostByPublicIDArg, err)
	}

	timeline, err := service.GetTimeline(types.Filter{Limit: 20, PostKind: post.PostKindPost})
	if err != nil || len(timeline.Posts) != 1 || postRepo.timelineFilter.Limit != 20 {
		t.Fatalf("GetTimeline() = %#v, filter=%#v, err=%v", timeline, postRepo.timelineFilter, err)
	}

	search, err := service.SearchPost(types.Filter{Limit: 5, PostKind: post.PostKindVideo})
	if err != nil || len(search.Posts) != 1 || postRepo.searchFilter.PostKind != post.PostKindVideo {
		t.Fatalf("SearchPost() = %#v, filter=%#v, err=%v", search, postRepo.searchFilter, err)
	}

	vibes, err := service.GetTimelineVibes(types.Filter{Limit: 7})
	if err != nil || len(vibes.Posts) != 1 || postRepo.timelineVibesFilter.Limit != 7 {
		t.Fatalf("GetTimelineVibes() = %#v, filter=%#v, err=%v", vibes, postRepo.timelineVibesFilter, err)
	}

	hashtags, err := service.GetRecentHashtags(types.Filter{Limit: 3})
	if err != nil || len(hashtags) != 1 || postRepo.recentHashtagsFilter.Limit != 3 {
		t.Fatalf("GetRecentHashtags() = %#v, filter=%#v, err=%v", hashtags, postRepo.recentHashtagsFilter, err)
	}

	pillars, err := service.GetPillarsWithClusters(types.Filter{Limit: 4})
	if err != nil || len(pillars) != 1 || postRepo.pillarsFilter.Limit != 4 {
		t.Fatalf("GetPillarsWithClusters() = %#v, filter=%#v, err=%v", pillars, postRepo.pillarsFilter, err)
	}
}

func TestPostServiceUserMediaAndNearbyUsersDelegateFilters(t *testing.T) {
	userUUID := uuid.New()
	cursor := int64(1001)
	distance := 12.5
	userRepo := &fakeUserRepository{
		userUUIDByPublicID:  map[int64]uuid.UUID{42: userUUID},
		fetchNearbyUsers:    []*models.User{{ID: userUUID, PublicID: 42}},
		fetchNearbyDistance: &distance,
	}
	postRepo := &fakePostRepository{
		userMedias:       []types.MediaWithUser{{User: models.User{ID: userUUID, PublicID: 42}}},
		userMediasCursor: &cursor,
	}
	service := NewPostService(userRepo, postRepo, &fakeMediaRepository{})

	medias, next, err := service.GetUserMedias(types.Filter{UserID: 42, Limit: 9})
	if err != nil || len(medias) != 1 || next == nil || *next != cursor {
		t.Fatalf("GetUserMedias() = %#v, %v, %v", medias, next, err)
	}
	if postRepo.userMediasFilter.UserUUID != userUUID || postRepo.userMediasFilter.Limit != 9 {
		t.Fatalf("expected media filter to include resolved uuid, got %#v", postRepo.userMediasFilter)
	}

	users, gotDistance, err := service.FetchNearbyUsers(types.Filter{Limit: 8})
	if err != nil || len(users) != 1 || gotDistance == nil || *gotDistance != distance {
		t.Fatalf("FetchNearbyUsers() = %#v, %v, %v", users, gotDistance, err)
	}
	if userRepo.fetchNearbyFilter.Limit != 8 {
		t.Fatalf("expected nearby filter to pass through, got %#v", userRepo.fetchNearbyFilter)
	}
}

func TestPostServiceMutationMethodsDelegateArguments(t *testing.T) {
	postRepo := &fakePostRepository{}
	service := NewPostService(&fakeUserRepository{}, postRepo, &fakeMediaRepository{})
	authUser := &models.User{ID: uuid.New(), PublicID: 50}
	choiceID := uuid.New()
	filter := types.Filter{PostID: 123, AuthUser: authUser}

	if err := service.Vote(context.Background(), choiceID, 2, 3, authUser.ID); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}
	if postRepo.voteChoiceID != choiceID || postRepo.voteWeight != 2 || postRepo.voteRank != 3 || postRepo.voteUserID != authUser.ID {
		t.Fatalf("Vote() did not pass args: %#v", postRepo)
	}
	going := postpayloads.EventAttendanceGoing
	if _, err := service.SetEventRSVP(context.Background(), 123, authUser, &going); err != nil {
		t.Fatalf("SetEventRSVP() error = %v", err)
	}
	if postRepo.eventRSVPPostID != 123 || postRepo.eventRSVPUserID != authUser.ID || postRepo.eventRSVPStatus == nil || *postRepo.eventRSVPStatus != going {
		t.Fatalf("SetEventRSVP() did not pass args: %#v", postRepo)
	}

	if err := service.Dislike(filter); err != nil {
		t.Fatalf("Dislike() error = %v", err)
	}
	if err := service.Banana(filter); err != nil {
		t.Fatalf("Banana() error = %v", err)
	}
	if err := service.Bookmark(filter); err != nil {
		t.Fatalf("Bookmark() error = %v", err)
	}
	if _, err := service.View(filter); err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if postRepo.dislikeFilter.PostID != 123 || postRepo.bananaFilter.PostID != 123 || postRepo.bookmarkFilter.PostID != 123 || postRepo.viewFilter.PostID != 123 {
		t.Fatalf("expected all filters to pass post id 123: dislike=%#v banana=%#v bookmark=%#v view=%#v", postRepo.dislikeFilter, postRepo.bananaFilter, postRepo.bookmarkFilter, postRepo.viewFilter)
	}

	if err := service.Report(context.Background(), 123, "spam", "bad post", authUser); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if postRepo.reportPostID != 123 || postRepo.reportKind != "spam" || postRepo.reportDescription != "bad post" || postRepo.reportAuthUser != authUser {
		t.Fatalf("Report() did not pass args: %#v", postRepo)
	}

	balance, err := service.Tip(context.Background(), 123, authUser, decimal.NewFromInt(4))
	if err != nil || balance == nil || !postRepo.tipAmount.Equal(decimal.NewFromInt(4)) || postRepo.tipAuthUser != authUser {
		t.Fatalf("Tip() balance=%v err=%v repo=%#v", balance, err, postRepo)
	}
}
