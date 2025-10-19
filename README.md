# CelesteCLI - CelesteAI Command Line Interface

A Go-based CLI tool for interacting with CelesteAI, a mischievous demon noble VTuber assistant. This tool provides various content generation capabilities while maintaining Celeste's distinctive personality and voice.

## Features

### Content Generation
- **Twitter/X Posts** - Generate tweets in Celeste's voice with proper hashtags and tone
- **YouTube Descriptions** - Create detailed video descriptions with game metadata from IGDB
- **Stream Titles** - Generate punchy, chaotic stream titles
- **Discord Announcements** - Create stream announcements for Discord
- **TikTok Captions** - Generate engaging short-form content captions
- **Pixiv Posts** - Generate artistic captions for illustrations
- **Skeb Requests** - Draft professional commission requests
- **Tarot Readings** - Generate mystical tarot readings with Celtic or three-card spreads
- **Goodnight Messages** - Create flirty or cozy goodnight tweets
- **Quote Tweets** - Generate witty quote tweet responses
- **Reply Snark** - Create snarky replies to tweets
- **Birthday Messages** - Generate celebratory birthday content
- **Alt Text** - Create descriptive image alt text

### Advanced Features
- **IGDB Integration** - Automatic game metadata retrieval and caching
- **Personality Switching** - Dynamic persona selection based on context
- **OpenSearch Sync** - Upload conversations to Celeste's knowledge base
- **NSFW Mode** - Uncensored content generation using Venice.ai
- **Scaffolding System** - External JSON configuration for prompt templates
- **Bot Integration** - Discord and Twitch bot support with user isolation
- **Override Functionality** - PGP-signed override commands for bypassing restrictions

## Installation

```bash
go build -o celestecli main.go scaffolding.go
```

## Configuration

### CelesteAI Agent Configuration

Create a `~/.celesteAI` config file with:
```bash
endpoint=https://your-celeste-api-endpoint
api_key=your-api-key
client_id=your-igdb-client-id
secret=your-igdb-client-secret

# NSFW Mode (optional)
venice_api_key=your-venice-api-key
```

Or set environment variables:
- `CELESTE_API_ENDPOINT`
- `CELESTE_API_KEY`
- `CELESTE_IGDB_CLIENT_ID`
- `CELESTE_IGDB_CLIENT_SECRET`
- `VENICE_API_KEY` (for NSFW mode)

### DigitalOcean Spaces Configuration (for --sync flag)

Create a `~/.celeste.cfg` config file with:
```bash
access_key_id=your_digitalocean_spaces_access_key_here
secret_access_key=your_digitalocean_spaces_secret_key_here
endpoint=https://sfo3.digitaloceanspaces.com
region=sfo3
bucket_name=whykusanagi
```

Or set environment variables:
- `DO_SPACES_ACCESS_KEY_ID`
- `DO_SPACES_SECRET_ACCESS_KEY`

## Usage

### Basic Commands

```bash
# Generate a tweet
./celestecli --type tweet --tone "chaotic funny"

# Create YouTube description with game context
./celestecli --type ytdesc --game "NIKKE" --tone "lewd"

# Generate stream title
./celestecli --type title --game "Schedule I" --tone "dramatic"

# Create Discord announcement
./celestecli --type discord --game "Blue Archive" --tone "hype"

# Generate Pixiv post caption
./celestecli --type pixivpost --game "Fall of Kirara" --tone "dramatic"

# Draft Skeb commission request
./celestecli --type skebreq --game "Celeste" --tone "professional" --context "bunny outfit"

# Generate tarot reading
./celestecli --type tarot --spread celtic

# Create goodnight message
./celestecli --type goodnight --tone "sweet teasing"

# NSFW Mode (uncensored content using Venice.ai)
./celestecli --nsfw --type tweet --tone "explicit" --game "NIKKE"
./celestecli --nsfw --type tiktok --tone "lewd" --game "NIKKE"
./celestecli --nsfw --type ytdesc --tone "adult" --game "NIKKE"
```

### Advanced Options

