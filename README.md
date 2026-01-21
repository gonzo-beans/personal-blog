# blog.andreani.in

## Setup

The Cloudflare account ID is configured in `.env` file (not tracked in git).
Create it if it doesn't exist:

```shell
echo "CLOUDFLARE_ACCOUNT_ID=xxx" > .env
```

## Deploy

```shell
bunx wrangler pages deploy
```