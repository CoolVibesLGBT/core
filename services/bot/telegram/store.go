package telegram

import (
	"coolvibes/helpers"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

type TopicData struct {
	Name      string `json:"name"`
	ThreadID  int    `json:"thread_id"`
	IconColor int    `json:"icon_color"`
}

type TopicStore struct {
	sync.Mutex
	FilePath string
	Topics   map[string][]TopicData // chatID string => topic listesi
}

func NewTopicStore(filePath string) (*TopicStore, error) {
	store := &TopicStore{
		FilePath: filePath,
		Topics:   make(map[string][]TopicData),
	}
	if err := store.load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Dosya yoksa yeni boş store ile devam et
			return store, nil
		}
		return nil, err
	}
	return store, nil
}

func (s *TopicStore) load() error {
	data, err := os.ReadFile(s.FilePath) // ioutil.ReadFile yerine
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Topics)
}

func (s *TopicStore) save() error {
	data, err := json.MarshalIndent(s.Topics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.FilePath, data, 0644) // ioutil.WriteFile yerine
}

// ChatID bazlı topic listesinden topic bul
func (s *TopicStore) GetByNameLegacy(chatID string, name string) *TopicData {
	s.Lock()
	defer s.Unlock()

	topics, ok := s.Topics[chatID]
	if !ok {
		return nil
	}

	for i := range topics {
		if helpers.GenerateSlug(topics[i].Name) == helpers.GenerateSlug(name) {
			return &topics[i]
		}
	}
	return nil
}

func (s *TopicStore) GetByName(chatID string, name string) *TopicData {
	s.Lock()
	defer s.Unlock()

	topics, ok := s.Topics[chatID]
	if !ok {
		return nil
	}

	slugName := helpers.GenerateSlug(name)

	for i := range topics {
		slugTopicName := helpers.GenerateSlug(topics[i].Name)
		if slugTopicName == slugName ||
			strings.Contains(slugTopicName, slugName) ||
			strings.Contains(slugName, slugTopicName) {
			return &topics[i]
		}
	}
	return nil
}

// ChatID bazlı topic ekle
func (s *TopicStore) Add(chatID string, topic TopicData) error {
	s.Lock()
	defer s.Unlock()

	topics := s.Topics[chatID]
	for _, t := range topics {
		if helpers.GenerateSlug(t.Name) == helpers.GenerateSlug(topic.Name) {
			return nil // zaten var
		}
	}

	s.Topics[chatID] = append(topics, topic)
	return s.save()
}
