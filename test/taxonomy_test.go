package test

import (
	"testing"

	"core/models/taxonomy"
	"core/models/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPillar_FullLifecycle(t *testing.T) {

	app := NewTestApp(t)

	slug := "spor-" + uuid.New().String()

	// ---------------------------
	// MAIN LIFECYCLE (TX)
	// ---------------------------
	tx := app.DB.Begin()
	defer tx.Rollback()

	pillar := taxonomy.Pillar{
		ID:       uuid.New(),
		Slug:     slug,
		Name:     utils.LocalizedString{"tr": "Spor", "en": "Sports"},
		IsActive: true,
	}

	assert.NoError(t, tx.Create(&pillar).Error)

	var found taxonomy.Pillar
	assert.NoError(t, tx.First(&found, "slug = ?", slug).Error)

	found.IsActive = false
	assert.NoError(t, tx.Save(&found).Error)

	assert.NoError(t, tx.Delete(&found).Error)

	var count int64
	tx.Model(&taxonomy.Pillar{}).
		Where("id = ?", found.ID).
		Count(&count)

	assert.Equal(t, int64(0), count)

	// ---------------------------
	// DUPLICATE TEST (NO TX)
	// ---------------------------

	uniqueSlug := "dup-" + uuid.New().String()

	original := taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: uniqueSlug,
		Name: utils.LocalizedString{"tr": "Spor"},
	}

	assert.NoError(t, app.DB.Create(&original).Error)

	duplicate := taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: uniqueSlug,
		Name: utils.LocalizedString{"tr": "Spor2"},
	}

	err := app.DB.Create(&duplicate).Error
	assert.Error(t, err)

	// temizle
	app.DB.Delete(&original)
}
