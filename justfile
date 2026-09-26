set windows-shell := ['nu', '-c']

pikchr:
    go run ./cmd/pikchrgen

build: pikchr
    go tool hugo build

deploy: build
    bunx wrangler pages deploy

pikchr-watch:
    go run ./cmd/pikchrgen --watch

hugo-watch:
    go tool hugo server --watch --port 12000

[parallel]
watch: pikchr-watch hugo-watch

new-typst title:
    go tool hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.typst

new-md title:
    go tool hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.md
