package main

import (
	"embed"
	"html/template"
	"io"
	"time"
)

//go:embed templates
var templateFS embed.FS

type pageData struct {
	Config Config
	Posts  []Post
	Pages  []Post
	Post   Post
	Now    time.Time
	Path   string
}

func compileTemplate(page string) (*template.Template, error) {
	return template.New("").ParseFS(templateFS, "templates/base.html", "templates/"+page+".html")
}

func renderTemplate(w io.Writer, tmpl *template.Template, name string, data pageData) error {
	return tmpl.ExecuteTemplate(w, name, data)
}