```bash
# Generate content with specific persona
./celestecli --type tweet --persona celeste_ad_read --tone "wink-and-nudge" --game "NIKKE"

# Generate content with media context
./celestecli --type tweet --media "https://example.com/image.jpg" --tone "teasing"

# Generate content with additional context
./celestecli --type ytdesc --context "This is a special stream event" --game "NIKKE"

# Upload conversation to OpenSearch
./celestecli --type tweet --sync --game "NIKKE"

# Enable debug mode
./celestecli --type tweet --debug
```

## Content Types

| Type | Description | Max Length | Platform |
|------|-------------|------------|----------|
| `tweet` | Twitter post | 280 | Twitter |
| `tweet_image` | Twitter post with image credit | 280 | Twitter |
| `tweet_thread` | Multi-part Twitter thread | 280 | Twitter |
| `title` | Stream title | 140 | Streaming |
| `ytdesc` | YouTube description | 5000 | YouTube |
| `tiktok` | TikTok caption | 2200 | TikTok |
| `discord` | Discord announcement | 2000 | Discord |
| `goodnight` | Goodnight message | 280 | Twitter |
| `pixivpost` | Pixiv post caption | 1000 | Pixiv |
| `skebreq` | Skeb commission request | 900 | Skeb |
| `quote_tweet` | Quote tweet response | 280 | Twitter |
| `reply_snark` | Snarky reply | 280 | Twitter |
| `birthday` | Birthday message | 280 | Twitter |
| `alt_text` | Image alt text | 125 | Accessibility |

## Tone Examples

- `lewd` - Suggestive and teasing
- `explicit` - Direct and uncensored (NSFW mode)
- `teasing` - Playful and mischievous
- `chaotic` - Wild and unpredictable
- `cute` - Sweet and endearing
- `official` - Professional and formal
- `dramatic` - Intense and emotional
- `parody` - Humorous and satirical
- `funny` - Comedy and entertainment
- `suggestive` - Hinting and playful
- `adult` - Mature and sophisticated
- `sweet` - Gentle and caring
- `snarky` - Sarcastic and witty
- `playful` - Fun and lighthearted
- `hype` - Energetic and exciting

## Personas

- `celeste_stream` - Default streaming persona (teasing, smug, mischievous, playful)
- `celeste_ad_read` - Advertisement reading persona (wink-and-nudge, promotional, engaging)
- `celeste_moderation_warning` - Moderation warning persona (authoritative, clear, firm but fair)

## NSFW Mode

The CLI supports NSFW mode using Venice.ai for uncensored content generation:

### Configuration
Add to `~/.celesteAI`:
```bash
venice_api_key=your_venice_api_key_here
venice_base_url=https://api.venice.ai/api/v1
venice_model=venice-uncensored
venice_upscaler=upscaler
```

### Usage
```bash
# NSFW Twitter post
./celestecli --nsfw --type tweet --tone "explicit" --game "NIKKE"

# NSFW TikTok caption
./celestecli --nsfw --type tiktok --tone "lewd" --game "NIKKE"
```

### Venice.ai Models
- **`venice-uncensored`** - No content filtering, full NSFW content generation
- **`lustify-sdxl`** - Uncensored image generation with SDXL
- **`wai-Illustrious`** - Anime-style generation for VTuber content
- **`upscaler`** - Image upscaling (2x $0.02, 4x $0.08)

## Scaffolding System

The CLI uses an external JSON configuration system for prompt templates:

### Configuration File: `scaffolding.json`
```json
{
  "content_types": {
    "tweet": {
      "description": "Write a post for X/Twitter",
      "scaffold": "🐦 Write a Twitter post in CelesteAI's voice...",
      "max_length": 280,
      "platform": "twitter"
    }
  },
  "tone_examples": {
    "lewd": "suggestive and teasing",
    "explicit": "direct and uncensored"
  },
  "platforms": {
    "twitter": {
      "max_length": 280,
      "hashtags": ["#CelesteAI", "#KusanagiAbyss", "#VTuberEN"],
      "emoji_usage": "1-2 per sentence"
    }
  }
}
```

### Benefits
- ✅ **No Code Changes**: Update templates via JSON
- ✅ **Easy Extension**: Add new content types
- ✅ **Platform Support**: Configure platform-specific settings
- ✅ **Maintainable**: Clear separation of data and logic

