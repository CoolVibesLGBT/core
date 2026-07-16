package test

import (
	"testing"

	"core/models/taxonomy"
	"core/models/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCluster_SearchVector(t *testing.T) {
	t.Parallel()

	app := NewTestApp(t)

	tx := app.DB.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()

	// 1️⃣ Pillar (unique slug)
	pillar := taxonomy.Pillar{
		ID:       uuid.New(),
		Slug:     "spor-cluster-test-" + uuid.New().String(),
		Name:     utils.LocalizedString{"tr": "Spor"},
		IsActive: true,
	}

	err := tx.Create(&pillar).Error
	assert.NoError(t, err)

	// 2️⃣ Cluster
	cluster := taxonomy.Cluster{
		ID:       uuid.New(),
		PillarID: pillar.ID,
		Slug:     "spor",
		Name:     utils.LocalizedString{"tr": "Spor"},
		MetaTitle: &utils.LocalizedString{
			"tr": "Futbol Haberleri",
		},
		MetaDescription: &utils.LocalizedString{
			"tr": "En guncel spor haberleri",
		},
		IsActive: true,
	}

	err = tx.Create(&cluster).Error
	assert.NoError(t, err)

	var saved taxonomy.Cluster
	err = tx.First(&saved, "id = ?", cluster.ID).Error
	assert.NoError(t, err)

	// 🔥 SearchVector build mantığına göre beklenen değer
	expected := "spor spor spor spor futbol haberleri en guncel spor haberleri"

	assert.Equal(t, expected, saved.SearchVector)
}
