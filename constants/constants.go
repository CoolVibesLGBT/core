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

const (
	UserAttributeGenderIdentity     = "gender_identity"    // Gender Identity
	UserAttributeSexualOrientations = "sexual_orientation" // Sexual
	UserAttributeSexRole            = "sex_role"           // Sex Role
	UserAttributeHairColor          = "hair_color"         // Saç rengi
	UserAttributeEyeColor           = "eye_color"          // Göz rengi
	UserAttributeSkinColor          = "skin_color"         // Ten rengi
	UserAttributeBodyType           = "body_type"          // Vücut yapısı
	UserAttributeTattoos            = "tattoos"            // Dovme

	UserAttributeEthnicity               = "ethnicity"                // Etnik köken
	UserAttributeZodiac                  = "zodiac_sign"              // Burç
	UserAttributeCircumcision            = "circumcision"             // Sünnet durumu kategorisi
	UserAttributePhysicalDisability      = "physical_disability"      // Fiziksel engel
	UserAttributeSmoking                 = "smoking"                  // Sigara kullanımı
	UserAttributeDrinking                = "drinking"                 // Alkol kullanımı
	UserAttributeHeight                  = "height"                   // Boy
	UserAttributeWeight                  = "weight"                   // Kilo
	UserAttributeReligion                = "religion"                 // Din
	UserAttributeEducation               = "education"                // Eğitim düzeyi
	UserAttributeRelationshipStatus      = "relationship_status"      // İlişki durumu
	UserAttributeRelationshipPreferences = "relationship_preferences" // iliski tercihleri
	UserAttributePets                    = "pets"                     // Evcil hayvan
	UserAttributePersonality             = "personality"              // Kişilik tipi
	UserAttributePreferredPartnerGender  = "preferred_partner_gender" //Partnerin Tercih Edilen Cinsiyeti
	UserAttributeMBTIType                = "mbti_type"                // MBTI kişilik tipleri, farklı kişilik özelliklerini ve tercihlerini tanımlar
	UserAttributeChronotype              = "cronotype"                //Uyku Turu
	UserAttributeSenseOfHumor            = "sense_of_humor"           // Mizah Anlayisi
	UserAttributeKidsPreference          = "kids_preference"          // Çocuk tercihi
	UserAttributeDietary                 = "dietary"                  // Beslenme diyet
	UserAttributeHIVAIDS                 = "hiv_aids_status"          // HIV / AIDS durumu
	UserAttributeBDSMInterest            = "bdsm_interest"
	UserAttributeBDSMRoles               = "bdsm_roles" // BDSM roller
	UserAttributeBDSMPlays               = "bdsm_plays" // BDSM oyun/aktivite
	UserAttributeInterests               = "interests"  // Gender Identity
	UserAttributeFantasies               = "fantasies"
)

type PrivacyLevel string

const (
	PrivacyPublic        PrivacyLevel = "public"
	PrivacyFriendsOnly   PrivacyLevel = "friends_only"
	PrivacyFollowersOnly PrivacyLevel = "followers_only"
	PrivacyMutualsOnly   PrivacyLevel = "mutuals_only"
	PrivacyPrivate       PrivacyLevel = "private"
)
