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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	telegramPackage "gopkg.in/telebot.v4"
)

type Service struct {
	Bot        *telegramPackage.Bot
	TopicStore *TopicStore

	webhookMode atomic.Bool
}

const telegramHTTPTimeout = 5 * time.Second

var telegramWebhookSecretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)

func botSettings(token string) telegramPackage.Settings {
	return telegramPackage.Settings{
		Token:   token,
		Poller:  &telegramPackage.LongPoller{Timeout: 10 * time.Second},
		Client:  &http.Client{Timeout: telegramHTTPTimeout},
		Offline: true,
	}
}

func New() (*Service, error) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return nil, nil
	}

	// Offline prevents telebot.NewBot from calling getMe. Telegram is an
	// optional integration and must never delay application readiness.
	bot, err := telegramPackage.NewBot(botSettings(token))
	if err != nil {
		helpers.Error("Error:telegramPackage: %s", err.Error())
		return nil, err
	}

	topicStore, err := NewTopicStore("topics.json")
	if err != nil {
		helpers.Error("Error:NewTopicStore : %s", err.Error())
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
	if s == nil || s.Bot == nil {
		return errors.New("Error:RegisterWebhook:Telegram service cannot be started")
	}

	webhookURL := strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_URL"))
	parsedURL, err := url.ParseRequestURI(webhookURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return errors.New("TELEGRAM_WEBHOOK_URL must be a valid https URL")
	}

	secret := strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_SECRET"))
	if !telegramWebhookSecretPattern.MatchString(secret) {
		return errors.New("TELEGRAM_WEBHOOK_SECRET must contain 1-256 letters, digits, underscores, or hyphens")
	}

	err = s.Bot.SetWebhook(&telegramPackage.Webhook{
		Endpoint: &telegramPackage.WebhookEndpoint{
			PublicURL: webhookURL,
		},
		SecretToken: secret,
	})
	if err != nil {
		return err
	}

	s.webhookMode.Store(true)
	return nil
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
		return fmt.Errorf("topic oluşturulamadı: %w", err)
	}

	fmt.Printf("Oluşturulan topic ID: %d, İsmi: %s\n", createdTopic.ThreadID, createdTopic.Name)
	return nil
}

func (s *Service) ProcessUpdate(update any) error {
	if s == nil || s.Bot == nil {
		return errors.New("telegram service is not configured")
	}

	telegramUpdate, ok := update.(telegramPackage.Update)
	if !ok {
		return fmt.Errorf("unsupported telegram update type %T", update)
	}

	s.Bot.ProcessUpdate(telegramUpdate)
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
				_, errBot := s.Bot.Send(chat, text, &telegramPackage.SendOptions{
					ParseMode:   telegramPackage.ModeHTML,
					ReplyMarkup: menu,
					ThreadID:    topic.ThreadID,
				})
				if errBot != nil {
					fmt.Println("Error sending news:", errBot)
				}
			}

		}
	} else {

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
				if err != nil {
					fmt.Println("Fotoğraf gönderilemedi:", err)
				}

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
	if s == nil || s.Bot == nil {
		helpers.Info("Start:Telegram service cannot be started.")
		return
	}
	if s.webhookMode.Load() {
		helpers.Info("Start:Telegram webhook mode enabled; long polling is disabled.")
		return
	}

	s.Bot.Start()
	helpers.Info("Start:Telegram long polling stopped.")
}
