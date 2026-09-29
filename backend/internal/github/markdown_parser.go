package github

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"

	"sourceink/backend/internal/model"
)

type ParsedMarkdown struct {
	Frontmatter model.Frontmatter
	Content     string
}

var errFrontmatterBlockMissing = errors.New("complete frontmatter block is required")

func ParseFrontmatter(markdown string) (ParsedMarkdown, error) {
	source := strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return ParsedMarkdown{}, fmt.Errorf("%w: frontmatter must start with ---", errFrontmatterBlockMissing)
	}

	closingDelimiter := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			closingDelimiter = i
			break
		}
	}
	if closingDelimiter == -1 {
		return ParsedMarkdown{}, fmt.Errorf("%w: frontmatter closing delimiter is missing", errFrontmatterBlockMissing)
	}

	yamlSource := strings.Join(lines[1:closingDelimiter], "\n")
	body := strings.Join(lines[closingDelimiter+1:], "\n")

	var fm model.Frontmatter
	err := yaml.Unmarshal([]byte(yamlSource), &fm)
	if err != nil {
		return ParsedMarkdown{}, fmt.Errorf("error unmarshalling frontmatter: %w", err)
	}

	if err = ValidateFrontmatter(fm); err != nil {
		return ParsedMarkdown{}, fmt.Errorf("error validating frontmatter: %w", err)
	}

	return ParsedMarkdown{
		Frontmatter: fm,
		Content:     body,
	}, nil
}

func ValidateFrontmatter(fm model.Frontmatter) error {
	if strings.TrimSpace(fm.Title) == "" {
		return errors.New("title is required")
	}

	if strings.TrimSpace(fm.Slug) == "" {
		return errors.New("slug is required")
	}

	switch fm.PublishMode {
	case string(model.ArticlePublishModeManual), string(model.ArticlePublishModeAuto):
	default:
		return errors.New("publish_mode must be manual or auto")
	}

	return nil
}
