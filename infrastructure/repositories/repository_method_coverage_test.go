package repositories

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

var repositoryTypes = map[string]reflect.Type{
	"ChatRepository":         reflect.TypeOf((*ChatRepository)(nil)),
	"EngagementRepository":   reflect.TypeOf((*EngagementRepository)(nil)),
	"ListingRepository":      reflect.TypeOf((*ListingRepository)(nil)),
	"MatchesRepository":      reflect.TypeOf((*MatchesRepository)(nil)),
	"MediaRepository":        reflect.TypeOf((*MediaRepository)(nil)),
	"ModerationRepository":   reflect.TypeOf((*ModerationRepository)(nil)),
	"NewsRepository":         reflect.TypeOf((*NewsRepository)(nil)),
	"NotificationRepository": reflect.TypeOf((*NotificationRepository)(nil)),
	"PaymentRepository":      reflect.TypeOf((*PaymentRepository)(nil)),
	"PlaceRepository":        reflect.TypeOf((*PlaceRepository)(nil)),
	"PostRepository":         reflect.TypeOf((*PostRepository)(nil)),
	"PrivatePhotoRepository": reflect.TypeOf((*PrivatePhotoRepository)(nil)),
	"SitemapRepository":      reflect.TypeOf((*SitemapRepository)(nil)),
	"SystemRepository":       reflect.TypeOf((*SystemRepository)(nil)),
	"UserRepository":         reflect.TypeOf((*UserRepository)(nil)),
}

