package constants

type FollowStatus string
type GenderIdentity string
type UserRole string
type RelationshipStatus string
type BDSMInterest string
type BDSMRole string

type ZodiacSign string

type SmokingHabit string
type DrinkingHabit string

type TravelPurpose string

const (
	ZodiacAries       ZodiacSign = "aries"       // Koç
	ZodiacTaurus      ZodiacSign = "taurus"      // Boğa
	ZodiacGemini      ZodiacSign = "gemini"      // İkizler
	ZodiacCancer      ZodiacSign = "cancer"      // Yengeç
	ZodiacLeo         ZodiacSign = "leo"         // Aslan
	ZodiacVirgo       ZodiacSign = "virgo"       // Başak
	ZodiacLibra       ZodiacSign = "libra"       // Terazi
	ZodiacScorpio     ZodiacSign = "scorpio"     // Akrep
	ZodiacSagittarius ZodiacSign = "sagittarius" // Yay
	ZodiacCapricorn   ZodiacSign = "capricorn"   // Oğlak
	ZodiacAquarius    ZodiacSign = "aquarius"    // Kova
	ZodiacPisces      ZodiacSign = "pisces"      // Balık
	ZodiacUnknown     ZodiacSign = "unknown"     // Bilinmiyor / Belirtilmemiş

	SmokingNever        SmokingHabit = "never"
	SmokingOccasionally SmokingHabit = "occasionally"
	SmokingRegularly    SmokingHabit = "regularly"
	SmokingTryingToQuit SmokingHabit = "trying_to_quit"
	SmokingOther        SmokingHabit = "other"

	DrinkingNever        DrinkingHabit = "never"
	DrinkingOccasionally DrinkingHabit = "occasionally"
	DrinkingRegularly    DrinkingHabit = "regularly"
	DrinkingTryingToQuit DrinkingHabit = "trying_to_quit"
	DrinkingOther        DrinkingHabit = "other"

	UserRoleUser       UserRole = "user"
	UserRoleModerator  UserRole = "moderator"
	UserRoleAdmin      UserRole = "admin"
	UserRoleSuperAdmin UserRole = "super_admin"
	UserRoleBanned     UserRole = "banned"
	UserRoleDeleted    UserRole = "deleted"
	UserRolePending    UserRole = "pending"
	UserRoleVerified   UserRole = "verified"
	UserRoleUnverified UserRole = "unverified"
)

type PrivacyLevel string

const (
	PrivacyPublic        PrivacyLevel = "public"
	PrivacyFriendsOnly   PrivacyLevel = "friends_only"
	PrivacyFollowersOnly PrivacyLevel = "followers_only"
	PrivacyMutualsOnly   PrivacyLevel = "mutuals_only"
	PrivacyPrivate       PrivacyLevel = "private"
)
