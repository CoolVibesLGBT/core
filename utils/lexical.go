package utils

import (
	"encoding/json"
	"strings"
)

// Lexical text node
type LexicalText struct {
	Detail  int    `json:"detail"`
	Format  int    `json:"format"`
	Mode    string `json:"mode"`
	Style   string `json:"style"`
	Text    string `json:"text"`
	Type    string `json:"type"`
	Version int    `json:"version"`
	Bold    bool   `json:"bold,omitempty"`
}

// Lexical paragraph node
type LexicalParagraph struct {
	Children   []LexicalText `json:"children"`
	Direction  interface{}   `json:"direction"` // null
	Format     string        `json:"format"`
	Indent     int           `json:"indent"`
	Type       string        `json:"type"`
	Version    int           `json:"version"`
	TextFormat int           `json:"textFormat"`
	TextStyle  string        `json:"textStyle"`
	Tag        string        `json:"tag,omitempty"` // Bu satırı ekle
}

// Lexical root and editor state
type LexicalRoot struct {
	Children   []LexicalParagraph `json:"children"`
	Direction  interface{}        `json:"direction"`
	Format     string             `json:"format"`
	Indent     int                `json:"indent"`
	Type       string             `json:"type"`
	Version    int                `json:"version"`
	TextFormat int                `json:"textFormat"` // Bu satırı ekle
}

type LexicalWrapper struct {
	Root LexicalRoot `json:"root"`
}

// Yardımcı fonksiyon: Lexical metin düğümü oluştur
func makeLexicalText(text string, bold bool) LexicalText {
	return LexicalText{
		Detail:  0,
		Format:  0,
		Mode:    "normal",
		Style:   "",
		Text:    text,
		Type:    "text",
		Version: 1,
		Bold:    bold,
	}
}

func MakeLexicalText(text string, bold bool) LexicalText {
	return makeLexicalText(text, bold)
}

func MakeHeading(children []LexicalText, tag string) LexicalParagraph {
	return LexicalParagraph{
		Type:       "heading",
		Tag:        tag,
		Version:    1,
		TextFormat: 1,
		Children:   children,
		Direction:  nil,
		Format:     "",
		Indent:     0,
		TextStyle:  "",
	}
}

func MakeLexicalTextWithFormat(text string, format int) LexicalText {
	return LexicalText{
		Detail:  0,
		Format:  format, // 1, 3, 7, 11 gibi
		Mode:    "normal",
		Style:   "",
		Text:    text,
		Type:    "text",
		Version: 1,
	}
}

// Yardımcı fonksiyon: Paragraf oluştur
func makeParagraph(children []LexicalText) LexicalParagraph {
	return LexicalParagraph{
		Children:   children,
		Direction:  nil,
		Format:     "",
		Indent:     0,
		Type:       "paragraph",
		Version:    1,
		TextFormat: 0,
		TextStyle:  "",
	}
}

func MakeParagraph(children []LexicalText) LexicalParagraph {
	return makeParagraph(children)
}

// Açıklamayı cümlelere böl (basitçe noktalama işaretlerine göre)
func splitDescription(desc string) []string {
	separators := []string{".", "!", "?"}
	for _, sep := range separators {
		desc = strings.ReplaceAll(desc, sep, sep+"|")
	}
	parts := strings.Split(desc, "|")
	var cleaned []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func SplitDescription(desc string) []string {
	return splitDescription(desc)
}

func splitToParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")

	var out []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func SplitToParagraphs(text string) []string {
	return splitToParagraphs(text)
}

func LexicalToJSON(wrapper LexicalWrapper) (string, error) {
	b, err := json.MarshalIndent(wrapper, "", "")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
