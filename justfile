set windows-shell := ['nu', '-c']

pikchr:
    go run ./cmd/pikchrgen

build: pikchr
    go tool hugo build

deploy: build
    bunx wrangler pages deploy

watch: pikchr
    go tool hugo server --watch --port 12000

new-typst title:
    go tool hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.typst

new-md title:
    go tool hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.md
