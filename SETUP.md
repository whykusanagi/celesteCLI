# CelesteCLI Setup Guide

## Quick Start

### 1. Build the CLI
```bash
go build -o celestecli main.go scaffolding.go
```

### 2. Configure CelesteAI Agent
Create `~/.celesteAI`:
```bash
endpoint=https://your-celeste-api-endpoint
api_key=your-api-key
client_id=your-igdb-client-id
secret=your-igdb-client-secret
```

### 3. Test Basic Functionality
```bash
./celestecli --type tweet --game "NIKKE" --tone "teasing"
```

## Optional Configurations

### DigitalOcean Spaces (for --sync flag)
Create `~/.celeste.cfg`:
```bash
access_key_id=your_do_spaces_access_key
secret_access_key=your_do_spaces_secret_key
endpoint=https://sfo3.digitaloceanspaces.com
region=sfo3
bucket_name=whykusanagi
```

### NSFW Mode (Venice.ai)
Add to `~/.celesteAI`:
```bash
venice_api_key=your_venice_api_key
```

## Usage Examples

### Regular Mode
```bash
# Twitter post
./celestecli --type tweet --game "NIKKE" --tone "teasing"

# YouTube description
./celestecli --type ytdesc --game "NIKKE" --tone "lewd"

# TikTok caption
./celestecli --type tiktok --game "NIKKE" --tone "chaotic"
```

### NSFW Mode
```bash
# Uncensored content
./celestecli --nsfw --type tweet --tone "explicit" --game "NIKKE"
```

### With Sync
```bash
# Upload to OpenSearch
./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync
```

## Troubleshooting

### Missing API Key
```
Missing CELESTE_API_ENDPOINT or CELESTE_API_KEY
```
**Fix**: Set environment variables or update `~/.celesteAI`

### Venice.ai Error
```
Venice.ai configuration error: missing Venice.ai API key
```
**Fix**: Set `VENICE_API_KEY` or add `venice_api_key` to `~/.celesteAI`

### S3 Upload Failed
```
Warning: Failed to upload conversation to S3
```
**Fix**: Check DigitalOcean Spaces credentials in `~/.celeste.cfg`

## Help
```bash
./celestecli --help
```
