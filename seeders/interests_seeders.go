package seeders

import (
	payloads "coolvibes/models/user_payloads"
	"coolvibes/models/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedInterests(db *gorm.DB) error {
	// JSON dosyasını aç
	file, err := os.Open("static/data/interests.json")
	if err != nil {
		return fmt.Errorf("Cannot open JSON file: %w", err)
	}
	defer file.Close()

	// JSON yapısını Go struct'larıyla eşliyoruz

	type Group struct {
		Gpb              string `json:"$gpb"`
		GroupID          int    `json:"group_id"`
		Name             string `json:"name"`
		ItemPreviewCount int    `json:"item_preview_count"`
	}

	type Interest struct {
		Gpb         string `json:"$gpb"`
		InterestID  int    `json:"interest_id"`
		Name        string `json:"name"`
		GroupID     int    `json:"group_id"`
		Emoji       string `json:"emoji"`
		HpElementID int    `json:"hp_element_id"`
	}

	type Data struct {
		Groups    []Group    `json:"groups"`
		Interests []Interest `json:"interests"`
	}

	var data Data
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return fmt.Errorf("Cannot decode JSON: %w", err)
	}

	for _, group := range data.Groups {
		fmt.Printf("Group: %s\n", group.Name)

		var interest payloads.Interest
		err := db.Where("name->>'en' = ?", group.Name).First(&interest).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Bulunamadı, ekle
				interest.ID = uuid.New()
				interest.Name = utils.LocalizedString{"en": group.Name}
				if err := db.Create(&interest).Error; err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal(err)
			}
		} else {
			// Bulundu, güncelle
			interest.Name = utils.LocalizedString{"en": group.Name}
			if err := db.Save(&interest).Error; err != nil {
				log.Fatal(err)
			}
		}

		for _, item := range data.Interests {
			if item.GroupID == group.GroupID {
				var interestItem payloads.InterestItem
				err := db.Where("name->>'en' = ? AND interest_id = ?", item.Name, interest.ID).First(&interestItem).Error
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						// Bulunamadı, ekle
						interestItem.ID = uuid.New()
						interestItem.InterestID = interest.ID
						interestItem.Name = utils.LocalizedString{"en": item.Name}
						interestItem.Emoji = item.Emoji
						if err := db.Create(&interestItem).Error; err != nil {
							log.Fatal(err)
						}
					} else {
						log.Fatal(err)
					}
				} else {
					// Bulundu, güncelle
					interestItem.Name = utils.LocalizedString{"en": item.Name}
					interestItem.Emoji = item.Emoji
					if err := db.Save(&interestItem).Error; err != nil {
						log.Fatal(err)
					}
				}
				fmt.Printf(" - %s %s\n", item.Emoji, item.Name)
			}
		}
	}
	return nil
}
