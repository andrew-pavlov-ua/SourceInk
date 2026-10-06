package github

import (
	"errors"
	"testing"
)

func TestParseFrontmatterRequiresOpeningAndClosingDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{name: "no delimiters", markdown: "# Plain Markdown\n"},
		{name: "no closing delimiter", markdown: "---\ntitle: Unfinished\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseFrontmatter(test.markdown)
			if !errors.Is(err, errFrontmatterBlockMissing) {
				t.Fatalf("ParseFrontmatter() error = %v, want missing frontmatter block", err)
			}
		})
	}
}

func TestParseFrontmatterAcceptsClosingDelimiterAtEndOfFile(t *testing.T) {
	parsed, err := ParseFrontmatter("---\ntitle: Hello\nslug: hello\npublish_mode: manual\n---")
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
	if parsed.Frontmatter.Title != "Hello" || parsed.Content != "" {
		t.Fatalf("ParseFrontmatter() = %#v", parsed)
	}
}

func TestParseFrontmatterAcceptsAutoPublishMode(t *testing.T) {
	parsed, err := ParseFrontmatter("---\ntitle: Hello\nslug: hello\npublish_mode: auto\n---\n# Hello")
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
	if parsed.Frontmatter.PublishMode != "auto" {
		t.Fatalf("publish mode = %q, want auto", parsed.Frontmatter.PublishMode)
	}
}

func TestParseFrontmatterRejectsUnsafeSlug(t *testing.T) {
	_, err := ParseFrontmatter("---\ntitle: Hello\nslug: Hello World\npublish_mode: manual\n---\n# Hello")
	if err == nil || err.Error() != "error validating frontmatter: slug must be 1-80 lowercase letters, numbers, or single hyphens" {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
}
