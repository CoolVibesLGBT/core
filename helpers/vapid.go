package helpers

import (
	"coolvibes/models"
	"coolvibes/push"

	"gorm.io/gorm"
)

func CreateVapidKeys(db *gorm.DB) (*models.VapidKey, error) {
	var key models.VapidKey
	result := db.First(&key)
	if result.Error == nil {
		// Zaten var, döndür
		return &key, nil
	}
	if result.Error != gorm.ErrRecordNotFound {
		// Başka hata varsa onu döndür
		return nil, result.Error
	}

	// Kayıt yoksa yeni üret
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
