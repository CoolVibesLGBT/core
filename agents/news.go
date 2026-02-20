package agents

import (
	"core/constants"
	"core/mcp"
	services "core/services/user"
	"core/types"
	"errors"
)

type NewsAgent struct {
	aiService *services.AIService
}

func NewNewsAgent(aiService *services.AIService) *NewsAgent {
	return &NewsAgent{aiService: aiService}
}

func (a *NewsAgent) Name() string {
	return "news"
}

func (a *NewsAgent) Handle(msg mcp.Envelope) (mcp.Envelope, error) {

	switch msg.Action {

	case constants.CMD_NEWS_CREATE:
		return a.handleCreate(msg)

	case constants.CMD_NEWS_CATEGORIES:
		return a.handleCategories(msg)

	case constants.CMD_NEWS_GET:
		return a.handleGet(msg)

	case constants.CMD_NEWS_FETCH:
		return a.handleList(msg)

	default:
		return mcp.NewMessage(
			a.Name(),
			msg.Source,
			"error",
			"unknown action",
			mcp.TypeError,
		), nil
	}
}

func (a *NewsAgent) handleList(msg mcp.Envelope) (mcp.Envelope, error) {
	return mcp.Envelope{}, errors.New(constants.ErrMethodNotImplemented.String())
	/*
		return mcp.NewMessage(
			"news",
			msg.Source,
			"news.list",
			news,
			mcp.TypeResponse,
		), nil
	*/
}

func (a *NewsAgent) handleGet(msg mcp.Envelope) (mcp.Envelope, error) {

	filters := types.Filter{}
	post, err := a.aiService.NewsRepo().Get(filters)
	if err != nil {
		return mcp.Envelope{}, errors.New(constants.ErrPostNotFound.String())
	}
	return mcp.NewMessage(
		a.Name(),
		msg.Source,
		constants.CMD_NEWS_GET,
		post,
		mcp.TypeResponse,
	), nil
}

func (a *NewsAgent) handleCreate(msg mcp.Envelope) (mcp.Envelope, error) {

	return mcp.Envelope{}, errors.New(constants.ErrMethodNotImplemented.String())
	/*
	   return mcp.NewMessage(

	   	"news",
	   	msg.Source,
	   	"news.created",
	   	news,
	   	mcp.TypeResponse,

	   ), nil
	*/
}

func (a *NewsAgent) handleCategories(msg mcp.Envelope) (mcp.Envelope, error) {
	filters := types.Filter{}
	categories, err := a.aiService.NewsRepo().Categories(filters)
	if err != nil {
		return mcp.Envelope{}, err
	}
	return mcp.NewMessage(
		a.Name(),
		msg.Source,
		constants.CMD_NEWS_CATEGORIES,
		categories,
		mcp.TypeResponse,
	), nil
}

/*
registry := mcp.NewRegistry()
newsAgent := agents.NewNewsAgent(db)

registry.Register(newsAgent)

router := mcp.NewRouter(registry)

msg := mcp.NewMessage(
	"api",
	"news",
	"news.create",
	post.CreateNewsPayload{
		Title:   title,
		Content: content,
		AuthorID: authorID.String(),
		Domain:   "coolvibes.lgbt",
	},
	mcp.TypeCommand,
)

response, err := router.Route(msg)
*/
