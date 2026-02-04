package news

import (
	"coolvibes/application"
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/models/post"
	"coolvibes/utils"
	"fmt"
	"strings"
)

func ArticleToLexical(article ArticleResult) utils.LexicalWrapper {
	var children []utils.LexicalParagraph

	// 1️⃣ Başlık (H1)
	if strings.TrimSpace(article.Title) != "" {
		titleNode := utils.MakeHeading(
			[]utils.LexicalText{
				utils.MakeLexicalText(article.Title, true),
			},
			"h1",
		)
		children = append(children, titleNode)
	}

	// 2️⃣ Metin → paragraflar
	paragraphs := utils.SplitToParagraphs(article.Text)

	for _, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		textNode := utils.MakeLexicalText(p, false)
		paragraph := utils.MakeParagraph([]utils.LexicalText{textNode})
		children = append(children, paragraph)
	}

	// 3️⃣ Kaynak bilgisi (Source) büyük harfli başlık ve URL
	sourceHeading := utils.MakeHeading(
		[]utils.LexicalText{
			utils.MakeLexicalText("SOURCE", true),
		},
		"h2",
	)
	children = append(children, sourceHeading)

	sourceNamePara := utils.MakeParagraph([]utils.LexicalText{
		utils.MakeLexicalText("Source Name: "+article.SourceName, false),
	})
	children = append(children, sourceNamePara)

	sourceURLPara := utils.MakeParagraph([]utils.LexicalText{
		utils.MakeLexicalText("Source URL: "+article.SourceURL, false),
	})
	children = append(children, sourceURLPara)

	// 4️⃣ Root
	root := utils.LexicalRoot{
		Type:       "root",
		Version:    1,
		Children:   children,
		Direction:  nil,
		Format:     "",
		Indent:     0,
		TextFormat: 0,
	}

	return utils.LexicalWrapper{
		Root: root,
	}
}

func ArticleToNewsRequest(article *ArticleResult) (map[string][]string, error) {
	lexArtical := ArticleToLexical(*article)
	lexJSON, err := utils.LexicalToJSON(lexArtical)

	if err != nil {
		return nil, err
	}

	return map[string][]string{
		"title":      {article.Title},
		"slug":       {article.Slug},
		"summary":    {article.Text},
		"hashtags[]": article.Categories,
		"content":    {lexJSON},
		"status":     {"published"},
		"audience":   {"public"},
	}, nil
}

func CreateNew(article *ArticleResult, app *application.App) (*post.Post, error) {
	request, err := ArticleToNewsRequest(article)
	if err != nil {
		return nil, err
	}

	authUser, err := app.Router.UserService.FetchUserProfileByUsername(constants.SystemUserNews)
	if err != nil {
		helpers.Error("PLACES:AuthUserNotFound", err)
		return nil, err
	}

	for _, img := range article.LocalImages {
		helpers.Println("Image", img)
	}

	files, err := utils.FilesFromDisk(article.LocalImages)
	if err != nil {
		fmt.Println("FilesFromDisk", err)
		return nil, err
	}

	for _, file := range files {
		fmt.Println("FileName:", file.Filename)
	}

	fmt.Println("ImageLen", len(article.LocalImages), "FILELEN", len(files))

	return app.Router.NewsService.CreateNews(request, files, authUser)
}
