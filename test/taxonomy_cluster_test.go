package test

import (
	"testing"

	"core/models/taxonomy"
	"core/models/utils"

	"github.com/google/uuid"
)

func TestCluster_SearchVector(t *testing.T) {
	app, err := NewTestApp()
	if err != nil {
		t.Fatal(err)
	}

	// 1️⃣ Test için özel pillar oluştur
	pillar := taxonomy.Pillar{
		ID:       uuid.New(),
		Slug:     "spor-cluster-test",
		Name:     utils.LocalizedString{"tr": "Spor"},
		IsActive: true,
	}

	err = app.DB.Create(&pillar).Error
	if err != nil {
		t.Fatalf("pillar create failed: %v", err)
	}

	// 2️⃣ Cluster oluştur
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

	err = app.DB.Create(&cluster).Error
	if err != nil {
		t.Fatalf("cluster create failed: %v", err)
	}

	var saved taxonomy.Cluster
	err = app.DB.First(&saved, "id = ?", cluster.ID).Error
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	expected := "spor spor futbol haberleri en guncel spor haberleri"

	if saved.SearchVector != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, saved.SearchVector)
	}
}
