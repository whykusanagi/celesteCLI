# CelesteCLI - CelesteAI Command Line Interface

A Go-based CLI tool for interacting with CelesteAI, a mischievous demon noble VTuber assistant. This tool provides various content generation capabilities while maintaining Celeste's distinctive personality and voice.

## Features

### Content Generation
- **Twitter/X Posts** - Generate tweets in Celeste's voice with proper hashtags and tone
- **YouTube Descriptions** - Create detailed video descriptions with game metadata from IGDB
- **Stream Titles** - Generate punchy, chaotic stream titles
- **Discord Announcements** - Create stream announcements for Discord
- **Pixiv Posts** - Generate artistic captions for illustrations
- **Skeb Requests** - Draft professional commission requests
- **Tarot Readings** - Generate mystical tarot readings with Celtic or three-card spreads
- **Goodnight Messages** - Create flirty or cozy goodnight tweets

### Advanced Features
- **IGDB Integration** - Automatic game metadata retrieval and caching
- **Local Caching** - Conversation history and game data caching
- **Personality Switching** - Dynamic persona selection based on context
- **Behavior Scoring** - Response quality assessment
- **Emote RAG** - 7TV emote integration for enhanced responses
- **OpenSearch Sync** - Upload conversations to Celeste's knowledge base

## Installation

```bash
go build -o celestecli main.go
```

## Configuration

Create a `~/.celesteAI` config file with:
```
endpoint=https://your-celeste-api-endpoint
api_key=your-api-key
client_id=your-igdb-client-id
secret=your-igdb-client-secret
```

Or set environment variables:
- `CELESTE_API_ENDPOINT`
- `CELESTE_API_KEY`
- `CELESTE_IGDB_CLIENT_ID`
- `CELESTE_IGDB_CLIENT_SECRET`

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
```

### Advanced Options

```bash
# Add media context
./celestecli --type tweet --tone "lewd" --media "https://example.com/image.jpg"

# Include additional context
./celestecli --type pixivpost --game "Celeste" --context "wearing bunny outfit at party"

# Enable debug mode
./celestecli --type tweet --debug

# Use three-card tarot spread
./celestecli --type tarot --spread three
```

## Personality System

The CLI supports multiple personality modes based on the `personality.yml` configuration:

- **celeste_stream** - Default mischievous, teasing mode
- **celeste_ad_read** - Promotional content with sponsor mentions
- **celeste_moderation_warning** - Playful but firm moderation responses

## Content Guidelines

### Tone Options
- `lewd` - Suggestive and flirty
- `teasing` - Playful and mischievous
- `chaotic` - High energy and unpredictable
- `cute` - Sweet and endearing
- `official` - Professional but still in character
- `dramatic` - Theatrical and over-the-top
- `parody` - Humorous and satirical
- `funny` - Comedic and lighthearted
- `sweet` - Warm and affectionate

### Platform Compliance
- **Twitter/X**: Max 280 characters, 1-2 emojis per sentence
- **YouTube**: Detailed descriptions with proper formatting
- **Discord**: Short announcements with emojis
- **Pixiv**: Artistic, aesthetic-focused captions
- **Skeb**: Professional, respectful commission requests

## Caching System

The CLI maintains local caches for:
- **Game Metadata** - IGDB data cached in `~/.cache/celesteCLI/cache.json`
- **Conversation History** - Chat logs stored in `~/.cache/celesteCLI/celeste-cli.log`
- **Usage Statistics** - Token usage and performance metrics

## OpenSearch Integration

Conversations can be uploaded to Celeste's OpenSearch database for:
- Context retrieval in future interactions
- Behavior pattern analysis
- Response quality improvement
- Knowledge base expansion

## Error Handling

The CLI includes robust error handling with:
- Retry logic with exponential backoff
- Circuit breaker pattern for API failures
- Graceful degradation when services are unavailable
- Detailed error logging and debugging

## Development

### Project Structure
```
celesteCLI/
├── main.go              # Main CLI application
├── personality.yml      # CelesteAI personality specification
├── celeste_api_prompt.json  # API prompt configuration
├── go.mod              # Go module definition
└── README.md           # This file
```

### Adding New Content Types

1. Add new case to the switch statement in `main()`
2. Define appropriate prompt scaffolding
3. Update help text and documentation
4. Test with various tone combinations

### Personality Customization

Modify `personality.yml` to adjust:
- Response patterns and archetypes
- Platform-specific guidelines
- Content safety boundaries
- Emote usage policies

## API Endpoints

The CLI communicates with CelesteAI through these endpoints:
- `POST /v1/chat/completions` - Main chat completion
- `POST /v1/tts` - Text-to-speech generation
- `POST /v1/events` - Event logging
- `POST /v1/behavior` - Behavior scoring
- `POST /v1/rag/emotes` - Emote retrieval
- `GET /v1/meta/vndb` - VNDB metadata
- `GET /v1/meta/igdb` - IGDB metadata

## Contributing

1. Follow Go best practices
2. Maintain Celeste's personality consistency
3. Test all content types with various tones
4. Update documentation for new features
5. Ensure platform compliance

## License

Private project for Kusanagi's CelesteAI system.

## Support

For issues or feature requests, contact the maintainer or check the project documentation.
