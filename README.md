# hubara

a simple barebones static site generator written in Go.

your **content** lives in one repo. this is the **tool** repo.

see [`examples/content-repo`](examples/content-repo) for a ready-made template with a Vercel deploy workflow.

## install

```bash
curl -sL https://github.com/canaribo/hubara/releases/latest/download/hubara-linux-amd64 -o hubara
chmod +x hubara
```

or with go:

```bash
go install github.com/canaribo/hubara@latest
```

## build

```bash
hubara                            # builds public/
hubara -site ./examples/content-repo   # resolves content, static, config, and output from this dir
hubara -serve                   # builds and serves on :8081
hubara -serve -port 3000        # custom port
```

flags:

```
-site     site root directory (resolves content, static, config, and output)
-content  content directory (default: content, or <site>/content if -site is set)
-static   static assets directory (default: static, or <site>/static if -site is set)
-out      output directory (default: public, or <site>/public if -site is set)
-c        config file path (default: hubara.yaml, or <site>/hubara.yaml if -site is set)
-serve    build and serve locally
-port     port to serve on (default: 8081)
```

## content structure

```
content/
  *.md          # pages (no date)
  posts/
    *.md        # blog posts (need a date)
static/
  main.css      # styles
hubara.yaml     # config
```

see the example repo for a full setup with frontmatter, config, and deploy to Vercel.

## license

MIT
