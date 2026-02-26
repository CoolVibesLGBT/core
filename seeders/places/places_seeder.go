package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"core/constants"
	"core/helpers"
	"core/models/taxonomy"
	"core/models/utils"
	"core/repositories"
	services "core/services/user"
	globalUtils "core/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Place struct {
	Name        string   `json:"Name"`
	Tag         string   `json:"Tag"`
	Address     string   `json:"Address"`
	Latitude    float64  `json:"Latitude"`
	Longitude   float64  `json:"Longitude"`
	Town        string   `json:"Town"`
	Province    string   `json:"Province"`
	Region      string   `json:"Region"`
	Postcode    string   `json:"Postcode"`
	Country     string   `json:"Country"`
	CountryCode string   `json:"CountryCode"`
	Telephone   string   `json:"Telephone"`
	Email       string   `json:"Email"`
	Website     string   `json:"Website"`
	Description string   `json:"Description"`
	URLs        []string `json:"URLs"`
	Image       string   `json:"Image"`
	Source      string   `json:"Source"`
	SourceID    string   `json:"SourceID"`
}

type PlaceDB struct {
	Name        string   `json:"name"`
	Tag         string   `json:"tag"`
	Address     string   `json:"address"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Town        string   `json:"town"`
	Province    string   `json:"province"`
	Region      string   `json:"region"`
	Postcode    string   `json:"postcode"`
	Country     string   `json:"country"`
	CountryCode string   `json:"country_code"`
	Telephone   string   `json:"telephone"`
	Email       string   `json:"email"`
	Website     string   `json:"website"`
	Description string   `json:"description"`
	URLs        []string `json:"urls"`
	Image       string   `json:"image"`
	Source      string   `json:"source"`
	SourceID    string   `json:"source_id"`
}

func placeToLexical(place Place) ([]byte, error) {
	var children []globalUtils.LexicalParagraph

	// Öncelikle bar ismi h1 olarak
	if place.Name != "" {
		heading := globalUtils.MakeHeading([]globalUtils.LexicalText{
			globalUtils.MakeLexicalText(place.Name, true), // bold
		}, "h1")
		children = append(children, heading)
	}

	appendLabelValue := func(label, value string) {
		if value == "" {
			return
		}
		children = append(children, globalUtils.MakeParagraph([]globalUtils.LexicalText{
			globalUtils.MakeLexicalTextWithFormat(label+": ", 1),
			globalUtils.MakeLexicalText(value, false),
		}))
	}

	appendLabelValue("Name", place.Name)
	appendLabelValue("Description", place.Description)
	appendLabelValue("Tag", place.Tag)
	appendLabelValue("Address", place.Address)
	appendLabelValue("Town", place.Town)
	appendLabelValue("Province", place.Province)
	appendLabelValue("Region", place.Region)
	appendLabelValue("Postcode", place.Postcode)
	appendLabelValue("Country", place.Country)
	appendLabelValue("CountryCode", place.CountryCode)
	appendLabelValue("Telephone", place.Telephone)
	appendLabelValue("Email", place.Email)
	appendLabelValue("Website", place.Website)
	for _, url := range place.URLs {
		appendLabelValue("URL", url)
	}

	// Description, URLs gibi diğer alanlar...

	root := globalUtils.LexicalRoot{
		Children:   children,
		Direction:  nil,
		Format:     "",
		Indent:     0,
		Type:       "root",
		Version:    1,
		TextFormat: 1,
	}

	wrapper := struct {
		Root globalUtils.LexicalRoot `json:"root"`
	}{Root: root}

	return json.MarshalIndent(wrapper, "", "")
}

func SeedPlaces(db *gorm.DB, node *helpers.Node) error {

	notificationRepo := repositories.NewNotificationRepository(db, node)
	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(db)
	userRepo := repositories.NewUserRepository(db, nil, node, engagementRepo)
	mediaRepo := repositories.NewMediaRepository(db, node)
	postRepo := repositories.NewPostRepository(db, node, mediaRepo, userRepo, notificationRepo)
	placesRepo := repositories.NewPlaceRepository(db, node, mediaRepo, userRepo, notificationRepo, postRepo)
	placeService := services.NewPlaceService(userRepo, postRepo, mediaRepo, placesRepo)

	pillarInfo := taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: "places",
		Name: utils.LocalizedString{
			"en": "Places",
			"tr": "Mekanlar",
		},
		Description: &utils.LocalizedString{
			"en": "A comprehensive categorization of venues, locations, and places including restaurants, bars, clubs, gyms, clinics, and more. Perfect for discovering, browsing, or managing different types of establishments.",
			"tr": "Restoranlar, barlar, kulüpler, spor salonları, klinikler ve daha fazlasını kapsayan mekanların kapsamlı bir sınıflandırması. Farklı türdeki yerleri keşfetmek, incelemek veya yönetmek için idealdir.",
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	isPillarExists, err := postRepo.PillarExistsBySlug(context.Background(), pillarInfo.Slug)
	if err != nil {
		return err
	}

	if !isPillarExists {
		postRepo.CreatePillar(context.Background(), &pillarInfo)
	}

	pillarEntry, err := postRepo.GetPillarBySlug(context.Background(), pillarInfo.Slug)
	if err != nil {
		return err
	}

	barClusterInfo := taxonomy.Cluster{
		ID:       uuid.New(),
		PillarID: pillarEntry.ID,
		Name: utils.LocalizedString{
			"en": "Bars",
			"tr": "Barlar",
		},
		Description: &utils.LocalizedString{
			"en": "All types of bars and nightlife venues",
			"tr": "Tüm bar türleri ve gece mekanları",
		},
		Slug:     "bars",
		IsActive: true,
	}

	isBarClusterExists, err := postRepo.ClusterExists(context.Background(), pillarEntry.ID, nil, barClusterInfo.Slug)
	if err != nil {
		return err
	}
	if !isBarClusterExists {
		postRepo.CreateCluster(context.Background(), &barClusterInfo)
	}

	barCluster, err := postRepo.GetCluster(context.Background(), pillarEntry.ID, nil, barClusterInfo.Slug)
	if err != nil {
		return err
	}

	clusters := []taxonomy.Cluster{
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Restaurant",
				"tr": "Restoran",
			},
			Description: &utils.LocalizedString{
				"en": "Restaurants and dining places",
				"tr": "Restoranlar ve yemek mekanları",
			},
			Slug:     "restaurant",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Team",
				"tr": "Takım",
			},
			Description: &utils.LocalizedString{
				"en": "Sports teams and clubs",
				"tr": "Spor takımları ve kulüpler",
			},
			Slug:     "team",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Clinic",
				"tr": "Klinik",
			},
			Description: &utils.LocalizedString{
				"en": "Medical and healthcare clinics",
				"tr": "Tıbbi ve sağlık klinikleri",
			},
			Slug:     "clinic",
			IsActive: true,
		},
		// Bars alt clusterleri
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			ParentID: &barCluster.ID,
			Name: utils.LocalizedString{
				"en": "Mix Bar",
				"tr": "Mix Bar",
			},
			Description: &utils.LocalizedString{
				"en": "Bars and nightlife venues",
				"tr": "Barlar ve gece mekanları",
			},
			Slug:     "mix_bar",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			ParentID: &barCluster.ID,
			Name: utils.LocalizedString{
				"en": "Gay Bar",
				"tr": "Gay Bar",
			},
			Description: &utils.LocalizedString{
				"en": "LGBTQ+ friendly bars and clubs",
				"tr": "LGBTQ+ dostu barlar ve kulüpler",
			},
			Slug:     "gay_bar",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			ParentID: &barCluster.ID,
			Name: utils.LocalizedString{
				"en": "Other Bar",
				"tr": "Diğer Bar",
			},
			Description: &utils.LocalizedString{
				"en": "Other types of bars",
				"tr": "Diğer bar türleri",
			},
			Slug:     "other_bar",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Beauty",
				"tr": "Güzellik",
			},
			Description: &utils.LocalizedString{
				"en": "Beauty salons and services",
				"tr": "Güzellik salonları ve hizmetleri",
			},
			Slug:     "beauty",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Real Estate",
				"tr": "Emlak",
			},
			Description: &utils.LocalizedString{
				"en": "Properties, apartments, and houses",
				"tr": "Mülkler, daireler ve evler",
			},
			Slug:     "real_estate",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Fitness",
				"tr": "Fitness",
			},
			Description: &utils.LocalizedString{
				"en": "Gyms and fitness centers",
				"tr": "Spor salonları ve fitness merkezleri",
			},
			Slug:     "fitness",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "LGBTQ+ Groups",
				"tr": "LGBTQ+ Grupları",
			},
			Description: &utils.LocalizedString{
				"en": "Groups and communities for LGBTQ+",
				"tr": "LGBTQ+ için gruplar ve topluluklar",
			},
			Slug:     "lgbtq_groups",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Others",
				"tr": "Diğer",
			},
			Description: &utils.LocalizedString{
				"en": "Other types of places",
				"tr": "Diğer mekan türleri",
			},
			Slug:     "others",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Massage",
				"tr": "Masaj",
			},
			Description: &utils.LocalizedString{
				"en": "Massage salons and services",
				"tr": "Masaj salonları ve hizmetleri",
			},
			Slug:     "massage",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			PillarID: pillarEntry.ID,
			Name: utils.LocalizedString{
				"en": "Associations",
				"tr": "Dernekler",
			},
			Description: &utils.LocalizedString{
				"en": "Local clubs, associations, and community groups for various interests and activities.",
				"tr": "Yerel kulüpler, dernekler ve çeşitli ilgi alanları ve etkinlikler için topluluk grupları.",
			},
			Slug:     "associations",
			IsActive: true,
		},
	}

	type SynonymSeed struct {
		Slug         string
		Words        utils.LocalizedString
		IsPrimary    bool
		SearchWeight int
	}

	addSynonyms := func(clusterSlug string, seeds []SynonymSeed) error {

		cluster, err := postRepo.GetCluster(context.Background(), pillarEntry.ID, nil, clusterSlug)
		if err != nil {
			return err
		}

		for _, s := range seeds {

			exists, _ := postRepo.SynonymExists(context.Background(), cluster.ID, s.Slug)
			if exists {
				continue
			}

			syn := taxonomy.Synonym{
				ID:           uuid.New(),
				ClusterID:    cluster.ID,
				Word:         s.Words,
				Slug:         s.Slug,
				IsPrimary:    s.IsPrimary,
				SearchWeight: s.SearchWeight,
				CreatedAt:    time.Now(),
			}

			if err := postRepo.CreateSynonym(context.Background(), &syn); err != nil {
				fmt.Println("Synonym create error:", s.Slug, err)
				continue
			}
		}

		return nil
	}

	// =========================
	// RESTAURANT
	// =========================
	addSynonyms("restaurant", []SynonymSeed{
		{"restaurant", utils.LocalizedString{"en": "Restaurant", "tr": "Restoran"}, true, 10},
		{"eatery", utils.LocalizedString{"en": "Eatery", "tr": "Yemek Yeri"}, false, 7},
		{"dining_place", utils.LocalizedString{"en": "Dining Place", "tr": "Yemek Mekanı"}, false, 6},
		{"cafe", utils.LocalizedString{"en": "Cafe", "tr": "Kafe"}, false, 6},
		{"bistro", utils.LocalizedString{"en": "Bistro", "tr": "Bistro"}, false, 5},
		{"lokanta", utils.LocalizedString{"en": "Lokanta", "tr": "Lokanta"}, false, 8},
		{"meyhane", utils.LocalizedString{"en": "Tavern", "tr": "Meyhane"}, false, 6},
		{"food_place", utils.LocalizedString{"en": "Food Place", "tr": "Yemek Yeri"}, false, 4},
	})

	// =========================
	// REAL ESTATE
	// =========================
	addSynonyms("real_estate", []SynonymSeed{
		{"real_estate", utils.LocalizedString{"en": "Real Estate", "tr": "Emlak"}, true, 10},
		{"property", utils.LocalizedString{"en": "Property", "tr": "Mülk"}, false, 8},
		{"apartment", utils.LocalizedString{"en": "Apartment", "tr": "Daire"}, false, 8},
		{"house", utils.LocalizedString{"en": "House", "tr": "Ev"}, false, 7},
		{"flat", utils.LocalizedString{"en": "Flat", "tr": "Daire"}, false, 6},
		{"daily_rental", utils.LocalizedString{"en": "Daily Rental", "tr": "Günlük Kiralık"}, false, 9},
		{"short_term_rent", utils.LocalizedString{"en": "Short Term Rent", "tr": "Kısa Dönem Kiralama"}, false, 7},
		{"roommate", utils.LocalizedString{"en": "Roommate", "tr": "Oda Arkadaşı"}, false, 9},
		{"housemate", utils.LocalizedString{"en": "Housemate", "tr": "Ev Arkadaşı"}, false, 9},
		{"shared_flat", utils.LocalizedString{"en": "Shared Flat", "tr": "Paylaşımlı Ev"}, false, 8},
		{"rental_listing", utils.LocalizedString{"en": "Rental Listing", "tr": "Kiralık İlan"}, false, 6},
	})

	// =========================
	// FITNESS
	// =========================
	addSynonyms("fitness", []SynonymSeed{
		{"fitness", utils.LocalizedString{"en": "Fitness", "tr": "Fitness"}, true, 10},
		{"gym", utils.LocalizedString{"en": "Gym", "tr": "Spor Salonu"}, false, 9},
		{"workout_center", utils.LocalizedString{"en": "Workout Center", "tr": "Antrenman Merkezi"}, false, 6},
		{"pilates", utils.LocalizedString{"en": "Pilates Studio", "tr": "Pilates Salonu"}, false, 7},
		{"crossfit", utils.LocalizedString{"en": "Crossfit", "tr": "Crossfit"}, false, 6},
		{"bodybuilding", utils.LocalizedString{"en": "Bodybuilding", "tr": "Vücut Geliştirme"}, false, 5},
	})

	// =========================
	// BEAUTY
	// =========================
	addSynonyms("beauty", []SynonymSeed{
		{"beauty", utils.LocalizedString{"en": "Beauty", "tr": "Güzellik"}, true, 10},
		{"beauty_salon", utils.LocalizedString{"en": "Beauty Salon", "tr": "Güzellik Salonu"}, false, 9},
		{"hair_salon", utils.LocalizedString{"en": "Hair Salon", "tr": "Kuaför"}, false, 8},
		{"barbershop", utils.LocalizedString{"en": "Barbershop", "tr": "Berber"}, false, 7},
		{"nail_salon", utils.LocalizedString{"en": "Nail Salon", "tr": "Tırnak Salonu"}, false, 6},
		{"spa", utils.LocalizedString{"en": "Spa", "tr": "Spa"}, false, 6},
	})

	// =========================
	// MASSAGE
	// =========================
	addSynonyms("massage", []SynonymSeed{
		{"massage", utils.LocalizedString{"en": "Massage", "tr": "Masaj"}, true, 10},
		{"massage_salon", utils.LocalizedString{"en": "Massage Salon", "tr": "Masaj Salonu"}, false, 9},
		{"therapy_massage", utils.LocalizedString{"en": "Therapy Massage", "tr": "Terapötik Masaj"}, false, 7},
		{"thai_massage", utils.LocalizedString{"en": "Thai Massage", "tr": "Thai Masajı"}, false, 6},
		{"relaxation_center", utils.LocalizedString{"en": "Relaxation Center", "tr": "Rahatlama Merkezi"}, false, 5},
	})

	// =========================
	// LGBTQ GROUPS
	// =========================
	addSynonyms("lgbtq_groups", []SynonymSeed{
		{"lgbtq_groups", utils.LocalizedString{"en": "LGBTQ+ Groups", "tr": "LGBTQ+ Grupları"}, true, 10},
		{"community_group", utils.LocalizedString{"en": "Community Group", "tr": "Topluluk Grubu"}, false, 7},
		{"support_group", utils.LocalizedString{"en": "Support Group", "tr": "Destek Grubu"}, false, 8},
		{"ngo", utils.LocalizedString{"en": "NGO", "tr": "Sivil Toplum Kuruluşu"}, false, 6},
		{"association", utils.LocalizedString{"en": "Association", "tr": "Dernek"}, false, 6},
	})

	for _, c := range clusters {
		exists, err := postRepo.ClusterExists(context.Background(), pillarEntry.ID, nil, c.Slug)
		if err != nil {
			fmt.Println("Error checking cluster:", c.Slug, err)
			continue
		}

		if !exists {
			c.PillarID = pillarEntry.ID
			c.ID = uuid.New()
			if err := postRepo.CreateCluster(context.Background(), &c); err != nil {
				fmt.Println("Error creating cluster:", c.Slug, err)
				continue
			}
			fmt.Println("Created cluster:", c.Slug)
		} else {
			fmt.Println("Cluster already exists:", c.Slug)
		}
	}

	var places []Place

	currentDirectory, _ := os.Getwd()
	placesJSONFile := fmt.Sprintf("%s/seeders/places/places.json", currentDirectory)

	file, err := os.Open(placesJSONFile)
	if err != nil {
		log.Fatalf("Dosya açılamadı: %v", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Dosya içeriğini oku
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Dosya okunamadı: %v", err)
	}

	err = json.Unmarshal(bytes, &places)
	if err != nil {
		log.Fatalf("JSON parse edilemedi: %v", err)
	}

	authUser, err := userRepo.GetUserByNameOrEmailOrNickname(constants.SystemUserExplorer)
	if err != nil {
		fmt.Println("PLACES:AuthUserNotFound", err)
		return err
	}
	fmt.Println("AuthUser", authUser.UserName)

	type SurveyQuestion struct {
		Question string
		Options  []string
		Optional bool
	}

	var LGBTPlaceSurveyQuestions = []SurveyQuestion{
		// Temel sorular
		{
			Question: "What is the overall rating of the place?",
			Options:  []string{"Very bad", "Bad", "Average", "Good", "Very good"},
			Optional: false,
		},
		{
			Question: "Is the place LGBT friendly?",
			Options:  []string{"Yes", "No"},
			Optional: false,
		},
		{
			Question: "Is the staff respectful and inclusive?",
			Options:  []string{"Yes", "No", "Somewhat"},
			Optional: false,
		},
		{
			Question: "Is the place safe for LGBT individuals?",
			Options:  []string{"Yes", "No", "Not sure"},
			Optional: false,
		},
		{
			Question: "Is the place accessible for disabled people?",
			Options:  []string{"Yes", "No"},
			Optional: false,
		},
		{
			Question: "Is the place visually appealing and comfortable?",
			Options:  []string{"Yes", "No"},
			Optional: false,
		},
		{
			Question: "Is the place easy to reach by public transportation?",
			Options:  []string{"Yes", "No", "Not applicable"},
			Optional: false,
		},
		{
			Question: "Are the restrooms gender-neutral or inclusive?",
			Options:  []string{"Yes", "No", "Not sure"},
			Optional: false,
		},
		{
			Question: "Is the pricing of products/services reasonable?",
			Options:  []string{"Yes", "No"},
			Optional: false,
		},
		{
			Question: "How is the overall crowd and atmosphere?",
			Options: []string{
				"Very welcoming",
				"Somewhat welcoming",
				"Neutral",
				"Somewhat unwelcoming",
				"Unwelcoming",
			},
			Optional: false,
		},
		{
			Question: "Would you recommend this place to other LGBT individuals?",
			Options:  []string{"Definitely", "Maybe", "No"},
			Optional: false,
		},
		{
			Question: "Are the staff knowledgeable about LGBT issues?",
			Options:  []string{"Yes", "No", "Somewhat"},
			Optional: false,
		},
		{
			Question: "Does the place openly advertise LGBT-friendly policies or events?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Are there events specifically targeted at the LGBT community?",
			Options:  []string{"Yes", "No", "Sometimes"},
			Optional: false,
		},
		{
			Question: "Are facilities and services accommodating for transgender individuals?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Is there a positive environment regarding gender diversity?",
			Options:  []string{"Yes", "No", "Somewhat"},
			Optional: false,
		},
		{
			Question: "Have you experienced or witnessed discrimination or negative behavior here?",
			Options:  []string{"Never", "Rarely", "Often", "Very often"},
			Optional: false,
		},
		{
			Question: "Do you feel comfortable being yourself in this place?",
			Options:  []string{"Definitely yes", "Yes", "Neutral", "No", "Definitely no"},
			Optional: false,
		},
		{
			Question: "Does the clientele support and respect the LGBT community?",
			Options:  []string{"Yes", "No", "Somewhat"},
			Optional: false,
		},
		{
			Question: "How well does the staff respect privacy and confidentiality?",
			Options:  []string{"Excellent", "Good", "Fair", "Poor", "Very poor"},
			Optional: false,
		},
		{
			Question: "Is the physical environment (lighting, decor, seating) welcoming to LGBT people?",
			Options:  []string{"Yes", "No", "Neutral"},
			Optional: false,
		},
		{
			Question: "How clean and hygienic is the place?",
			Options:  []string{"Very clean", "Clean", "Average", "Dirty", "Very dirty"},
			Optional: false,
		},
		{
			Question: "How would you rate the music and entertainment quality?",
			Options:  []string{"Excellent", "Good", "Average", "Poor", "Very poor"},
			Optional: false,
		},
		{
			Question: "Are food and beverage options inclusive and satisfactory?",
			Options:  []string{"Yes", "No", "Not applicable"},
			Optional: false,
		},
		{
			Question: "Are the place policies (e.g. age restrictions, dress code) clear and fair?",
			Options:  []string{"Yes", "No", "Not sure"},
			Optional: false,
		},
		{
			Question: "Does the place support LGBT community in social responsibility projects?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Is the staff communication inclusive and respectful?",
			Options:  []string{"Yes", "No", "Sometimes"},
			Optional: false,
		},
		{
			Question: "Are services (e.g. reservations, private events) suitable for LGBT individuals?",
			Options:  []string{"Yes", "No", "Partially"},
			Optional: false,
		},
		{
			Question: "Does the place have LGBT-themed decorations or symbols?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Is the place suitable for families with children?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Does the place’s social media share positive content for LGBT individuals?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Is freedom of gender identity and expression encouraged here?",
			Options:  []string{"Yes", "No", "Partially"},
			Optional: false,
		},
		{
			Question: "Is the overall atmosphere relaxed and friendly?",
			Options:  []string{"Yes", "No", "Neutral"},
			Optional: false,
		},
		{
			Question: "Are there clear policies preventing discrimination based on sexual orientation or gender identity?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
		{
			Question: "Does the place offer support groups or resources for LGBTQ+ individuals?",
			Options:  []string{"Yes", "No", "Don't know"},
			Optional: false,
		},
	}

	for _, p := range places {
		lexicalJSON, err := placeToLexical(p)
		if err != nil {
			panic(err)
		}

		newP := PlaceDB(p)

		jsonPlaceBytes, _ := json.Marshal(newP)
		jsonPlaceString := string(jsonPlaceBytes)

		request := map[string][]string{
			"title":                  {p.Name},
			"slug":                   {helpers.GenerateSlug(p.Name)},
			"content":                {string(lexicalJSON)},
			"audience":               {"public"},
			"hashtags[]":             {"lgbtiqa", "lgbtcommunity", "queer", "pride", "loveislove", "lgbt", "lgbtlife", "lgbtfriendly", "lgbtally", "lgbtvisibility", "bar", p.Tag},
			"location[address]":      {p.Address},
			"location[lat]":          {fmt.Sprintf("%f", p.Latitude)},
			"location[lng]":          {fmt.Sprintf("%f", p.Longitude)},
			"location[town]":         {p.Town},
			"location[province]":     {p.Province},
			"location[region]":       {p.Region},
			"location[postcode]":     {p.Postcode},
			"location[country]":      {p.Country},
			"location[country_code]": {p.CountryCode},
			"location[telephone]":    {p.Telephone},
			"location[email]":        {p.Email},
			"location[website]":      {p.Website},
			"extras[place]":          {jsonPlaceString},
		}

		for i, question := range LGBTPlaceSurveyQuestions {
			prefix := fmt.Sprintf("polls[%d]", i)
			request[fmt.Sprintf("%s.question", prefix)] = []string{question.Question}
			request[fmt.Sprintf("%s.kind", prefix)] = []string{"single"}
			request[fmt.Sprintf("%s.duration", prefix)] = []string{"0"}
			request[fmt.Sprintf("%s.max_selectable", prefix)] = []string{"1"}

			if question.Options != nil {
				for j, option := range question.Options {
					request[fmt.Sprintf("%s.options[%d]", prefix, j)] = []string{option}
				}
			}
		}

		exists, err := placesRepo.ExistsBySourceAndPlaceSourceID(p.Source, p.SourceID)
		if err != nil {
			log.Println("exists check failed:", err)
			continue
		}

		if exists {
			fmt.Println("Place already exists with SourceID:", p.SourceID)
			continue
		}

		authUser.DefaultLanguage = constants.CountryToLanguage[strings.ToUpper(p.CountryCode)]
		place, err := placeService.CreatePlace(context.Background(), request, nil, authUser)
		if err != nil {
			return err
		}

		postCluster, err := postRepo.GetCluster(context.Background(), pillarEntry.ID, nil, p.Tag)

		if err != nil {
			fmt.Println("cluster not found:", p.Tag, err)
			return err
		}

		err = db.Model(place).Association("Clusters").Append(postCluster)
		if err != nil {
			fmt.Println("failed to attach cluster:", err)
		}

	}
	return nil
}