// This matrix is intentionally exhaustive for exported repository methods.
// When a repository method is added or removed, this test forces the test
// coverage plan to be updated in the same change.
var repositoryMethodCoverage = map[string]string{
	"ChatRepository": strings.Join([]string{
		"AddMessageToChat", "AddParticipant", "CreateChat", "CreateGroupChat", "CreatePrivateChat", "DB",
		"DeleteChat", "DeleteChatForAll", "DeleteChatForUser", "DeleteChatHistoryForAll", "DeleteChatHistoryForUser",
		"DeleteMessage", "DeleteMessageForAll", "DeleteMessageForUser", "ExpireMessages", "OpenMessage", "GetChatByID", "GetChatByIDWithoutRelations",
		"GetChatsByUserID", "GetChatsByUserIDW", "GetChatsByUserIDWithCursor", "GetMessagesByChatID", "ListChats", "ListChatMessages",
		"GetMessagesByChatIDWithCursor", "GetParticipants", "GetPrivateChatBetweenUsers", "GetUserChatIDsByUserPublicID",
		"MarkChatMessageRead", "Node", "NotifyChatParticipants", "PinMessage", "RemoveParticipant", "SendTypingEvent",
		"UnpinMessage",
	}, " "),
	"EngagementRepository": strings.Join([]string{
		"AddEngagement", "AddTip", "ApplyReciprocalUserInteraction", "CreateEngagementDetail", "DB", "GetEngagement", "GetEngagementDetails",
		"GetEngagementDetailsWithCursor", "GetEngagements", "HasUserEngaged", "ListEngagementDetailsDeprecated",
		"RecordViewOnce", "RemoveEngagementDetail", "ToggleEngagement",
	}, " "),
	"ListingRepository": strings.Join([]string{
		"Create", "GetClassified", "GetJobOffers", "GetJobSearches",
	}, " "),
	"MatchesRepository": strings.Join([]string{
		"GetLikesAfter", "GetMatchesAfter", "GetPassesAfter", "GetUnseenUsers", "RecordView",
	}, " "),
	"MediaRepository": strings.Join([]string{
		"AddMedia", "ClaimNextPendingMedia", "FindMediaAccessPrincipal", "FindMediaFileAccess", "GenerateStoragePath",
		"IsActiveChatParticipant", "MakeSureDirectoryPathExists", "MakeVideoVariant", "Node", "ProcessClaimedMedia",
		"RequeueStaleProcessing", "SaveUploadedFile",
	}, " "),
	"NewsRepository": strings.Join([]string{
		"Categories", "Category", "Create", "DB", "Get", "GetNews", "IsNewsExists", "MediaRepo", "Node",
		"NotificationRepo", "PostRepo", "UserRepo",
	}, " "),
	"NotificationRepository": strings.Join([]string{
		"CreateNotification", "DB", "FetchAndMarkShownNotifications", "GetAllSubscriptions", "MarkNotificationAsRead",
		"Node", "NotifyPrivatePhotoAccessRequested", "NotifyPrivatePhotoAccessResponded", "SendNotificationToUser",
	}, " "),
	"PaymentRepository": strings.Join([]string{
		"Crypto", "DB", "Deposit", "GooglePay", "IBAN", "Node", "ProcessPayment", "Transactions", "Withdraw",
	}, " "),
	"PlaceRepository": strings.Join([]string{
		"Create", "DB", "ExistsBySourceAndPlaceSourceID", "GetNearByPlaces", "GetPlaceByID", "GetPlacesCategories",
		"MediaRepo", "Node", "NotificationRepo", "PostRepo", "UserRepo",
	}, " "),
	"PostRepository": strings.Join([]string{
		"Banana", "Bookmark", "ClusterExists", "CreateCluster", "CreateContentablePost", "CreateEvent", "CreatePillar",
		"CreatePoll", "CreatePost", "CreateSynonym", "DB", "Delete", "Dislike", "ExistsBySlug", "FindClusterBySlug",
		"FetchPublicTimeline", "FetchPublicTimelineVibes", "FetchPublicUserMedia", "FetchPublicUserPostReplies", "FetchPublicUserPosts",
		"FindPostByPublicID", "FindPostsByKind", "FindPublicPostByID", "FindPublicPostByPublicID", "FindPublicPostBySlug",
		"GetCluster", "GetClusters", "GetOrCreateCluster", "GetOrCreatePillar",
		"GetOrCreateSynonym", "GetPillarBySlug", "GetPillars", "GetPillarsWithClusters", "GetPillarsWithClustersWithSlug",
		"GetPostByID", "GetPostByIDEx", "GetPostByIDIncludingUnpublished", "GetPostByIDWithoutRelations", "GetPostByPublicID", "GetPostBySlug",
		"GetPostsByKind", "GetRecentHashtags", "GetSynonym", "GetTimeline", "GetTimelineVibes", "GetUserMedias",
		"GetUserPostReplies", "GetUserPosts", "Like", "Node", "PillarExistsBySlug", "Report", "SearchPublicPosts", "SendNotification", "SetEventRSVP",
		"SynonymExists", "Tip", "View", "Vote",
	}, " "),
	"PrivatePhotoRepository": strings.Join([]string{
		"AddPrivatePhoto", "ArePrivatePhotoUsersBlocked", "CountPrivatePhotos", "DeletePrivatePhoto", "ListProfilePhotos", "MoveProfilePhoto",
		"FindPrivatePhotoAccessByPublicID", "FindPrivatePhotoUserByID", "FindPrivatePhotoUserByPublicID", "GetPrivatePhotoAccess",
		"HasApprovedPrivatePhotoAccess", "ListPrivatePhotoAccessRequests", "ListPrivatePhotos",
		"RequestPrivatePhotoAccess", "RespondPrivatePhotoAccess", "RevokePrivatePhotoAccess",
		"RevokePrivatePhotoAccessBetween",
	}, " "),
	"ModerationRepository": strings.Join([]string{
		"FetchReports", "ResolveReport", "SetPostPublished",
	}, " "),
	"SitemapRepository": strings.Join([]string{
		"BuildNewsSitemap", "BuildPostSitemap", "CountPublishedPosts", "DB", "GenerateCategoriesSitemap",
		"GenerateClusterSitemap", "GenerateImageSitemap", "GenerateNewsSitemap", "GeneratePostSitemap",
		"GenerateSitemapIndex", "GenerateVideoSitemap", "GetLatestNewsPosts", "GetSitemapPosts",
	}, " "),
	"SystemRepository": strings.Join([]string{
		"GetEventKinds", "GetPaymentMethod", "GetPreferences", "GetReportKinds", "GetVapidPublicKey", "SaveVapidSubscription",
	}, " "),
	"UserRepository": strings.Join([]string{
		"AddBalance", "AddReferral", "CheckIn", "Create", "DB", "DeleteUser", "ExistsByEmail", "ExistsByNameOrMail", "ExistsByUsername", "FetchLiveUsers",
		"FetchNearbyUsers", "FetchPublicUserProfile", "FetchUserNotifications", "FindBroadcastUser", "GEOIPDB", "GetByID", "GetByNameOrMailWithoutRelations",
		"GetBySubscriptionSourceID", "GetByUserNameOrEmailOrUsername", "GetEngagementRepository", "GetLocationFromIP",
		"GetPreferences", "GetSessionUserByPublicID", "GetUserByNameOrEmailOrNickname",
		"GetUserByPublicIdWithoutRelations", "GetUserByUUIDdWithoutRelations", "GetUserUUIDByPublicID",
		"GetUsersStartingWith", "Login", "LoginViaToken", "Node", "ResetBotBroadcastPresence", "SearchPublicUsers", "SearchUsers", "TestUser", "UpdateBroadcastState",
		"UpdateLocation", "UpdateUser", "UpdateUserProfile", "Report", "UpdateUserSocket", "UpsertLocation", "UpsertUserPreference",
	}, " "),
}

func TestRepositoryMethodCoverageMatrixIsCurrent(t *testing.T) {
	for repoName, repoType := range repositoryTypes {
		t.Run(repoName, func(t *testing.T) {
			wantRaw, ok := repositoryMethodCoverage[repoName]
			if !ok {
				t.Fatalf("missing coverage entry for %s", repoName)
			}

			got := exportedMethodNames(repoType)
			want := strings.Fields(wantRaw)
			sort.Strings(want)

			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("repository method coverage matrix is stale for %s\nwant: %v\ngot:  %v", repoName, want, got)
			}
		})
	}

	for repoName := range repositoryMethodCoverage {
		if _, ok := repositoryTypes[repoName]; !ok {
			t.Fatalf("coverage entry %s does not match a registered repository type", repoName)
		}
	}
}

func exportedMethodNames(repoType reflect.Type) []string {
	methods := make([]string, 0, repoType.NumMethod())
	for i := 0; i < repoType.NumMethod(); i++ {
		methods = append(methods, repoType.Method(i).Name)
	}
	sort.Strings(methods)
	return methods
}
