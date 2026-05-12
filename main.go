package main

import (
	"flag"
	"log"
	"path/filepath"
)

func main() {
	siteDir := flag.String("site", "", "site root directory (sets content, static, config, and output paths)")
	configPath := flag.String("c", "", "config file path")
	contentDir := flag.String("content", "", "content directory")
	staticDir := flag.String("static", "", "static assets directory")
	outDir := flag.String("out", "", "output directory")
	serveFlag := flag.Bool("serve", false, "build and serve locally")
	port := flag.String("port", "8081", "port to serve on")
	flag.Parse()

	if *siteDir != "" {
		if *contentDir == "" {
			*contentDir = filepath.Join(*siteDir, "content")
		}
		if *staticDir == "" {
			*staticDir = filepath.Join(*siteDir, "static")
		}
		if *outDir == "" {
			*outDir = filepath.Join(*siteDir, "public")
		}
		if *configPath == "" {
			*configPath = filepath.Join(*siteDir, "hubara.yaml")
		}
	} else {
		if *contentDir == "" {
			*contentDir = "content"
		}
		if *staticDir == "" {
			*staticDir = "static"
		}
		if *outDir == "" {
			*outDir = "public"
		}
		if *configPath == "" {
			*configPath = "hubara.yaml"
		}
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}

	if err := buildSite(cfg, *contentDir, *staticDir, *outDir); err != nil {
		log.Fatalf("build failed: %v", err)
	}

	log.Print("built successfully")

	if *serveFlag {
		if err := startServer(":"+*port, *configPath, *contentDir, *staticDir, *outDir); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}
}
