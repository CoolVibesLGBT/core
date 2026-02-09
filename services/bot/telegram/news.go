package telegram

import (
	"core/constants"
	"core/helpers"
	"core/models/post"
	"core/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	telegramPackage "gopkg.in/telebot.v4"
)

type Service struct {
	Bot        *telegramPackage.Bot
	TopicStore *TopicStore
}

func New() (*Service, error) {
	pref := telegramPackage.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &telegramPackage.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telegramPackage.NewBot(pref)
	if err != nil {
		return nil, err
	}

	topicStore, err := NewTopicStore("topics.json")
	if err != nil {
		return nil, err
	}

	s := &Service{
		Bot:        bot,
		TopicStore: topicStore,
	}

	s.registerHandlers()

	return s, nil
}

func (s *Service) RegisterWebhook() error {
	if err := s.Bot.RemoveWebhook(); err != nil {
		return err
	}

	err := s.Bot.SetWebhook(&telegramPackage.Webhook{
		Endpoint: &telegramPackage.WebhookEndpoint{
			PublicURL: constants.TELEGRAM_WEBHOOK_PUBLIC_URL,
			//PublicURL: "https://api.coolvibes.lgbt/webhook/bot/telegram/",
		},
		SecretToken: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
	})

	return err
}

func (s *Service) TestMessage() error {
	chat := &telegramPackage.Chat{ID: constants.TELEGRAM_NEWS_GROUP_ID}

	messageText := "Test Message"

	_, err := s.Bot.Send(chat, messageText)
	return err

}

func (s *Service) TestTopic(topicName string) error {
	topic := &telegramPackage.Topic{
		Name:      topicName,
		IconColor: utils.NameToColor(topicName),
	}

	chat := &telegramPackage.Chat{ID: constants.TELEGRAM_NEWS_GROUP_ID}

	createdTopic, err := s.Bot.CreateTopic(chat, topic)
	if err != nil {
		log.Fatalf("Topic oluşturulamadı: %v", err)
	}

	fmt.Printf("Oluşturulan topic ID: %d, İsmi: %s\n", createdTopic.ThreadID, createdTopic.Name)
	return nil
}

func (s *Service) EnsureTopic(chat *telegramPackage.Chat, topicName string) (*TopicData, error) {
	chatID := fmt.Sprint(chat.ID)

	topic := s.TopicStore.GetByName(chatID, helpers.GenerateSlug(topicName))
	if topic != nil {
		log.Printf("Topic bulundu dosyada: %s (ID: %d)", topic.Name, topic.ThreadID)
		return topic, nil
	}

	t := &telegramPackage.Topic{
		Name:      helpers.GenerateSlug(topicName),
		IconColor: utils.NameToColor(topicName),
	}

	createdTopic, err := s.Bot.CreateTopic(chat, t)
	if err != nil {
		return nil, fmt.Errorf("topic oluşturulamadı: %w", err)
	}

	newTopic := TopicData{
		Name:      helpers.GenerateSlug(createdTopic.Name),
		ThreadID:  createdTopic.ThreadID,
		IconColor: t.IconColor,
	}

	// Yeni topic'i dosyaya kaydet
	if err := s.TopicStore.Add(chatID, newTopic); err != nil {
		return nil, fmt.Errorf("topic dosyaya yazılamadı: %w", err)
	}

	log.Printf("Yeni topic oluşturuldu ve kaydedildi: %s (ID: %d)", newTopic.Name, newTopic.ThreadID)
	return &newTopic, nil
}

func (s *Service) SendNews(post *post.Post) error {
	if post == nil {
		return errors.New("SendNews:PostNull")
	}
	chat := &telegramPackage.Chat{ID: constants.TELEGRAM_NEWS_GROUP_ID}

	type PostExtras struct {
		Source struct {
			Name string `json:"source_name"`
			URL  string `json:"source_url"`
		} `json:"source"`
	}

	var extras PostExtras
	_ = json.Unmarshal(post.Extras, &extras)

	//post.Extras.

	title := post.Title.DefaultValue()
	content := post.Summary.DefaultValue()
	if len([]rune(content)) > 512 {
		content = string([]rune(content)[:512]) + "..."
	}
	sourceURL := extras.Source.URL

	hasAttachments := len(post.Attachments) > 0
	hasHastags := len(post.Hashtags) > 0

	//read more
	menu := &telegramPackage.ReplyMarkup{}
	btnJoin := menu.URL("Join Us", "https://t.me/rokethaberler")
	btn := menu.URL("Read More", sourceURL)

	menu.Inline(menu.Row(btn, btnJoin))
	text := fmt.Sprintf("<b>%s</b>\n\n%s", title, content)
	if !hasAttachments {
		if !hasHastags {
			_, err := s.Bot.Send(chat, text, &telegramPackage.SendOptions{
				ParseMode:   telegramPackage.ModeHTML,
				ReplyMarkup: menu,
			})
			return err
		} else {
			tag := post.Hashtags[0]
			topic, err := s.EnsureTopic(chat, tag.Tag)
			if err == nil {

				_, err = s.Bot.Send(chat, text, &telegramPackage.SendOptions{
					ParseMode:   telegramPackage.ModeHTML,
					ReplyMarkup: menu,
					ThreadID:    topic.ThreadID, // Burada topic ID ile topic içinde gönder
				})
			}

		}
	} else {

		rand.Seed(time.Now().UnixNano()) // Rastgelelik için seed ataması

		randomIndex := rand.Intn(len(post.Attachments)) // 0 ile len-1 arasında sayı

		firstAttachment := post.Attachments[randomIndex]
		absolutePath, err := filepath.Abs(firstAttachment.File.StoragePath)

		if !hasHastags {
			if err != nil {
				return err
			}
			photo := &telegramPackage.Photo{
				File:    telegramPackage.FromDisk(absolutePath),
				Caption: text,
			}
			_, err = s.Bot.Send(chat, photo, &telegramPackage.SendOptions{
				ParseMode:   telegramPackage.ModeHTML,
				ReplyMarkup: menu,
			})
			if err != nil {
				fmt.Println("Fotoğraf gönderilemedi:", err)
			}
		} else {
			tag := post.Hashtags[0]
			topic, err := s.EnsureTopic(chat, tag.Tag)
			if err == nil {

				photo := &telegramPackage.Photo{
					File:    telegramPackage.FromDisk(absolutePath),
					Caption: text,
				}

				_, err = s.Bot.Send(chat, photo, &telegramPackage.SendOptions{
					ParseMode:   telegramPackage.ModeHTML,
					ReplyMarkup: menu,
					ThreadID:    topic.ThreadID,
				})

			}

		}

	}

	return nil
}

/*

photo := &tele.Photo{
	File:    tele.FromDisk("images/news.jpg"),
	Caption: "Local news image 🚀",
}

_, err := s.Bot.Send(chat, photo)
*/

func (s *Service) Start() {
	log.Println("Telegram service started")
	s.Bot.Start()
}
