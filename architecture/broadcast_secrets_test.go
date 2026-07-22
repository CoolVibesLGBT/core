package architecture_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestBroadcastCredentialsAreNotHardcodedInGoSources(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)r:[0-9a-f]{24,}`),
		regexp.MustCompile(`(?i)eyJ2Ijpb[[:alnum:]+/=]{20,}`),
		regexp.MustCompile(`(?i)["'](?:x-parse-session-token|x-parse-client-key|x-parse-installation-id|newrelic|x-newrelic-id)["']\s*[,=:]\s*["'][^"']{4,}`),
	}

	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range patterns {
			if pattern.Match(content) {
				t.Errorf("broadcast credential-like literal found in %s", strings.TrimPrefix(path, "../"))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan Go sources: %v", err)
	}
}
