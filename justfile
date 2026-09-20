set windows-shell := ['nu', '-c']

pikchr:
    go run ./cmd/pikchrgen

build: pikchr
    hugo build

deploy: build
    bunx wrangler pages deploy

watch: pikchr
    hugo server --watch --port 12000

new-typst title:
    hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.typst

new-md title:
    hugo new content/posts/$(date +%Y-%m-%d)-{{ title }}.md
