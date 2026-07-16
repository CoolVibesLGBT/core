package helpers

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

func ReadJSON[T any](path string) (*T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer CloseQuietly(file)

	var result T

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func ReadFileToString(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ExtractJSON(input string) string {
	re := regexp.MustCompile(`\{.*\}`)
	match := re.FindString(input)
	return strings.TrimSpace(match)
}
