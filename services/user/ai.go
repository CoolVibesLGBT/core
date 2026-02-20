package services

import (
	"core/mcp"
	"core/repositories"
)

type AIService struct {
	mcpServer *mcp.MCPServer
	mediaRepo *repositories.MediaRepository
	userRepo  *repositories.UserRepository
	postRepo  *repositories.PostRepository
	placeRepo *repositories.PlaceRepository
	newsRepo  *repositories.NewsRepository
}

func NewAIService(
	mcpServer *mcp.MCPServer,
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	placeRepo *repositories.PlaceRepository,
	newsRepo *repositories.NewsRepository) *AIService {
	return &AIService{mcpServer: mcpServer, postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo, newsRepo: newsRepo}
}

func (s *AIService) ServiceName() string {
	return "AIService"
}
func (s *AIService) MCPServer() *mcp.MCPServer {
	return s.mcpServer
}

func (s *AIService) PostRepo() *repositories.PostRepository {
	return s.postRepo
}

func (s *AIService) NewsRepo() *repositories.NewsRepository {
	return s.newsRepo
}
