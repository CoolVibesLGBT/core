package lgbt

import (
	"coolvibes/models"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedLGBTPreferences(db *gorm.DB) error {
	_attributes, errAttributes := FetchAttributes()
	_interests, errInterests := FetchInterests()
	_fantasies, errFantasies := FetchFantasies()

	var tag string = "LGBT"

	if errAttributes != nil || errInterests != nil || errFantasies != nil {
		return fmt.Errorf("error fetching preferences data: %v %v %v", errAttributes, errInterests, errFantasies)
	}

	fmt.Println("AttributesLen", len(_attributes))
	fmt.Println("InterestsLen", len(_interests))
	fmt.Println("FantasiesLen", len(_fantasies))

	var bitIndex int64 = 0

	for i := range _attributes {
		for j := range _attributes[i].Items {
			_attributes[i].Items[j].BitIndex = bitIndex
			bitIndex++
		}
	}

	for i := range _interests {
		for j := range _interests[i].Items {
			_interests[i].Items[j].BitIndex = bitIndex
			bitIndex++
		}
	}

	for i := range _fantasies {
		for j := range _fantasies[i].Items {
			_fantasies[i].Items[j].BitIndex = bitIndex
			bitIndex++
		}
	}

	preferencesData := models.PreferencesData{
		Attributes: _attributes,
		Interests:  _interests,
		Fantasies:  _fantasies,
	}

	preferences := models.Preferences{
		ID:       uuid.New(),
		Category: tag,
		Data:     preferencesData,
		BitCount: bitIndex,
	}

	fmt.Println("Bit Count", bitIndex)

	var existing models.Preferences
	err := db.Where("category = ?", preferences.Category).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Kayıt yok, yeni oluştur
			if err := db.Create(&preferences).Error; err != nil {
				return fmt.Errorf("failed to seed preferences: %w", err)
			}
		} else {
			return err
		}
	} else {
		// Kayıt var, güncelle
		preferences.ID = existing.ID // Güncelleme için aynı ID'yi kullan
		if err := db.Save(&preferences).Error; err != nil {
			return fmt.Errorf("failed to update preferences: %w", err)
		}
	}
	fmt.Println("Preferences seeded successfully")

	return nil
}
