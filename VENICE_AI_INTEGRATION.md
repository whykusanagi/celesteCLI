# Venice.ai Integration Guide

## Overview

The CelesteCLI now supports full Venice.ai integration for NSFW image generation and upscaling using the actual Venice.ai API endpoints.

## Features

### 🎨 Image Generation
- **Model Support**: `lustify-sdxl`, `animewan`, and other Venice.ai models
- **NSFW Content**: Uncensored image generation
- **High Quality**: 1024x1024 resolution by default
- **Customizable**: Full control over generation parameters

### 🔍 Image Upscaling
- **Local Images**: Upscale images from your local filesystem
- **Base64 Encoding**: Automatic image encoding for API
- **Quality Enhancement**: AI-powered upscaling with creativity controls
- **Multiple Scales**: 2x, 4x upscaling support

## Configuration

### Venice.ai API Setup

Create or update your `~/.celesteAI` file:

```bash
# Venice.ai Configuration
venice_api_key=your_venice_api_key_here
venice_base_url=https://api.venice.ai/api/v1
venice_model=lustify-sdxl
venice_upscaler=upscaler
```

### Available Models

You can use any Venice.ai model by setting `venice_model`:

- **`lustify-sdxl`** - Uncensored SDXL model for NSFW content
- **`animewan`** - Anime-style generation
- **`hidream`** - High-quality dream-like images
- **`wai-Illustrious`** - Illustrious anime model

## Usage Examples

### Image Generation

#### Basic NSFW Image Generation
```bash
celestecli --nsfw --image --tone "explicit" --context "Generate detailed NSFW image of Celeste with specific physical details, clothing, pose, and composition"
```

#### Game-Themed NSFW Images
```bash
celestecli --nsfw --image --game "NIKKE" --tone "lewd" --context "Celeste in NIKKE-themed NSFW pose with detailed anatomical descriptions and game-specific elements"
```

#### Anime-Style Images
```bash
# Set model to animewan in ~/.celesteAI
celestecli --nsfw --image --tone "explicit" --context "Generate anime-style NSFW image of Celeste with detailed anime art style"
```

#### Maximum Detail Images
```bash
celestecli --nsfw --image --tone "explicit" --context "Very detailed NSFW image description of Celeste including: specific body measurements, clothing details, pose composition, lighting, facial expressions, skin texture, hair details, eye color, and any other visual elements needed for high-quality image generation"
```

### Image Upscaling

#### Basic Upscaling
```bash
celestecli --nsfw --upscale --image-path "/path/to/your/image.jpg"
```

#### Upscaling with Enhancement
```bash
celestecli --nsfw --upscale --image-path "/path/to/your/image.jpg" --context "Enhance with gold lighting and high detail"
```

## API Parameters

### Image Generation Parameters

The CLI automatically sets these parameters for optimal results:

```json
{
  "cfg_scale": 7.5,
  "embed_exif_metadata": false,
  "format": "webp",
  "height": 1024,
  "hide_watermark": false,
  "model": "lustify-sdxl",
  "negative_prompt": "blurry, low quality, distorted",
  "prompt": "your_prompt_here",
  "return_binary": false,
  "variants": 1,
  "safe_mode": false,
  "steps": 20,
  "width": 1024
}
```

### Upscaling Parameters

```json
{
  "enhance": true,
  "enhanceCreativity": 0.5,
  "enhancePrompt": "high quality, detailed, sharp",
  "image": "base64_encoded_image",
  "scale": 2
}
```

## Response Format

### Image Generation Response
```json
{
  "images": [
    {
      "url": "https://venice.ai/generated/image.jpg",
      "seed": 123456789,
      "model": "lustify-sdxl"
    }
  ],
  "status": "success"
}
```

### Upscaling Response
```json
{
  "upscaled_image": "https://venice.ai/upscaled/image.jpg",
  "original_size": "512x512",
  "upscaled_size": "1024x1024",
  "scale": 2
}
```

## Advanced Usage

### Custom Model Selection

To use different models, update your `~/.celesteAI` file:

```bash
# For anime-style images
venice_model=animewan

# For high-quality dream images
venice_model=hidream

# For illustrious anime
venice_model=wai-Illustrious
```

### Batch Processing

You can process multiple images by creating a script:

```bash
#!/bin/bash
# Batch upscale script
for image in /path/to/images/*.jpg; do
    celestecli --nsfw --upscale --image-path "$image"
done
```

### Integration with Other Tools

The CLI returns image URLs that can be used with other tools:

```bash
# Generate image and download
IMAGE_URL=$(celestecli --nsfw --image --tone "explicit" --context "Celeste NSFW image")
curl -o celeste_image.jpg "$IMAGE_URL"
```

## Error Handling

### Common Issues

1. **Missing API Key**
   ```
   Error: missing Venice.ai API key. Set VENICE_API_KEY environment variable or venice_api_key in ~/.celesteAI
   ```

2. **Invalid Image Path**
   ```
   Error: failed to read image file: open /path/to/image.jpg: no such file or directory
   ```

3. **API Rate Limits**
   ```
   Venice.ai API error: rate limit exceeded
   ```

### Troubleshooting

1. **Check API Key**: Ensure your Venice.ai API key is valid
2. **Verify Model**: Make sure the model name is correct
3. **Image Format**: Supported formats: JPG, PNG, WEBP
4. **File Size**: Check if image file is too large for API limits

## Security Considerations

### NSFW Content
- **Uncensored Generation**: Uses Venice.ai's uncensored models
- **Local Processing**: Images are processed locally before upload
- **API Security**: All requests use HTTPS with Bearer token authentication

### Data Privacy
- **No Local Storage**: Generated images are not stored locally
- **API Only**: All processing happens on Venice.ai servers
- **Temporary URLs**: Generated image URLs may expire

## Performance Tips

### Optimization
1. **Image Size**: Use appropriate image sizes for upscaling
2. **Prompt Length**: Longer prompts may take more time
3. **Model Selection**: Some models are faster than others
4. **Batch Processing**: Process multiple images in sequence

### Best Practices
1. **Test with Small Images**: Start with smaller images for testing
2. **Use Specific Prompts**: More specific prompts yield better results
3. **Monitor API Usage**: Keep track of your Venice.ai API usage
4. **Save Results**: Download and save generated images locally

## Examples

### Complete Workflow

```bash
# 1. Generate NSFW image
celestecli --nsfw --image --tone "explicit" --context "Detailed NSFW image of Celeste" > image_url.txt

# 2. Download the image
IMAGE_URL=$(cat image_url.txt | grep -o 'https://[^"]*')
curl -o celeste_generated.jpg "$IMAGE_URL"

# 3. Upscale the image
celestecli --nsfw --upscale --image-path "celeste_generated.jpg" > upscaled_url.txt

# 4. Download upscaled version
UPSCALED_URL=$(cat upscaled_url.txt | grep -o 'https://[^"]*')
curl -o celeste_upscaled.jpg "$UPSCALED_URL"
```

### Integration with Discord Bot

```bash
# Generate image for Discord bot
CELESTE_USER_ID="discord_user_123" \
CELESTE_PLATFORM="discord" \
celestecli --nsfw --image --tone "explicit" --context "Discord NSFW image of Celeste"
```

## Conclusion

The Venice.ai integration provides powerful NSFW image generation and upscaling capabilities directly from the command line. With support for multiple models and full API parameter control, you can generate high-quality uncensored content for various use cases.

**Remember to use responsibly and in accordance with your local laws and platform terms of service.**
