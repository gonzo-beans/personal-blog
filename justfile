build:
    hugo build

deploy: build
    bunx wrangler pages deploy
