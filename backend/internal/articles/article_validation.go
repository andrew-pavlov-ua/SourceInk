package articles

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"sourceink/backend/internal/model"
)

const maxArticleTitleLength = 120

// ValidateArticle applies the publication rules that do not require storage.
// It returns the draft with an actionable ValidationError when the article is
// invalid. Operational failures, such as context cancellation, do not mutate
// the draft's validation state.
func (s *Service) ValidateArticle(
	ctx context.Context,
	draft model.UnpublishedArticle,
) (model.UnpublishedArticle, error) {
	if err := ctx.Err(); err != nil {
		return draft, fmt.Errorf("validate article: %w", err)
	}

	if draft.ValidationError != nil {
		return draft, fmt.Errorf(
			"validate article %s: %s: %w",
			draft.SourcePath,
			*draft.ValidationError,
			model.ErrArticleNotPublishable,
		)
	}

	issues := make([]string, 0, 5)
	if !draft.Present {
		issues = append(issues, "source file is no longer present")
	}

	title := strings.TrimSpace(draft.Title)
	switch {
	case title == "":
		issues = append(issues, "title is required")
	case utf8.RuneCountInString(title) > maxArticleTitleLength:
		issues = append(issues, fmt.Sprintf("title must be %d characters or fewer", maxArticleTitleLength))
	}

	if strings.TrimSpace(draft.Slug) == "" {
		issues = append(issues, "slug is required")
	}

	if len(issues) == 0 {
		return draft, nil
	}

	err := ValidateWordsProcentile(strings.Fields(draft.Content))
	if err != nil {
		issues = append(issues, fmt.Sprintf("one word is used too many times in the article: %w", err))
	}

	validationMessage := strings.Join(issues, "; ")
	draft.ValidationError = &validationMessage
	return draft, fmt.Errorf(
		"validate article %s: %s: %w",
		draft.SourcePath,
		validationMessage,
		model.ErrArticleNotPublishable,
	)
}

const maxWordProcentile = 0.3

func ValidateWordsProcentile(wordsSummary []string) error {
	wordCount := make(map[string]int)
	for _, word := range wordsSummary {
		wordCount[word]++
	}

	for word, count := range wordCount {
		if float64(count)/float64(len(wordsSummary)) > maxWordProcentile {
			return fmt.Errorf("word \"%s\" used more than %d%% of the text", word, int(maxWordProcentile*100))
		}
	}

	return nil
}
