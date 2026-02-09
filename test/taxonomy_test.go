package test

import (
	"testing"

	"core/models/taxonomy"
	"core/models/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPillar_FullLifecycle(t *testing.T) {

	app, err := NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	pillar := taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: "spor",
		Name: utils.LocalizedString{
			"tr": "Spor",
			"en": "Sports",
		},
		IsActive: true,
	}

	err = app.DB.Create(&pillar).Error
	assert.NoError(t, err)

	var found taxonomy.Pillar
	err = app.DB.First(&found, "slug = ?", "spor").Error
	assert.NoError(t, err)

	assert.Equal(t, "spor", found.Slug)
	assert.Equal(t, "Spor", found.Name["tr"])
	assert.True(t, found.IsActive)

	duplicate := taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: "spor",
		Name: utils.LocalizedString{"tr": "Spor2"},
	}

	err = app.DB.Create(&duplicate).Error
	assert.Error(t, err) // unique index çalışmalı

	// 4️⃣ Update
	found.IsActive = false
	err = app.DB.Save(&found).Error
	assert.NoError(t, err)

	var updated taxonomy.Pillar
	app.DB.First(&updated, found.ID)
	assert.False(t, updated.IsActive)

	// 5️⃣ Soft Delete
	err = app.DB.Delete(&updated).Error
	assert.NoError(t, err)

	var count int64
	app.DB.Model(&taxonomy.Pillar{}).Where("id = ?", updated.ID).Count(&count)
	assert.Equal(t, int64(0), count) // soft delete default filtreli

	// 6️⃣ Unscoped kontrol
	app.DB.Unscoped().
		Model(&taxonomy.Pillar{}).
		Where("id = ?", updated.ID).
		Count(&count)

	assert.Equal(t, int64(1), count)
}
