package ports

type TelegramUpdateProcessor interface {
	ProcessUpdate(update any) error
}
