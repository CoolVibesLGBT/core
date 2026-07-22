package usecases

import (
	"context"
	"core/application/ports"
)

type SitemapService struct {
	repository ports.SitemapRepository
}

func NewSitemapService(repository ports.SitemapRepository) *SitemapService {
	return &SitemapService{repository: repository}
}

func (s *SitemapService) Index(baseURL string) (string, error) {
	return s.repository.GenerateSitemapIndex(baseURL)
}

func (s *SitemapService) Posts(ctx context.Context, frontendURL string) ([]byte, error) {
	return s.repository.GeneratePostSitemap(ctx, frontendURL)
}

func (s *SitemapService) News(ctx context.Context, frontendURL string) ([]byte, error) {
	return s.repository.GenerateNewsSitemap(ctx, frontendURL)
}

func (s *SitemapService) Categories(ctx context.Context, frontendURL string) ([]byte, error) {
	return s.repository.GenerateCategoriesSitemap(ctx, frontendURL)
}

func (s *SitemapService) Images(ctx context.Context, frontendURL, apiURL string) ([]byte, error) {
	return s.repository.GenerateImageSitemap(ctx, frontendURL, apiURL)
}

func (s *SitemapService) Videos(ctx context.Context, frontendURL, apiURL string) ([]byte, error) {
	return s.repository.GenerateVideoSitemap(ctx, frontendURL, apiURL)
}
