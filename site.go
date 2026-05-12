package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func buildSite(cfg Config, contentDir, staticDir, outDir string) error {
	postTmpl, err := compileTemplate("post")
	if err != nil {
		return fmt.Errorf("compile post template: %w", err)
	}
	indexTmpl, err := compileTemplate("index")
	if err != nil {
		return fmt.Errorf("compile index template: %w", err)
	}
	pageTmpl, err := compileTemplate("page")
	if err != nil {
		return fmt.Errorf("compile page template: %w", err)
	}

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("clean output dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	now := time.Now()

	posts, err := loadPosts(contentDir)
	if err != nil {
		return err
	}

	pages, err := loadPages(contentDir)
	if err != nil {
		return err
	}

	for i := range posts {
		p := posts[i]
		dir := filepath.Join(outDir, "posts", p.Slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		var buf bytes.Buffer
		data := pageData{Config: cfg, Post: p, Pages: pages, Path: "/posts/" + p.Slug + "/", Now: now}
		if err := renderTemplate(&buf, postTmpl, "post.html", data); err != nil {
			return fmt.Errorf("render post %s: %w", p.Slug, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write post %s: %w", p.Slug, err)
		}
		log.Printf("post  %s", p.Slug)
	}

	{
		var buf bytes.Buffer
		data := pageData{Config: cfg, Posts: posts, Pages: pages, Path: "/", Now: now}
		if err := renderTemplate(&buf, indexTmpl, "index.html", data); err != nil {
			return fmt.Errorf("render index: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "index.html"), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write index: %w", err)
		}
		log.Printf("index /")
	}

	for i := range pages {
		p := pages[i]
		dir := filepath.Join(outDir, p.Slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		var buf bytes.Buffer
		data := pageData{Config: cfg, Post: p, Pages: pages, Path: "/" + p.Slug + "/", Now: now}
		if err := renderTemplate(&buf, pageTmpl, "page.html", data); err != nil {
			return fmt.Errorf("render page %s: %w", p.Slug, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write page %s: %w", p.Slug, err)
		}
		log.Printf("page  %s", p.Slug)
	}

	if err := copyDir(staticDir, outDir); err != nil {
		return fmt.Errorf("copy static: %w", err)
	}

	return nil
}

func loadPosts(contentDir string) ([]Post, error) {
	paths, err := filepath.Glob(filepath.Join(contentDir, "posts", "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob posts: %w", err)
	}

	var posts []Post
	for _, path := range paths {
		post, err := parsePost(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if post.Date == "" {
			return nil, fmt.Errorf("%s: posts require a date in frontmatter", path)
		}
		if post.Draft {
			continue
		}
		html, err := renderMarkdown(post.Body)
		if err != nil {
			return nil, fmt.Errorf("render markdown %s: %w", path, err)
		}
		post.HTMLBody = template.HTML(html)
		posts = append(posts, post)
	}

	sortPostsByDate(posts)
	return posts, nil
}

func loadPages(contentDir string) ([]Post, error) {
	paths, err := filepath.Glob(filepath.Join(contentDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}

	var pages []Post
	for _, path := range paths {
		page, err := parsePost(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if page.Date != "" {
			return nil, fmt.Errorf("%s: pages should not have a date in frontmatter — move to content/posts/", path)
		}
		html, err := renderMarkdown(page.Body)
		if err != nil {
			return nil, fmt.Errorf("render markdown %s: %w", path, err)
		}
		page.HTMLBody = template.HTML(html)
		pages = append(pages, page)
	}

	sortPagesByNavOrder(pages)
	return pages, nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", src, err)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dst, err)
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy dir %s: %w", srcPath, err)
			}
		} else {
			s, err := os.Open(srcPath)
			if err != nil {
				return fmt.Errorf("open %s: %w", srcPath, err)
			}
			d, err := os.Create(dstPath)
			if err != nil {
				s.Close()
				return fmt.Errorf("create %s: %w", dstPath, err)
			}
			_, err = io.Copy(d, s)
			s.Close()
			d.Close()
			if err != nil {
				return fmt.Errorf("copy %s to %s: %w", srcPath, dstPath, err)
			}
		}
	}
	return nil
}
