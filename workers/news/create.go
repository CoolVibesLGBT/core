package news

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/helpers"
	"core/models/post"
	"core/utils"
	"encoding/json"
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

	paragraphs := utils.SplitToParagraphs(article.Text)

	for _, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		textNode := utils.MakeLexicalText(p, false)
		paragraph := utils.MakeParagraph([]utils.LexicalText{textNode})
		children = append(children, paragraph)
	}

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

func CreateNew(article *ArticleResult, dependencies Dependencies) (*post.Post, error) {
	return CreateNewContext(context.Background(), article, dependencies)
}

func CreateNewContext(ctx context.Context, article *ArticleResult, dependencies Dependencies) (*post.Post, error) {
	if err := dependencies.validate(); err != nil {
		return nil, err
	}
	if article == nil {
		return nil, fmt.Errorf("article is required")
	}
	request, err := ArticleToNewsRequest(article)
	if err != nil {
		return nil, err
	}

	authUser, err := dependencies.Users.FetchUserProfileByUsername(constants.SystemUserNews)
	if err != nil {
		helpers.Error("PLACES:AuthUserNotFound", err)
		return nil, err
	}

	for _, img := range article.LocalImages {
		helpers.Println("Image", img)
	}

	files, err := newDiskUploadedFiles(article.LocalImages)
	if err != nil {
		fmt.Println("FilesFromDisk", err)
		return nil, err
	}

	for _, file := range files {
		fmt.Println("FileName:", file.Filename())
	}

	fmt.Println("ImageLen", len(article.LocalImages), "FILELEN", len(files))

	filters := types.Filter{Search: &article.Slug, PostKind: post.PostKindNews}
	isExists, err := dependencies.News.IsNewsExists(filters)
	if err != nil {
		helpers.Println("CheckExistsError", err.Error())
		return nil, err
	}

	articleJSON, err := json.Marshal(article)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal article to JSON: %w", err)
	}

	request["extras[source]"] = []string{string(articleJSON)}
	if !isExists {
		return dependencies.News.CreateNews(ctx, ports.FormData{
			Values: request,
			Files:  files,
		}, authUser)
	} else {
		helpers.Println("AlreadyExists: %s", *filters.Search)
	}

	return nil, nil
}