## Bot Integration

The CLI supports Discord and Twitch bot integration with proper user isolation:

### User Isolation
```bash
# Discord bot integration
CELESTE_USER_ID="discord_user_123" CELESTE_PLATFORM="discord" ./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync

# Twitch bot integration  
CELESTE_USER_ID="twitch_user_456" CELESTE_PLATFORM="twitch" ./celestecli --type tweet --game "NIKKE" --tone "chaotic" --sync
```

### Override Functionality
```bash
# PGP-signed override commands
CELESTE_OVERRIDE_ENABLED="true" CELESTE_PGP_SIGNATURE="kusanagi-abyss-override" ./celestecli --type tweet --game "NIKKE" --tone "explicit"
```

### Environment Variables
- `CELESTE_USER_ID` - User ID for conversation tracking
- `CELESTE_PLATFORM` - Platform (discord, twitch, cli)
- `CELESTE_OVERRIDE_ENABLED` - Enable override mode (true/false)
- `CELESTE_PGP_SIGNATURE` - PGP signature for override commands

## OpenSearch Integration

The CLI can upload conversations to DigitalOcean Spaces for OpenSearch RAG:

### Data Structure
```json
{
  "id": "conversation_id",
  "timestamp": "2024-01-01T00:00:00Z",
  "content_type": "tweet",
  "tone": "teasing",
  "game": "NIKKE",
  "persona": "celeste_stream",
  "prompt": "user_prompt",
  "response": "ai_response",
  "intent": "social_media",
  "purpose": "tweet",
  "topics": ["nikke", "gaming"],
  "sentiment": "positive",
  "platform": "twitter",
  "tags": ["celeste", "ai", "content"],
  "context": "Game: NIKKE, Tone: teasing, Persona: celeste_stream",
  "success": true
}
```

### S3 Path Structure
```
s3://whykusanagi/celeste/conversations/
├── 1760832573516177000.json
├── 1760832573516177001.json
└── ...
```

## Error Handling

The CLI provides comprehensive error handling:

- **Missing Credentials**: Clear error messages for missing API keys
- **Network Issues**: Retry logic for API requests
- **Invalid Responses**: Graceful handling of malformed responses
- **Configuration Errors**: Helpful error messages for config issues

## Development

### Project Structure
```
celesteCLI/
├── main.go                 # Main application
├── scaffolding.go         # Scaffolding logic
├── scaffolding.json       # External prompt templates
├── personality.yml        # Personality configuration
├── go.mod                 # Dependencies
├── go.sum                 # Dependency checksums
└── README.md              # This file
```

### Adding New Content Types

1. **Update `scaffolding.json`**:
```json
{
  "content_types": {
    "new_type": {
      "description": "Description of new content type",
      "scaffold": "Prompt template for new content type",
      "max_length": 280,
      "platform": "twitter"
    }
  }
}
```

2. **Update help menu in `main.go`** (if needed)
3. **Test the new content type**

### Adding New Platforms

1. **Update `scaffolding.json`**:
```json
{
  "platforms": {
    "new_platform": {
      "max_length": 500,
      "hashtags": ["#CelesteAI"],
      "emoji_usage": "1-2 per sentence"
    }
  }
}
```

2. **Update platform detection logic** (if needed)

## Troubleshooting

### Common Issues

1. **Missing API Key**
   ```
   Missing CELESTE_API_ENDPOINT or CELESTE_API_KEY
   ```
   **Solution**: Set environment variables or update `~/.celesteAI`

2. **Venice.ai Configuration Error**
   ```
   Venice.ai configuration error: missing Venice.ai API key
   ```
   **Solution**: Set `VENICE_API_KEY` or add `venice_api_key` to `~/.celesteAI`

3. **S3 Upload Failed**
   ```
   Warning: Failed to upload conversation to S3
   ```
   **Solution**: Check DigitalOcean Spaces credentials in `~/.celeste.cfg`

### Debug Mode
Use `--debug` flag to see raw API responses:
```bash
./celestecli --type tweet --debug
```

## License

This project is part of the CelesteAI ecosystem and is intended for use with the CelesteAI agent.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Verify configuration files
3. Test with debug mode
4. Check API endpoint status