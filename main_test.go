package main

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "h.yaml")
		if err := os.WriteFile(configPath, []byte("title: My Blog\nauthor: Alice\nbaseURL: https://example.com\n"), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadConfig(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Title != "My Blog" || cfg.Author != "Alice" || cfg.BaseURL != "https://example.com" {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("missing file returns empty", func(t *testing.T) {
		cfg, err := loadConfig("/nonexistent/path.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Title != "" {
			t.Errorf("expected empty config, got %+v", cfg)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "h.yaml")
		if err := os.WriteFile(configPath, []byte(": invalid\n"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := loadConfig(configPath)
		if err == nil {
			t.Fatal("expected error for invalid yaml")
		}
	})
}

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantFM         string
		wantBody       string
		wantOK         bool
	}{
		{"basic", "---\ntitle: hello\n---\nbody here", "title: hello", "body here", true},
		{"no frontmatter", "just body", "", "", false},
		{"empty", "", "", "", false},
		{"only dashes", "---\n---\n", "", "", true},
		{"windows line endings", "---\r\ntitle: hello\r\n---\r\nbody", "title: hello", "body", true},
		{"multi-line fm", "---\ntitle: hello\ndate: 2026-01-01\n---\n\nbody\n", "title: hello\ndate: 2026-01-01", "body", true},
		{"leading whitespace trimmed", "  ---\ntitle: hello\n---\nbody", "title: hello", "body", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, ok := splitFrontmatter(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if fm != tt.wantFM {
				t.Errorf("fm = %q, want %q", fm, tt.wantFM)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		want  string // time.Time formatted as 2006-01-02
	}{
		{"2026-05-09", "2026-05-09"},
		{"2026-05-09T12:00:00Z", "2026-05-09"},
		{"2026-05-09T12:00:00", "2026-05-09"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Format("2006-01-02") != tt.want {
				t.Errorf("got %s, want %s", got.Format("2006-01-02"), tt.want)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		_, err := parseDate("not a date")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSlugFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"content/hello-world.md", "hello-world"},
		{"content/foo.markdown", "foo"},
		{"/abs/path/post.md", "post"},
		{"post.md", "post"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := slugFromPath(tt.path)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePost(t *testing.T) {
	t.Run("valid post", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "hello.md")
		content := "---\ntitle: Hello World\ndate: 2026-01-15\n---\n\n# Hi\n\nThis is a post.\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		post, err := parsePost(path)
		if err != nil {
			t.Fatal(err)
		}
		if post.Title != "Hello World" {
			t.Errorf("title = %q", post.Title)
		}
		if post.Slug != "hello" {
			t.Errorf("slug = %q", post.Slug)
		}
		if post.Body != "# Hi\n\nThis is a post." {
			t.Errorf("body = %q", post.Body)
		}
		if post.parsed.Format("2006-01-02") != "2026-01-15" {
			t.Errorf("date = %s", post.parsed.Format("2006-01-02"))
		}
	})

	t.Run("draft post", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "draft.md")
		if err := os.WriteFile(path, []byte("---\ntitle: Draft\ndate: 2026-01-01\ndraft: true\n---\nbody\n"), 0644); err != nil {
			t.Fatal(err)
		}

		post, err := parsePost(path)
		if err != nil {
			t.Fatal(err)
		}
		if !post.Draft {
			t.Error("expected draft = true")
		}
	})

	t.Run("missing frontmatter", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "nofm.md")
		if err := os.WriteFile(path, []byte("just body"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := parsePost(path)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid date", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "bad.md")
		if err := os.WriteFile(path, []byte("---\ntitle: Bad\ndate: yesterday\n---\nbody\n"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := parsePost(path)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no date", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "nodate.md")
		if err := os.WriteFile(path, []byte("---\ntitle: No Date\n---\nbody\n"), 0644); err != nil {
			t.Fatal(err)
		}

		post, err := parsePost(path)
		if err != nil {
			t.Fatal(err)
		}
		if !post.parsed.IsZero() {
			t.Error("expected zero time")
		}
	})
}

func TestSortPostsByDate(t *testing.T) {
	posts := []Post{
		{Title: "old", parsed: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "new", parsed: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "mid", parsed: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
	}
	sortPostsByDate(posts)
	if posts[0].Title != "new" {
		t.Errorf("pos 0 = %s, want new", posts[0].Title)
	}
	if posts[1].Title != "mid" {
		t.Errorf("pos 1 = %s, want mid", posts[1].Title)
	}
	if posts[2].Title != "old" {
		t.Errorf("pos 2 = %s, want old", posts[2].Title)
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"heading", "# Hello", "<h1>Hello</h1>\n"},
		{"paragraph", "hello world", "<p>hello world</p>\n"},
		{"code", "`code`", "<p><code>code</code></p>\n"},
		{"link", "[text](https://x.com)", "<p><a href=\"https://x.com\">text</a></p>\n"},
		{"bold", "**bold**", "<p><strong>bold</strong></p>\n"},
		{"list", "- a\n- b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderMarkdown(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("code block", func(t *testing.T) {
		got, err := renderMarkdown("```go\nfunc main() {}\n```")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "<pre><code") {
			t.Errorf("expected code block in: %s", got)
		}
	})
}

func TestCompileTemplates(t *testing.T) {
	tmpl, err := compileTemplate("post")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Lookup("post.html") == nil {
		t.Error("missing post.html template")
	}

	tmpl2, err := compileTemplate("index")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl2.Lookup("index.html") == nil {
		t.Error("missing index.html template")
	}
}

func TestRenderPostTemplate(t *testing.T) {
	tmpl, err := compileTemplate("post")
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{Title: "Test Blog", Author: "Test Author"}
	post := Post{
		Title:     "Hello World",
		Slug:      "hello-world",
		HTMLBody:  template.HTML("<p>content</p>"),
		DateShort: "2026-05-09",
		DateLong:  "May 9, 2026",
	}

	data := pageData{Config: cfg, Post: post, Now: time.Now()}

	var buf bytes.Buffer
	err = renderTemplate(&buf, tmpl, "post.html", data)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	checks := []string{
		"<!DOCTYPE html>",
		"Test Blog",
		"Hello World",
		"<p>content</p>",
		"May 9, 2026",
		"Test Author",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("expected output to contain %q", c)
		}
	}
}

func TestRenderIndexTemplate(t *testing.T) {
	tmpl, err := compileTemplate("index")
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{Title: "Test Blog", Author: "Test Author"}
	posts := []Post{
		{Title: "Post One", Slug: "post-one", DateShort: "2026-05-09"},
		{Title: "Post Two", Slug: "post-two", DateShort: "2026-01-01"},
	}

	data := pageData{Config: cfg, Posts: posts, Now: time.Now()}

	var buf bytes.Buffer
	err = renderTemplate(&buf, tmpl, "index.html", data)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	checks := []string{
		"<!DOCTYPE html>",
		"Test Blog",
		"Test Author",
		"Post One",
		"Post Two",
		"/posts/post-one/",
		"/posts/post-two/",
		"blog-posts",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("expected output to contain %q", c)
		}
	}
}

func TestBuildIntegration(t *testing.T) {
	src := t.TempDir()

	postsDir := filepath.Join(src, "content", "posts")
	if err := os.MkdirAll(postsDir, 0755); err != nil {
		t.Fatal(err)
	}

	staticDir := filepath.Join(src, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(postsDir, "first-post.md"), []byte("---\ntitle: First Post\ndate: 2026-05-09\n---\n\n# Hello\n\nFirst post body.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "second-post.md"), []byte("---\ntitle: Second Post\ndate: 2026-01-01\n---\n\nSecond post body.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "draft-post.md"), []byte("---\ntitle: Draft\ndate: 2026-05-10\ndraft: true\n---\nSecret.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(src, "content", "links.md"), []byte("---\ntitle: Links\nnav: true\nnavOrder: 1\n---\nSome links.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "content", "about.md"), []byte("---\ntitle: About\n---\nAbout this site.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(staticDir, "main.css"), []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(src)
	defer os.Chdir(orig)

	cfg := Config{Title: "Test", Author: "Tester"}

	if err := buildSite(cfg, "content", "static", "public"); err != nil {
		t.Fatal(err)
	}

	idx, err := os.ReadFile(filepath.Join(src, "public", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "First Post") {
		t.Error("index missing First Post")
	}
	if !strings.Contains(string(idx), "Second Post") {
		t.Error("index missing Second Post")
	}
	if strings.Contains(string(idx), "Draft") {
		t.Error("index should not contain draft post")
	}
	fpIdx := strings.Index(string(idx), "First Post")
	spIdx := strings.Index(string(idx), "Second Post")
	if fpIdx > spIdx {
		t.Error("posts sorted newest-first: First Post should appear before Second Post")
	}
	if !strings.Contains(string(idx), "Links") {
		t.Error("index nav missing Links page")
	}
	navStart := strings.Index(string(idx), "<nav>")
	navEnd := strings.Index(string(idx), "</nav>")
	if navStart == -1 || navEnd == -1 {
		t.Fatal("missing nav in index")
	}
	navHTML := string(idx)[navStart:navEnd]
	if strings.Contains(navHTML, "/about/") {
		t.Error("index nav should not contain about page (nav:false)")
	}

	p1, err := os.ReadFile(filepath.Join(src, "public", "posts", "first-post", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(p1), "First Post") {
		t.Error("post page missing title")
	}
	if !strings.Contains(string(p1), "First post body") {
		t.Error("post page missing body")
	}

	p2, err := os.ReadFile(filepath.Join(src, "public", "posts", "second-post", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(p2), "Second Post") {
		t.Error("second post page missing title")
	}

	_, err = os.Stat(filepath.Join(src, "public", "posts", "draft-post"))
	if err == nil {
		t.Error("draft post should not be published")
	}

	linksFile, err := os.ReadFile(filepath.Join(src, "public", "links", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(linksFile), "Some links") {
		t.Error("links page missing body")
	}

	aboutFile, err := os.ReadFile(filepath.Join(src, "public", "about", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(aboutFile), "About this site") {
		t.Error("about page missing body")
	}

	css, err := os.ReadFile(filepath.Join(src, "public", "main.css"))
	if err != nil {
		t.Fatal(err)
	}
	if string(css) != "body{}" {
		t.Error("CSS not copied correctly")
	}
}

func TestHTMLSafeOutput(t *testing.T) {
	got, err := renderMarkdown("<script>alert(1)</script>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "<script>") {
		t.Error("markdown output should not contain raw HTML")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	b1, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != "hello" {
		t.Errorf("a.txt has %q", string(b1))
	}
	b2, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b2) != "world" {
		t.Errorf("sub/b.txt has %q", string(b2))
	}
}

func TestCopyDirMissing(t *testing.T) {
	dst := t.TempDir()
	err := copyDir("/nonexistent/src", dst)
	if err != nil {
		t.Fatal(err)
	}
}
