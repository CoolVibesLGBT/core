package places

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"coolvibes/application"
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/repositories"
	services "coolvibes/services/user"
	"coolvibes/utils"
	globalUtils "coolvibes/utils"
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
		heading := utils.MakeHeading([]globalUtils.LexicalText{
			globalUtils.MakeLexicalText(place.Name, true), // bold
		}, "h1")
		children = append(children, heading)
	}

	appendLabelValue := func(label, value string) {
		if value == "" {
			return
		}
		children = append(children, utils.MakeParagraph([]globalUtils.LexicalText{
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

	root := utils.LexicalRoot{
		Children:   children,
		Direction:  nil,
		Format:     "",
		Indent:     0,
		Type:       "root",
		Version:    1,
		TextFormat: 1,
	}

	wrapper := struct {
		Root utils.LexicalRoot `json:"root"`
	}{Root: root}

	return json.MarshalIndent(wrapper, "", "")
}

func SeedPlaces(application *application.App) error {

	var places []Place

	currentDirectory, _ := os.Getwd()
	placesJSONFile := fmt.Sprintf("%s/seeders/places/places.json", currentDirectory)

	file, err := os.Open(placesJSONFile)
	if err != nil {
		log.Fatalf("Dosya açılamadı: %v", err)
	}
	defer file.Close()

	// Dosya içeriğini oku
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Dosya okunamadı: %v", err)
	}

	err = json.Unmarshal(bytes, &places)
	if err != nil {
		log.Fatalf("JSON parse edilemedi: %v", err)
	}

	// Örnek çıktı
	for i, p := range places {
		fmt.Printf("%d: %s (%s), Lat: %f, Lon: %f\n", i+1, p.Name, p.Address, p.Latitude, p.Longitude)
	}

	notificationRepo := repositories.NewNotificationRepository(application.DB, application.SnowFlakeNode)
	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(application.DB)
	userRepo := repositories.NewUserRepository(application.DB, application.SnowFlakeNode, engagementRepo)
	mediaRepo := repositories.NewMediaRepository(application.DB, application.SnowFlakeNode)
	postRepo := repositories.NewPostRepository(application.DB, application.SnowFlakeNode, mediaRepo, userRepo, notificationRepo)
	placesRepo := repositories.NewPlaceRepository(application.DB, application.SnowFlakeNode, mediaRepo, userRepo, notificationRepo, postRepo)
	placeService := services.NewPlaceService(userRepo, postRepo, mediaRepo, placesRepo)

	authUser, err := userRepo.GetUserByNameOrEmailOrNickname(constants.SystemUserExplorer)
	if err != nil {
		fmt.Println("PLACES:AuthUserNotFound", err)
		return err
	}
	fmt.Println("AuthUser", authUser.UserName)

	fmt.Println(placeService.ServiceName())

	type SurveyQuestion struct {
		Question string
		Options  []string
		Optional bool // Yorum gibi isteğe bağlı sorular için
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
		fmt.Println(p.Name)

		lexicalJSON, err := placeToLexical(p)
		if err != nil {
			panic(err)
		}

		/*
			fileName := fmt.Sprintf("lexical_%s.json", strings.ReplaceAll(p.SourceID, " ", "_"))

			err = os.WriteFile(fileName, lexicalJSON, 0644)
			if err != nil {
				panic(err)
			}
		*/

		newP := PlaceDB{
			Name:        p.Name,
			Tag:         p.Tag,
			Address:     p.Address,
			Latitude:    p.Latitude,
			Longitude:   p.Longitude,
			Town:        p.Town,
			Province:    p.Province,
			Region:      p.Region,
			Postcode:    p.Postcode,
			Country:     p.Country,
			CountryCode: p.CountryCode,
			Telephone:   p.Telephone,
			Email:       p.Email,
			Website:     p.Website,
			Description: p.Description,
			URLs:        p.URLs,
			Image:       p.Image,
			Source:      p.Source,
			SourceID:    p.SourceID,
		}

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
			continue // veya return err
		}

		if exists {
			fmt.Println("Place already exists with SourceID:", p.SourceID)
			continue
		}

		authUser.DefaultLanguage = constants.CountryToLanguage[strings.ToUpper(p.CountryCode)]
		placeService.CreatePlace(request, nil, authUser)
	}
	return nil
}
