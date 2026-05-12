package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Post struct {
	Title     string        `yaml:"title"`
	Date      string        `yaml:"date"`
	Draft     bool          `yaml:"draft"`
	Nav       bool          `yaml:"nav"`
	NavOrder  int           `yaml:"navOrder"`
	Slug      string        `yaml:"-"`
	Body      string        `yaml:"-"`
	HTMLBody  template.HTML `yaml:"-"`
	parsed    time.Time     `yaml:"-"`
	DateShort string        `yaml:"-"`
	DateLong  string        `yaml:"-"`
}

func parsePost(path string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, err
	}

	p := Post{
		Slug: slugFromPath(path),
	}

	frontmatter, body, ok := splitFrontmatter(string(data))
	if !ok {
		return Post{}, fmt.Errorf("%s: missing frontmatter", path)
	}

	if err := yaml.Unmarshal([]byte(frontmatter), &p); err != nil {
		return Post{}, fmt.Errorf("%s: invalid frontmatter: %w", path, err)
	}

	p.Body = body

	if p.Date != "" {
		t, err := parseDate(p.Date)
		if err != nil {
			return Post{}, fmt.Errorf("%s: invalid date %q: %w", path, p.Date, err)
		}
		p.parsed = t
		p.DateShort = t.Format("2006-01-02")
		p.DateLong = t.Format("January 2, 2006")
	}

	return p, nil
}

func splitFrontmatter(raw string) (frontmatter string, body string, ok bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---\n") && !strings.HasPrefix(raw, "---\r\n") {
		return "", "", false
	}
	raw = raw[3:]
	idx := strings.Index(raw, "\n---")
	if idx < 0 {
		return "", "", false
	}
	frontmatter = strings.TrimSpace(raw[:idx])
	rest := raw[idx+1:]
	if strings.HasPrefix(rest, "---\r\n") {
		rest = rest[5:]
	} else if strings.HasPrefix(rest, "---\n") {
		rest = rest[4:]
	} else if strings.HasPrefix(rest, "---") {
		rest = rest[3:]
	}
	return frontmatter, strings.TrimSpace(rest), true
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

func slugFromPath(path string) string {
	base := filepath.Base(path)
	for _, ext := range []string{".md", ".markdown"} {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

func sortPostsByDate(posts []Post) {
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].parsed.After(posts[j].parsed)
	})
}

func sortPagesByNavOrder(pages []Post) {
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].NavOrder != pages[j].NavOrder {
			return pages[i].NavOrder < pages[j].NavOrder
		}
		return pages[i].Slug < pages[j].Slug
	})
}

