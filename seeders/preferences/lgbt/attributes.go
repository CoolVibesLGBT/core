package lgbt

import (
	helpers "coolvibes/helpers"
	utils "coolvibes/models/utils"

	payloads "coolvibes/constants"
	"coolvibes/models"
	"fmt"

	"github.com/google/uuid"
)

func _genderIdentities() models.PreferenceCategory {
	category_title := "Gender Identity"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeGenderIdentity

	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Gender identity refers to a person’s deeply held sense of their own gender how they personally experience themselves as male, female, both, neither, or somewhere along the gender spectrum. It may or may not align with the sex they were assigned at birth, and it is an internal, individual understanding of who they are, rather than how others perceive them."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Male", "tr": "Erkek"},
		{"en": "Female", "tr": "Kadın"},
		{"en": "Agender", "tr": "Agender"},
		{"en": "Androgynous", "tr": "Androjen"},
		{"en": "Aporagender", "tr": "Aporagender"},
		{"en": "Bigender", "tr": "Çift cinsiyetli"},
		{"en": "Demiboy", "tr": "Yarım erkek"},
		{"en": "Demigirl", "tr": "Yarım kız"},
		{"en": "Demigender", "tr": "Demicinsiyet"},
		{"en": "Genderfluid", "tr": "Cinsiyet akışkan"},
		{"en": "Genderflux", "tr": "Genderflux"},
		{"en": "Gender neutral", "tr": "Cinsiyetsiz"},
		{"en": "Gender questioning", "tr": "Cinsiyet sorgulayan"},
		{"en": "Genderqueer", "tr": "Genderqueer"},
		{"en": "Graygender", "tr": "Graygender"},
		{"en": "Hijara", "tr": "Hijara"},
		{"en": "Intergender", "tr": "Intergender"},
		{"en": "Intersex", "tr": "Interseks"},
		{"en": "Maverique", "tr": "Maverique"},
		{"en": "Multigender", "tr": "Çoklu cinsiyet"},
		{"en": "Neutrois", "tr": "Neutrois"},
		{"en": "Non-binary", "tr": "Non-binary"},
		{"en": "Pangender", "tr": "Pangender"},
		{"en": "Polygender", "tr": "Poligender"},
		{"en": "Transfeminine", "tr": "Transfeminen"},
		{"en": "Transmasculine", "tr": "Transmaskülen"},
		{"en": "Transneutral", "tr": "Transneutral"},
		{"en": "Trigender", "tr": "Trigender"},
		{"en": "Two-spirit", "tr": "Two-spirit"},
		{"en": "Waria", "tr": "Waria"},
		{"en": "Xenogender", "tr": "Xenogender"},
		{"en": "Unlabeled", "tr": "Etiketsiz"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _sexualOrientations() models.PreferenceCategory {
	category_title := "Sexual Orientation"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeSexualOrientations
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Sexual orientation describes a person's emotional, romantic, or sexual attraction to others."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Abrosexual", "tr": "Abroseksüel"},
		{"en": "Aceflux", "tr": "Aceflux"},
		{"en": "Androsexual", "tr": "Androseksüel"},
		{"en": "Aroace", "tr": "Aroace"},
		{"en": "Aroflux", "tr": "Aroflux"},
		{"en": "Aromantic", "tr": "Aromantik"},
		{"en": "Asexual", "tr": "Aseksüel"},
		{"en": "A-Spec", "tr": "A-Spec"},
		{"en": "Biromantic", "tr": "Biromantik"},
		{"en": "Bisexual", "tr": "Biseksüel"},
		{"en": "Ceteroromantic", "tr": "Ceteroromantik"},
		{"en": "Ceterosexual", "tr": "Ceteroseksüel"},
		{"en": "Demiromantic", "tr": "Demiromantik"},
		{"en": "Demisexual", "tr": "Demiseksüel"},
		{"en": "Diamoric", "tr": "Diamoric"},
		{"en": "Frayromantic", "tr": "Frayromantik"},
		{"en": "Gay men", "tr": "Gay erkek"},
		{"en": "Gyneromantic", "tr": "Gyneromantik"},
		{"en": "Gynesexual", "tr": "Gyneseksüel"},
		{"en": "Lesbian", "tr": "Lezbiyen"},
		{"en": "Multisexual", "tr": "Multiseksüel"},
		{"en": "Omnisexual", "tr": "Omniseksüel"},
		{"en": "Pansexual", "tr": "Panseksüel"},
		{"en": "Polyamorous", "tr": "Poliamoröz"},
		{"en": "Polyromantic", "tr": "Polyromantik"},
		{"en": "Polysexual", "tr": "Poliseksüel"},
		{"en": "Pomosexual", "tr": "Pomoseksüel"},
		{"en": "Queer", "tr": "Queer"},
		{"en": "Sapphic", "tr": "Sapphic"},
		{"en": "Straight queer", "tr": "Hetero-queer"},
		{"en": "Panromantic", "tr": "Panromantik"},
		{"en": "Omniromantic", "tr": "Omniromantik"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _sexRoles() models.PreferenceCategory {
	category_title := "Sex Role"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeSexRole
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Sexual orientation describes a person's emotional, romantic, or sexual attraction to others."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Top/Active", "tr": "Aktif / Top"},
		{"en": "Bottom/Passive", "tr": "Pasif / Bottom"},
		{"en": "Versatile/Flexible", "tr": "Versatil / Esnek"},
		{"en": "Switch", "tr": "Switch"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _heightAttributes() models.PreferenceCategory {
	category_title := "Height"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeHeight
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Height represents a person's stature measured from base to top."
	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}
	for h := 140; h <= 210; h++ {
		var ls = utils.LocalizedString{
			"en": fmt.Sprintf("%d cm", h),
			"tr": fmt.Sprintf("%d cm", h),
		}
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: h,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _weightAttributes() models.PreferenceCategory {
	category_title := "Weight"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeWeight
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Weight represents a person's body mass or heaviness."
	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}
	for w := 40; w <= 150; w++ {
		var ls = utils.LocalizedString{
			"en": fmt.Sprintf("%d kg", w),
			"tr": fmt.Sprintf("%d kg", w),
		}
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: w,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _hairColors() models.PreferenceCategory {
	category_title := "Hair Color"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeHairColor
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Hair color represents the various natural and dyed colors of a person's hair."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Black", "tr": "Siyah"},
		{"en": "Dark Brown", "tr": "Koyu Kahverengi"},
		{"en": "Brown", "tr": "Kahverengi"},
		{"en": "Light Brown", "tr": "Açık Kahverengi"},
		{"en": "Blonde", "tr": "Sarı"},
		{"en": "Red", "tr": "Kızıl"},
		{"en": "Gray", "tr": "Gri"},
		{"en": "White", "tr": "Beyaz"},
		{"en": "Other", "tr": "Diğer"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _eyeColors() models.PreferenceCategory {
	category_title := "Eye Color"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeEyeColor
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Eye color refers to the color of the iris, which varies between individuals."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Brown", "tr": "Kahverengi"},
		{"en": "Blue", "tr": "Mavi"},
		{"en": "Green", "tr": "Yeşil"},
		{"en": "Hazel", "tr": "Ela"},
		{"en": "Gray", "tr": "Gri"},
		{"en": "Amber", "tr": "Kehribar"},
		{"en": "Other", "tr": "Diğer"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _skinColors() models.PreferenceCategory {
	category_title := "Skin Color"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeSkinColor
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Skin color refers to the natural color of a person’s skin, ranging from very fair to dark tones."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "Very Fair", "tr": "Çok Açık Ten"},
		{"en": "Fair", "tr": "Açık Ten"},
		{"en": "Light", "tr": "Açık Buğday"},
		{"en": "Medium", "tr": "Buğday"},
		{"en": "Olive", "tr": "Zeytin Ten"},
		{"en": "Tan", "tr": "Bronz"},
		{"en": "Brown", "tr": "Esmer"},
		{"en": "Dark Brown", "tr": "Koyu Esmer"},
		{"en": "Black", "tr": "Siyah"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _bodyTypes() models.PreferenceCategory {
	category_title := "Body Type"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeBodyType
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Body type describes the general shape and build of a person's physique."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	bodyTypes := []utils.LocalizedString{
		{"en": "Slim", "tr": "İnce"},
		{"en": "Athletic", "tr": "Atletik"},
		{"en": "Muscular", "tr": "Kaslı"},
		{"en": "Average", "tr": "Orta"},
		{"en": "Chubby", "tr": "Göbekli"},
		{"en": "Heavyset", "tr": "Kilolu"},
	}

	for i, ls := range bodyTypes {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _tattoos() models.PreferenceCategory {
	category_title := "Tattos"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeTattoos
	category_description := "Tattoos indicate whether a person has body art and how extensive it is."

	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	identities := []utils.LocalizedString{
		{"en": "None", "tr": "Yok"},
		{"en": "A few", "tr": "Birkaç tane"},
		{"en": "Many", "tr": "Çok fazla"},
		{"en": "Covered", "tr": "Vücudu kaplıyor"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range identities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _ethnicities() models.PreferenceCategory {
	category_title := "Ethnicity"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeEthnicity
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Ethnicity refers to a person's cultural, regional, or ancestral background."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	ethnicities := []utils.LocalizedString{
		{"en": "White", "tr": "Beyaz"},
		{"en": "Black", "tr": "Siyah"},
		{"en": "Hispanic / Latino", "tr": "Hispanik / Latino"},
		{"en": "Asian", "tr": "Asyalı"},
		{"en": "East Asian", "tr": "Doğu Asyalı"},
		{"en": "South Asian", "tr": "Güney Asyalı"},
		{"en": "Southeast Asian", "tr": "Güneydoğu Asyalı"},
		{"en": "Middle Eastern / North African", "tr": "Orta Doğulu / Kuzey Afrikalı"},
		{"en": "Native American / Indigenous", "tr": "Yerli / Kızılderili"},
		{"en": "Pacific Islander", "tr": "Pasifik Adalı"},
		{"en": "Mixed", "tr": "Melez"},
		{"en": "Other", "tr": "Diğer"},
	}

	for i, ls := range ethnicities {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _zodiacs() models.PreferenceCategory {
	category_title := "Zodiac"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeZodiac
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Zodiac signs represent astrological symbols based on the position of the sun at the time of birth."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	zodiacs := []utils.LocalizedString{
		{"en": "Aries", "tr": "Koç"},
		{"en": "Taurus", "tr": "Boğa"},
		{"en": "Gemini", "tr": "İkizler"},
		{"en": "Cancer", "tr": "Yengeç"},
		{"en": "Leo", "tr": "Aslan"},
		{"en": "Virgo", "tr": "Başak"},
		{"en": "Libra", "tr": "Terazi"},
		{"en": "Scorpio", "tr": "Akrep"},
		{"en": "Sagittarius", "tr": "Yay"},
		{"en": "Capricorn", "tr": "Oğlak"},
		{"en": "Aquarius", "tr": "Kova"},
		{"en": "Pisces", "tr": "Balık"},
	}

	for i, ls := range zodiacs {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _circumcisionStatus() models.PreferenceCategory {
	category_title := "Circumcision Status"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeCircumcision
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Circumcision status indicates whether a person has undergone circumcision or not."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Circumcised", "tr": "Sünnetli"},
		{"en": "Uncircumcised", "tr": "Sünnetsiz"},
		{"en": "Other", "tr": "Diğer"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _disabilities() models.PreferenceCategory {
	category_title := "Physical Disabilities"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributePhysicalDisability
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "This category includes various physical disabilities or conditions that may affect mobility, sensory abilities, or health."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "No disability", "tr": "Hiçbir engelim yok"},
		{"en": "Blind", "tr": "Kör"},
		{"en": "Low vision", "tr": "Az görme"},
		{"en": "Deaf", "tr": "Sağır"},
		{"en": "Hard of hearing", "tr": "İşitme zorluğu"},
		{"en": "Wheelchair user", "tr": "Tekerlekli sandalye kullanıcısı"},
		{"en": "Crutches user", "tr": "Koltuk değneği kullanıcısı"},
		{"en": "Amputee (Missing limb)", "tr": "Ampute (Eksik uzuv)"},
		{"en": "Limited arm function", "tr": "Kısıtlı kol fonksiyonu"},
		{"en": "Limited leg function", "tr": "Kısıtlı bacak fonksiyonu"},
		{"en": "Missing hand", "tr": "Eksik el"},
		{"en": "Missing foot", "tr": "Eksik ayak"},
		{"en": "Chronic illness", "tr": "Kronik hastalık"},
		{"en": "Neurological disorder", "tr": "Nörolojik rahatsızlık"},
		{"en": "Respiratory disorder", "tr": "Solunum rahatsızlığı"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
		{"en": "Other", "tr": "Diğer"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _smoking() models.PreferenceCategory {
	category_title := "Smoking Habits"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeSmoking
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's smoking habits."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Non-smoker", "tr": "Sigara içmiyor"},
		{"en": "Occasionally", "tr": "Ara sıra"},
		{"en": "Regular smoker", "tr": "Düzenli içiyor"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _drinking() models.PreferenceCategory {
	category_title := "Drinking Habits"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeDrinking
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's alcohol consumption habits."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Non-drinker", "tr": "Alkol kullanmıyor"},
		{"en": "Occasionally", "tr": "Ara sıra"},
		{"en": "Social drinker", "tr": "Sosyal içici"},
		{"en": "Regular drinker", "tr": "Düzenli içici"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _relationshipStatuses() models.PreferenceCategory {
	category_title := "Relationship Statuses"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeRelationshipStatus
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's relationship status."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Single", "tr": "Bekar"},
		{"en": "In a Relationship", "tr": "İlişkide"},
		{"en": "Married", "tr": "Evli"},
		{"en": "Partnership", "tr": "Ortaklık"},
		{"en": "In Between", "tr": "Arada"},
		{"en": "I don't know", "tr": "Bilmiyorum"},
		{"en": "Divorced", "tr": "Boşanmış"},
		{"en": "Widowed", "tr": "Dul"},
		{"en": "Separated", "tr": "Ayrı"},
		{"en": "Open", "tr": "Açık"},
		{"en": "Engaged", "tr": "Nişanlı"},
		{"en": "It’s complicated", "tr": "Karmaşık"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _preferredPartnerGenders() models.PreferenceCategory {
	category_title := "Preferred Partner Gender"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributePreferredPartnerGender
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Preferred gender(s) of a person's partner."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	partner_genders := []utils.LocalizedString{
		{"en": "Male", "tr": "Erkek"},
		{"en": "Female", "tr": "Kadın"},
		{"en": "Non-binary", "tr": "İkili olmayan"},
		{"en": "Transgender", "tr": "Transgender"},
		{"en": "Any", "tr": "Herhangi"},
	}

	for i, ls := range partner_genders {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _relationshipPreferences() models.PreferenceCategory {
	category_title := "Relationship Preferences"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeRelationshipPreferences
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Relationship preferences such as monogamous, polyamorous, open relationships, casual dating, or long term relationships."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	preferences := []utils.LocalizedString{
		{"en": "Monogamous", "tr": "Tek eşli"},
		{"en": "Polyamorous", "tr": "Çok eşli"},
		{"en": "Open relationship", "tr": "Açık ilişki"},
		{"en": "Casual dating", "tr": "Gündelik flört"},
		{"en": "Long term relationship", "tr": "Uzun süreli ilişki"},
		{"en": "Situational relationship", "tr": "Durumsal ilişki"},
		{"en": "Friends with benefits", "tr": "Arkadaşlık artı"},
		{"en": "Queerplatonic relationship", "tr": "Queerplatonic ilişki"},
		{"en": "Asexual relationship", "tr": "Aseksüel ilişki"},
		{"en": "Open marriage", "tr": "Açık evlilik"},
		{"en": "Swinging", "tr": "Partner değiş tokuşu"},
		{"en": "Living apart together (LAT)", "tr": "Ayrı yaşayan birliktelik"},
		{"en": "Co-parenting relationship", "tr": "Ortak ebeveynlik"},
		{"en": "Relationship anarchy", "tr": "İlişki anarşisi"},
		{"en": "No relationship", "tr": "İlişki istemiyorum"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range preferences {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _kidsPreferences() models.PreferenceCategory {
	category_title := "Kids Preferences"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeKidsPreference
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's preferences regarding having kids."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	preferences := []utils.LocalizedString{
		{"en": "I’d like them someday", "tr": "Bir gün isterim"},
		{"en": "I’d like them soon", "tr": "Yakında isterim"},
		{"en": "I don’t want kids", "tr": "Çocuk istemiyorum"},
		{"en": "I already have kids", "tr": "Zaten çocuklarım var"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range preferences {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _pets() models.PreferenceCategory {
	category_title := "Pets"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributePets
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's pet preferences or ownership."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	pets := []utils.LocalizedString{
		{"en": "Cat(s)", "tr": "Kedi(ler)"},
		{"en": "Dog(s)", "tr": "Köpek(ler)"},
		{"en": "Both cats and dogs", "tr": "Hem kedi hem köpek"},
		{"en": "Other animals", "tr": "Diğer hayvanlar"},
		{"en": "No pets", "tr": "Hayvan yok"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range pets {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _dietaryPreferences() models.PreferenceCategory {
	category_title := "Dietary Preferences"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeDietary
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's dietary preferences."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	dietary := []utils.LocalizedString{
		{"en": "Vegetarian", "tr": "Vejetaryen"},
		{"en": "Vegan", "tr": "Vegan"},
		{"en": "Pescatarian", "tr": "Balıkçılar"},
		{"en": "Omnivore", "tr": "Her şeyi yiyen"},
		{"en": "Keto", "tr": "Ketojenik"},
		{"en": "Other", "tr": "Diğer"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range dietary {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _educationLevels() models.PreferenceCategory {
	category_title := "Education Levels"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeEducation
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's level of education."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	educationLevels := []utils.LocalizedString{
		{"en": "No formal education", "tr": "Resmî eğitim yok"},
		{"en": "Primary school", "tr": "İlkokul"},
		{"en": "Middle school", "tr": "Ortaokul"},
		{"en": "High school", "tr": "Lise"},
		{"en": "Vocational school", "tr": "Meslek lisesi / Meslek yüksekokulu"},
		{"en": "Undergraduate degree", "tr": "Lisans"},
		{"en": "Graduate degree", "tr": "Yüksek lisans"},
		{"en": "Doctorate / PhD", "tr": "Doktora / PhD"},
		{"en": "In college", "tr": "Üniversite öğrencisi"},
		{"en": "In grad school", "tr": "Yüksek lisans öğrencisi"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range educationLevels {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _personalities() models.PreferenceCategory {
	category_title := "Personality"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributePersonality
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's personality type."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	personalities := []utils.LocalizedString{
		{"en": "Introvert", "tr": "İçe dönük"},
		{"en": "Extrovert", "tr": "Dışa dönük"},
		{"en": "Somewhere in between", "tr": "Arada bir"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range personalities {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _mbtiTypes() models.PreferenceCategory {
	category_title := "Personality Type"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeMBTIType
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "MBTI personality types describe different personality traits and preferences."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	mbti_types := []utils.LocalizedString{
		{"en": "INTJ", "tr": "Analist - Mimar"},
		{"en": "INTP", "tr": "Düşünür - Mantıkçı"},
		{"en": "ENTJ", "tr": "Komutan"},
		{"en": "ENTP", "tr": "Tartışmacı"},
		{"en": "INFJ", "tr": "Danışman"},
		{"en": "INFP", "tr": "Arabulucu"},
		{"en": "ENFJ", "tr": "Düzenleyici"},
		{"en": "ENFP", "tr": "Kampanyacı"},
		{"en": "ISTJ", "tr": "Lojistikçi"},
		{"en": "ISFJ", "tr": "Savunucu"},
		{"en": "ESTJ", "tr": "Yönetici"},
		{"en": "ESFJ", "tr": "Konseye Üye"},
		{"en": "ISTP", "tr": "Zanaatkâr"},
		{"en": "ISFP", "tr": "Macera Avcısı"},
		{"en": "ESTP", "tr": "Girişimci"},
		{"en": "ESFP", "tr": "Eğlendirici"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range mbti_types {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _chronotypes() models.PreferenceCategory {
	category_title := "Chronotype"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeChronotype
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Chronotype describes whether a person is more active in the morning, evening, or night."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	chronotypes := []utils.LocalizedString{
		{"en": "Morning person", "tr": "Sabah insanı"},
		{"en": "Evening person", "tr": "Akşam insanı"},
		{"en": "Night owl", "tr": "Gece kuşu"},
		{"en": "Flexible", "tr": "Esnek"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range chronotypes {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _humorStyles() models.PreferenceCategory {
	category_title := "Sense of Humor"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeSenseOfHumor
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Sense of humor describes the kind of comedy or jokes a person enjoys or uses when communicating."
	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	humortypes := []utils.LocalizedString{
		{"en": "Dark humor", "tr": "Kara mizah"},
		{"en": "Sarcastic", "tr": "İğneleyici"},
		{"en": "Wholesome", "tr": "Tatlı / pozitif"},
		{"en": "Dry / subtle", "tr": "Kuru / ince"},
		{"en": "Goofy", "tr": "Saçma / eğlenceli"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range humortypes {
		slug := helpers.GenerateSlug(ls["en"])
		item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           item_id,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}
	return category
}

func _religions() models.PreferenceCategory {
	category_title := "Religion"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeReligion
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's religious beliefs or affiliations."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	religions := []utils.LocalizedString{
		{"en": "Agnostic", "tr": "Agnostik"},
		{"en": "Atheist", "tr": "Ateist"},
		{"en": "Buddhist", "tr": "Budist"},
		{"en": "Catholic", "tr": "Katolik"},
		{"en": "Christian", "tr": "Hristiyan"},
		{"en": "Hindu", "tr": "Hindu"},
		{"en": "Jain", "tr": "Jain"},
		{"en": "Jewish", "tr": "Yahudi"},
		{"en": "Mormon", "tr": "Mormon"},
		{"en": "Muslim", "tr": "Müslüman"},
		{"en": "Zoroastrian", "tr": "Zerdüşt"},
		{"en": "Sikh", "tr": "Sih"},
		{"en": "Spiritual", "tr": "Spiritüel"},
		{"en": "Baháʼí", "tr": "Bahai"},
		{"en": "Shinto", "tr": "Şinto"},
		{"en": "Taoism", "tr": "Taoizm"},
		{"en": "Confucianism", "tr": "Konfüçyüsçülük"},
		{"en": "Animism", "tr": "Animizm"},
		{"en": "Pagan", "tr": "Pagan"},
		{"en": "Rastafarian", "tr": "Rastafari"},
		{"en": "Indigenous", "tr": "Yerli inançlar"},
		{"en": "Other", "tr": "Diğer"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range religions {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _hivAidsStatuses() models.PreferenceCategory {
	category_title := "HIV/AIDS Status"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeHIVAIDS
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Indicates the user's HIV/AIDS status and related health information."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Negative", "tr": "Negatif"},
		{"en": "Positive", "tr": "Pozitif"},
		{"en": "Undetectable (U=U)", "tr": "Tespit Edilemez (U=U)"},
		{"en": "On treatment (ART)", "tr": "Tedavi altında (ART)"},
		{"en": "Living with AIDS", "tr": "AIDS ile yaşıyor"},
		{"en": "On PrEP (Pre-exposure prophylaxis)", "tr": "PrEP kullanıyor (Koruyucu ilaç)"},
		{"en": "On PEP (Post-exposure prophylaxis)", "tr": "PEP kullanıyor (Maruziyet sonrası koruma)"},
		{"en": "Never tested", "tr": "Hiç test yaptırmadı"},
		{"en": "Recently tested negative", "tr": "Yakın zamanda negatif test sonucu aldı"},
		{"en": "Status unknown", "tr": "Durumu bilinmiyor"},
		{"en": "Prefer not to say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _bdsmInterests() models.PreferenceCategory {
	category_title := "BDSM Interest"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeBDSMInterest
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "User's interest in BDSM activities."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: false,
		Items:         []models.PreferenceItem{},
	}

	statuses := []utils.LocalizedString{
		{"en": "Yes", "tr": "Evet"},
		{"en": "No", "tr": "Hayır"},
		{"en": "Curious", "tr": "Meraklı"},
		{"en": "Experienced", "tr": "Deneyimli"},
		{"en": "Other", "tr": "Diğer"},
	}

	for i, ls := range statuses {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _bdsmRoles() models.PreferenceCategory {
	category_title := "BDSM Roles"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeBDSMRoles
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "User's preferred BDSM roles."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	roles := []utils.LocalizedString{
		{"en": "Dominant", "tr": "Hakim"},
		{"en": "Submissive", "tr": "İtaatkâr / Teslimiyetçi"},
		{"en": "Switch", "tr": "Değişken"},
		{"en": "Top", "tr": "Aktif"},
		{"en": "Bottom", "tr": "Pasif"},
		{"en": "Verse", "tr": "Çift yönlü"},
		{"en": "Vers-top", "tr": "Çift yönlü - Top ağırlıklı"},
		{"en": "Vers-bottom", "tr": "Çift yönlü - Bottom ağırlıklı"},
		{"en": "Side", "tr": "Yan rol"},
		{"en": "Service Top", "tr": "Hizmet Top"},
		{"en": "Service Bottom", "tr": "Hizmet Bottom"},
		{"en": "Sadist", "tr": "Sadist"},
		{"en": "Masochist", "tr": "Mazoşist"},
		{"en": "Rigger", "tr": "Bağlayıcı"},
		{"en": "Rope bunny", "tr": "Halat tavşanı"},
		{"en": "Pet", "tr": "Evcil"},
		{"en": "Caregiver", "tr": "Bakıcı"},
		{"en": "Brat", "tr": "Yaramaz / İtaatsiz"},
		{"en": "Observer", "tr": "Gözlemci / İzleyici"},
		{"en": "Exhibitionist", "tr": "Gösterişçi"},
		{"en": "Other", "tr": "Diğer"},
		{"en": "I'd rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range roles {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func _bdsmPlays() models.PreferenceCategory {
	category_title := "BDSM Plays"
	category_slug := helpers.GenerateSlug(category_title)
	category_tag := payloads.UserAttributeBDSMPlays
	category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
	category_description := "Types of BDSM play activities user is interested in."

	category := models.PreferenceCategory{
		ID:            category_id,
		Tag:           &category_tag,
		Slug:          &category_slug,
		Title:         utils.MakeLocalizedString("en", category_title),
		Description:   utils.MakeLocalizedString("en", category_description),
		Icon:          nil,
		AllowMultiple: true,
		Items:         []models.PreferenceItem{},
	}

	plays := []utils.LocalizedString{
		{"en": "Bondage", "tr": "Bağlama"},
		{"en": "Discipline", "tr": "Disiplin"},
		{"en": "Sadism", "tr": "Sadizm"},
		{"en": "Masochism", "tr": "Mazohizm"},
		{"en": "Role Play", "tr": "Rol yapma"},
		{"en": "Impact Play", "tr": "Fiziksel oyun"},
		{"en": "Sensory Play", "tr": "Duyu oyunları"},
		{"en": "Pet Play", "tr": "Evcil hayvan oyunu"},
		{"en": "Edge Play", "tr": "Riskli oyun"},
		{"en": "Other", "tr": "Diğer"},
		{"en": "I’d rather not say", "tr": "Belirtmek istemiyorum"},
	}

	for i, ls := range plays {
		slug := helpers.GenerateSlug(ls["en"])
		itemID := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug+"-"+slug))
		slugPtr := &slug
		item := models.PreferenceItem{
			ID:           itemID,
			DisplayOrder: i,
			Slug:         slugPtr,
			Title:        &ls,
			Description:  nil,
			Icon:         nil,
			Visible:      true,
		}
		category.Items = append(category.Items, item)
	}

	return category
}

func FetchAttributes() ([]models.PreferenceCategory, error) {
	var attributes []models.PreferenceCategory

	attributes = append(attributes, _genderIdentities())
	attributes = append(attributes, _sexualOrientations())
	attributes = append(attributes, _sexRoles())

	attributes = append(attributes, _relationshipStatuses())
	attributes = append(attributes, _preferredPartnerGenders())
	attributes = append(attributes, _relationshipPreferences())

	attributes = append(attributes, _heightAttributes())
	attributes = append(attributes, _weightAttributes())
	attributes = append(attributes, _hairColors())
	attributes = append(attributes, _eyeColors())
	attributes = append(attributes, _skinColors())

	attributes = append(attributes, _bodyTypes())
	attributes = append(attributes, _tattoos())
	attributes = append(attributes, _ethnicities())

	attributes = append(attributes, _zodiacs())
	attributes = append(attributes, _circumcisionStatus())
	attributes = append(attributes, _disabilities())
	attributes = append(attributes, _smoking())
	attributes = append(attributes, _drinking())

	attributes = append(attributes, _educationLevels())
	attributes = append(attributes, _personalities())
	attributes = append(attributes, _mbtiTypes())

	attributes = append(attributes, _chronotypes())
	attributes = append(attributes, _humorStyles())

	attributes = append(attributes, _kidsPreferences())
	attributes = append(attributes, _pets())
	attributes = append(attributes, _dietaryPreferences())

	attributes = append(attributes, _religions())

	attributes = append(attributes, _hivAidsStatuses())

	attributes = append(attributes, _bdsmInterests())
	attributes = append(attributes, _bdsmPlays())
	attributes = append(attributes, _bdsmRoles())

	for i := range attributes {
		attributes[i].DisplayOrder = i
	}

	return attributes, nil
}
