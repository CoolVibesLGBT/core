package news

import (
	"coolvibes/application"
	"coolvibes/helpers"
	"coolvibes/workers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"

	"github.com/mmcdole/gofeed"
)

type ArticleResult struct {
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	Images      []string `json:"images"`
	LocalImages []string `json:"local_images"`
	Categories  []string `json:"categories"`
	Slug        string   `json:"slug"`
	SourceName  string   `json:"source_name"`
	SourceURL   string   `json:"source_url"`
}

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
}

func applyBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	//req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func extractArticle(feedURI string) (*ArticleResult, error) {
	client := &http.Client{
		Timeout: 50 * time.Second,
	}

	parsedURL, err := url.Parse(feedURI)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	applyBrowserHeaders(req)

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RSSReader/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	article, err := readability.FromReader(reader, parsedURL)
	if err != nil {
		return nil, err
	}

	var buf strings.Builder

	err = article.RenderText(&buf)
	if err != nil {
		return nil, err
	}

	var rawImages []string
	if article.Node != nil {
		extractImages(article.Node, &rawImages)
	}

	var images []string
	for _, img := range rawImages {
		full := resolveURL(parsedURL, img)
		if full != "" {
			images = append(images, full)
		}
	}

	return &ArticleResult{
		Text:   buf.String(),
		Images: images,
	}, nil
}

func resolveURL(base *url.URL, raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func MakeSureDirectoryPathExists(path string) error {
	return os.MkdirAll(path, 0755)
}

func downloadImage(imgURL, path string) error {
	client := &http.Client{
		Timeout: 45 * time.Second,
	}

	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return err
	}

	applyBrowserHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func extractImages(n *html.Node, images *[]string) {
	if n.Type == html.ElementNode && n.Data == "img" {
		for _, attr := range n.Attr {
			if attr.Key == "src" {
				*images = append(*images, attr.Val)
			}
			// lazy-load için
			if attr.Key == "data-src" {
				*images = append(*images, attr.Val)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractImages(c, images)
	}
}

func fetchAndProcessRSS(source RSSSource, app *application.App) error {
	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(source.URL)
	if err != nil {
		return err
	}

	// Tüm feed objesini JSON'a çeviriyoruz
	data, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}

	fileName := "./workers/temp/" + source.Name + ".json"
	// Dosyaya yaz
	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		return err
	}

	// Burada istediğin şekilde feed.Items üzerinde işlem yapabilirsin.
	for _, item := range feed.Items {

		articleSlug := helpers.GenerateSlug(item.Title)
		articleFileFolder := "./workers/temp/" + articleSlug + "/"
		err := MakeSureDirectoryPathExists(articleFileFolder)
		if err != nil {
			helpers.Println("MakeSureDirectoryPathExists", err)
		}

		articleContent, err := extractArticle(item.Link)
		if err != nil {
			helpers.Println("extractArticle", err.Error())
			continue
		}

		articleContent.SourceName = source.Name
		articleContent.SourceURL = item.Link

		articleContent.Title = item.Title
		articleContent.Slug = articleSlug
		for _, category := range item.Categories {
			articleContent.Categories = append(articleContent.Categories, category)
		}

		// articleContent zaten ArticleResult
		articleJSON, err := json.MarshalIndent(articleContent, "", "  ")
		if err != nil {
			fmt.Println("articleJSON", err.Error())
			continue
		}

		err = os.WriteFile(articleFileFolder+"article.json", articleJSON, 0644)
		if err != nil {
			fmt.Println("WriteFileArtJSON", err.Error())
			continue
		}

		for i, imgURL := range articleContent.Images {
			if strings.HasPrefix(imgURL, "data:") {
				continue
			}

			parsed, err := url.Parse(imgURL)
			if err != nil {
				helpers.Println("invalid image url:", imgURL, err)
				continue
			}
			ext := filepath.Ext(parsed.Path)
			if ext == "" {
				ext = ".jpg"
			}

			fileName := fmt.Sprintf("img_%d%s", i+1, ext)
			savePath := filepath.Join(articleFileFolder, fileName)

			err = downloadImage(imgURL, savePath)
			if err != nil {
				helpers.Println("failed to download:", imgURL, err.Error())
				continue
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err // ya da logla, handle et
			}

			articleContent.LocalImages = append(articleContent.LocalImages, filepath.Join(cwd, savePath))

		}

		helpers.Println("Kayit Ediliyor : %s", articleContent.Title)
		post, err := CreateNew(articleContent, app)
		if err != nil {
			helpers.Println("CreateNew : %s", err.Error())
			continue
		}

		telegramErr := app.Router.TelegramService.SendNews(post)
		if telegramErr != nil {
			fmt.Println("TELEGRAM ERROR", telegramErr)
			helpers.Println("FETCHER:TelegramService.SendNews %s", telegramErr.Error())
			continue
		}

	}

	return nil
}

func SubmitRSSFetchTasks(dispatcher *workers.Dispatcher, app *application.App) {

	sources := DefaultRSSSources
	for _, src := range sources {
		s := src
		dispatcher.Submit(func() {
			err := fetchAndProcessRSS(s, app)
			if err != nil {
				log.Printf("RSS fetch error for %s: %v", s.Name, err)
			} else {
				log.Printf("RSS fetch success: %s", s.Name)
			}
		})

	}
}
