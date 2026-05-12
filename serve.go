package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var reloadScript = []byte(`<script>(function(){var e=new EventSource("/__hubara_reload");e.onmessage=function(){e.close();location.reload()};e.onerror=function(){e.close()}})()</script>`)

func startServer(addr string, configPath string, contentDir, staticDir, outDir string) error {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir(outDir))

	// live-reload: SSE clients
	var (
		mu      sync.Mutex
		clients = make(map[chan struct{}]struct{})
	)

	broadcast := func() {
		mu.Lock()
		defer mu.Unlock()
		for ch := range clients {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	// watch files and rebuild
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		mods := make(map[string]time.Time)
		scan := func() (bool, error) {
			current := make(map[string]time.Time)

			for _, dir := range []string{contentDir, staticDir} {
				if err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() {
						current[p] = info.ModTime()
					}
					return nil
				}); err != nil {
					return false, fmt.Errorf("watch %s: %w", dir, err)
				}
			}

			if info, err := os.Stat(configPath); err == nil {
				current[configPath] = info.ModTime()
			}

			changed := len(current) != len(mods)
			if !changed {
				for p, t := range current {
					if old, ok := mods[p]; !ok || !t.Equal(old) {
						changed = true
						break
					}
				}
			}
			mods = current
			return changed, nil
		}

		if _, err := scan(); err != nil {
			log.Printf("watcher: %v", err)
		}
		for range ticker.C {
			changed, err := scan()
			if err != nil {
				log.Printf("watcher: %v", err)
				continue
			}
			if !changed {
				continue
			}

			log.Print("change detected, rebuilding...")
			cfg, err := loadConfig(configPath)
			if err != nil {
				log.Printf("rebuild: could not reload config: %v", err)
				continue
			}
			if err := buildSite(cfg, contentDir, staticDir, outDir); err != nil {
				log.Printf("rebuild: %v", err)
				continue
			}
			log.Print("rebuilt")
			broadcast()
		}
	}()

	mux.HandleFunc("GET /__hubara_reload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctrl := http.NewResponseController(w)

		ch := make(chan struct{}, 1)
		mu.Lock()
		clients[ch] = struct{}{}
		mu.Unlock()

		defer func() {
			mu.Lock()
			delete(clients, ch)
			mu.Unlock()
		}()

		fmt.Fprintf(w, ": connected\n\n")
		if err := ctrl.Flush(); err != nil {
			return
		}

		for {
			select {
			case <-ch:
				fmt.Fprintf(w, "data: reload\n\n")
				if err := ctrl.Flush(); err != nil {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if urlPath == "" || strings.HasSuffix(r.URL.Path, "/") {
			urlPath = path.Join(urlPath, "index.html")
		}

		if strings.HasSuffix(urlPath, ".html") {
			f, err := http.Dir(outDir).Open(urlPath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			content, err := io.ReadAll(f)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			content = bytes.Replace(content, []byte("</body>"), append(reloadScript, []byte("</body>")...), 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(content)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	server := &http.Server{Addr: addr, Handler: mux}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	url := "http://" + addr
	if strings.HasPrefix(addr, ":") {
		url = "http://localhost" + addr
	}
	log.Printf("serving %s", url)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
