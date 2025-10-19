# NSFW Mode with Venice.ai

## Overview

Celeste CLI now supports **NSFW mode** using Venice.ai for uncensored content generation. This mode bypasses content filters and allows for unrestricted content creation.

## Configuration

### 1. Venice.ai API Key

Add your Venice.ai API key to `~/.celesteAI`:

```bash
# Add to ~/.celesteAI
venice_api_key=your_venice_api_key_here
```

Or set as environment variable:
```bash
export VENICE_API_KEY=your_venice_api_key_here
```

### 2. Optional Venice.ai Configuration

You can customize Venice.ai settings in `~/.celesteAI`:

```bash
# Venice.ai Configuration (optional)
venice_base_url=https://api.venice.ai/api/v1
venice_model=venice-uncensored
venice_upscaler=upscaler
```

## Usage

### Basic NSFW Mode

```bash
./celestecli --nsfw --type tweet --tone "lewd" --game "NIKKE"
```

### NSFW Content Types

All content types work with NSFW mode:

```bash
# NSFW Twitter posts
./celestecli --nsfw --type tweet --tone "explicit" --game "NIKKE"

# NSFW TikTok captions
./celestecli --nsfw --type tiktok --tone "suggestive" --game "NIKKE"

# NSFW YouTube descriptions
./celestecli --nsfw --type ytdesc --tone "adult" --game "NIKKE"

# NSFW Discord announcements
./celestecli --nsfw --type discord --tone "lewd" --game "NIKKE"
```

## Venice.ai Models

### Available Models:

1. **`venice-uncensored`** (Default)
   - No content filtering
   - Full NSFW content generation
   - Text generation only

2. **`lustify-sdxl`**
   - Uncensored image generation
   - SDXL-based model
   - High-quality NSFW images

3. **`wai-Illustrious`**
   - Anime-style generation
   - Uncensored anime content
   - Perfect for VTuber content

4. **`upscaler`**
   - Image upscaling service
   - 2x ($0.02) or 4x ($0.08) upscaling
   - Enhances generated images

## Pricing

- **Text Generation**: Varies by model
- **Image Generation**: Varies by model
- **Upscaling**: $0.01 base + 2x ($0.02) or 4x ($0.08)

## Examples

### NSFW Tweet Generation
```bash
./celestecli --nsfw --type tweet --tone "explicit" --game "NIKKE" --context "Character is in a suggestive pose"
```

### NSFW TikTok Caption
```bash
./celestecli --nsfw --type tiktok --tone "lewd" --game "NIKKE" --context "Dance video with suggestive movements"
```

### NSFW YouTube Description
```bash
./celestecli --nsfw --type ytdesc --tone "adult" --game "NIKKE" --context "18+ content warning"
```

## Safety Notes

⚠️ **Important**: NSFW mode generates uncensored content. Use responsibly:

- Only use for legitimate content creation
- Respect platform guidelines
- Consider your audience
- Use appropriate content warnings

## Configuration Priority

1. **Environment Variables** (highest priority)
   - `VENICE_API_KEY`

2. **Configuration File** (fallback)
   - `~/.celesteAI` file

## Error Handling

If Venice.ai configuration is missing:
```
Venice.ai configuration error: missing Venice.ai API key. Set VENICE_API_KEY environment variable or venice_api_key in ~/.celesteAI
```

## Integration with Existing Features

NSFW mode works with all existing CLI features:

- ✅ All content types (`tweet`, `tiktok`, `ytdesc`, etc.)
- ✅ All tone options (`lewd`, `explicit`, `suggestive`, etc.)
- ✅ Game context (`--game`)
- ✅ Media context (`--media`)
- ✅ Additional context (`--context`)
- ✅ Debug mode (`--debug`)
- ❌ Sync mode (`--sync`) - Not supported with Venice.ai

## Troubleshooting

### Missing API Key
```bash
# Set environment variable
export VENICE_API_KEY=your_key_here

# Or add to ~/.celesteAI
echo "venice_api_key=your_key_here" >> ~/.celesteAI
```

### API Errors
- Check your Venice.ai API key
- Verify your account has credits
- Check Venice.ai service status

### Content Issues
- NSFW mode bypasses all content filters
- Generated content may be explicit
- Use appropriate content warnings
