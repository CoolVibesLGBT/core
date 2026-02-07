package telegram

import (
	"fmt"

	tele "gopkg.in/telebot.v4"
)

func (s *Service) registerHandlers() {

	s.Bot.Handle("/start", func(c tele.Context) error {
		fmt.Println("ChatID", c.Chat().ID)
		return c.Send("Welcome")
	})

	s.Bot.Handle("/news", func(c tele.Context) error {
		return c.Send("Latest news coming soon.")
	})

	s.Bot.Handle("/dating", func(c tele.Context) error {
		return c.Send("Dating feature is under development.")
	})
}
