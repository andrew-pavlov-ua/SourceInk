package articles_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	articles "sourceink/backend/internal/articles"
	"sourceink/backend/internal/model"
)

func TestValidateArticleReturnsValidDraft(t *testing.T) {
	draft := validationTestDraft()

	validated, err := new(articles.Service).ValidateArticle(context.Background(), draft)
	if err != nil {
		t.Fatalf("ValidateArticle() error = %v", err)
	}
	if validated.ValidationError != nil {
		t.Fatalf("ValidateArticle() validation error = %q", *validated.ValidationError)
	}
}

func TestValidateArticleReturnsDraftWithAllValidationIssues(t *testing.T) {
	draft := validationTestDraft()
	draft.Present = false
	draft.Title = "   "
	draft.Slug = ""

	validated, err := new(articles.Service).ValidateArticle(context.Background(), draft)
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("ValidateArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if validated.ValidationError == nil {
		t.Fatal("ValidateArticle() validation error is nil")
	}

	wantParts := []string{
		"source file is no longer present",
		"title is required",
		"slug is required",
	}
	for _, want := range wantParts {
		if !strings.Contains(*validated.ValidationError, want) {
			t.Errorf("ValidationError = %q, want %q", *validated.ValidationError, want)
		}
	}
}

func TestValidateArticleCountsTitleCharactersAsRunes(t *testing.T) {
	draft := validationTestDraft()
	draft.Title = strings.Repeat("ї", 121)

	validated, err := new(articles.Service).ValidateArticle(context.Background(), draft)
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("ValidateArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if validated.ValidationError == nil || !strings.Contains(*validated.ValidationError, "120 characters or fewer") {
		t.Fatalf("ValidateArticle() validation error = %v", validated.ValidationError)
	}
}

func TestValidateArticlePreservesParserValidationError(t *testing.T) {
	draft := validationTestDraft()
	parserError := "publish_mode must be manual or auto"
	draft.ValidationError = &parserError

	validated, err := new(articles.Service).ValidateArticle(context.Background(), draft)
	if !errors.Is(err, model.ErrArticleNotPublishable) {
		t.Fatalf("ValidateArticle() error = %v, want ErrArticleNotPublishable", err)
	}
	if validated.ValidationError == nil || *validated.ValidationError != parserError {
		t.Fatalf("ValidateArticle() validation error = %v, want preserved parser error", validated.ValidationError)
	}
}

func TestValidateArticleDoesNotStoreContextErrorAsValidation(t *testing.T) {
	draft := validationTestDraft()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	validated, err := new(articles.Service).ValidateArticle(ctx, draft)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ValidateArticle() error = %v, want context.Canceled", err)
	}
	if validated.ValidationError != nil {
		t.Fatalf("ValidateArticle() validation error = %q, want nil", *validated.ValidationError)
	}
}

func validationTestDraft() model.UnpublishedArticle {
	return model.UnpublishedArticle{
		SourcePath: "articles/validation.md",
		Frontmatter: model.Frontmatter{
			Title: "Valid article",
			Slug:  "valid-article",
		},
		Present: true,
	}
}
