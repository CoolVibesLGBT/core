package helpers

import (
	"core/infrastructure/push"
	"core/models"

	"gorm.io/gorm"
)

func CreateVapidKeys(db *gorm.DB) (*models.VapidKey, error) {
	var key models.VapidKey
	result := db.First(&key)
	if result.Error == nil {
		return &key, nil
	}
	if result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}
	privateKey, publicKey, err := push.GenerateVAPIDKeys()
	if err != nil {
		return nil, err
	}

	key = models.VapidKey{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	if err := db.Create(&key).Error; err != nil {
		return nil, err
	}

	return &key, nil
}
