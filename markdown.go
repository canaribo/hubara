package main

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
)

var md = goldmark.New()

func renderMarkdown(input string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return buf.String(), nil
}
