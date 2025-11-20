package faker

import (
	"fmt"
	"time"

	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/utils"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateUser(db *gorm.DB, snowFlakeNode *helpers.Node) models.User {
	gofakeit.Seed(time.Now().UnixNano())

	// Opsiyonel pointer alanlar için helper fonksiyon
	stringPtr := func(s string) *string {
		return &s
	}

	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Rastgele doğum tarihi (18-60 yaş arası)
	password := "denemetest"
	hash, err := helpers.HashPasswordArgon2id(password)
	if err != nil {
		fmt.Println("failed to create hash password: %w", err)
	}

	dob := gofakeit.DateRange(time.Now().AddDate(-60, 0, 0), time.Now().AddDate(-18, 0, 0))

	user := models.User{
		ID:          uuid.New(),
		PublicID:    snowFlakeNode.Generate().Int64(),
		UserName:    gofakeit.Username(),
		DisplayName: gofakeit.Name(),
		Email:       gofakeit.Email(),
		Password:    hash,
		SocketID:    stringPtr(gofakeit.UUID()),
		Bio:         utils.MakeLocalizedString("en", gofakeit.Sentence(10)),
		DateOfBirth: timePtr(dob),

		// Burada ilişkisel alanları elle doldurabilirsin veya boş bırakabilirsin.

		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		LastOnline: timePtr(time.Now().Add(-time.Hour * time.Duration(gofakeit.Number(1, 72)))),

		Location:        nil,
		DefaultLanguage: "en",

		AvatarID: nil,
		Avatar:   nil,

		CoverID: nil,
		Cover:   nil,

		Stories: nil,

		Languages:     []string{"en", "tr"},
		Hobbies:       []string{"reading", "gaming"},
		MoviesGenres:  []string{"action", "comedy"},
		TVShowsGenres: []string{"drama", "thriller"},
		TheaterGenres: []string{"musical"},
		CinemaGenres:  []string{"sci-fi"},
		ArtInterests:  []string{"painting", "sculpture"},
		Entertainment: []string{"music", "concerts"},

		Travel:          models.TravelData{},      // Gerekirse doldurabilirsin
		SocialRelations: models.SocialRelations{}, // Aynı şekilde

		// jwt.StandardClaims içindeki alanları istersen ekle ya da boş bırak
	}

	db.Create(&user)
	return user
}
func FakeUser(db *gorm.DB, snowFlakeNode *helpers.Node) {
	for i := 0; i < 2; i++ {
		CreateUser(db, snowFlakeNode)

	}
}
