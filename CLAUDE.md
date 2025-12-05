# CLAUDE.md - AI Assistant Development Guide

This document provides comprehensive guidelines for AI assistants (like Claude) working on CelesteCLI development, testing, and validation.

## Table of Contents

- [Overview](#overview)
- [Feature Validation](#feature-validation)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)
- [Code Quality Checks](#code-quality-checks)
- [NSFW Mode Features](#nsfw-mode-features)
- [Venice.ai Integration](#veniceai-integration)

---

## Overview

CelesteCLI is a premium TUI-based CLI for interacting with LLM providers. It features:
- Multi-provider support (OpenAI, Venice.ai, Grok/xAI)
- 18+ AI-powered skills via function calling
- NSFW mode with uncensored image generation via Venice.ai
- Session persistence and conversation history
- Bubble Tea TUI with corruption-themed aesthetics

---

## Feature Validation

### Before Committing Code

Every feature implementation must pass these validation steps:

#### 1. Build Validation
```bash
# Clean build
go build -o celeste ./cmd/celeste

# Should complete without errors
# Binary should be ~15-20MB
```

#### 2. Code Quality Checks
```bash
# Format check (should return nothing)
gofmt -l ./cmd

# Vet check (should pass)
go vet ./...

# Mod tidy (ensure dependencies are clean)
go mod tidy
```

#### 3. Manual Testing Checklist

For each feature, create a test plan with:

**Example: NSFW Mode Image Generation**
- [ ] `/nsfw` command switches to Venice.ai endpoint
- [ ] Header shows "🔥 NSFW" indicator
- [ ] `/help` displays NSFW-specific help menu
- [ ] `image: prompt` generates image successfully
- [ ] `anime: prompt` uses wai-Illustrious model
- [ ] `dream: prompt` uses hidream model
- [ ] `/set-model wai-Illustrious` changes model
- [ ] Header shows "img:wai-Illustrious" after model change
- [ ] Generated images save to ~/Downloads
- [ ] Generated images are PNG format
- [ ] Image quality is high (40 steps, CFG 12.0)
- [ ] `/safe` returns to OpenAI mode
- [ ] Session persists NSFW state across restarts

**Example: Session Persistence**
- [ ] Start chat, send messages
- [ ] Exit application (Ctrl+C)
- [ ] Restart `celeste chat`
- [ ] Previous conversation restored
- [ ] Endpoint/model settings preserved
- [ ] NSFW mode state preserved

#### 4. Error Handling Validation

Test failure scenarios:

```bash
# Missing API key
unset OPENAI_API_KEY
celeste chat
# Should show clear error message

# Invalid config
echo "invalid json" > ~/.celeste/config.json
celeste chat
# Should handle gracefully

# Network failure
# Disconnect network, try generating image
# Should show timeout/network error
```

#### 5. Log Verification

Check logs for issues:

```bash
# View logs
tail -f ~/.celeste/logs/celeste_$(date +%Y-%m-%d).log

# Should show:
# - API calls with masked keys
# - Model selection logic
# - Error messages with context
# - Success confirmations
```

---

## Testing Requirements

### Manual Testing

CelesteCLI currently uses manual testing. For each PR, verify:

1. **Core Functionality**
   - Chat mode works
   - Skills execute correctly
   - Streaming displays properly
   - Input history navigation works

2. **Configuration**
   - Config loading from files
   - Environment variable overrides
   - Named configs work
   - Skills.json separation

3. **Provider Switching**
   - `/endpoint openai` works
   - `/endpoint venice` works
   - `/nsfw` switches to Venice
   - `/safe` returns to OpenAI
   - Models change correctly

4. **Media Generation (NSFW Mode)**
   - Image generation with all models
   - Model shortcuts (anime:, dream:)
   - Custom model syntax (image[model]:)
   - /set-model command
   - Header displays current model
   - Downloads to correct directory

5. **UI/UX**
   - No screen flicker
   - Typing animation smooth
   - Skills panel updates
   - Status messages clear
   - Error formatting readable

### Automated Testing (Future)

When writing tests:

```go
// cmd/celeste/venice/media_test.go
func TestParseMediaCommand(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantType  string
        wantModel string
        wantMedia bool
    }{
        {
            name:      "anime shortcut",
            input:     "anime: magical girl",
            wantType:  "image",
            wantModel: "wai-Illustrious",
            wantMedia: true,
        },
        {
            name:      "custom model",
            input:     "image[hidream]: sunset",
            wantType:  "image",
            wantModel: "hidream",
            wantMedia: true,
        },
        {
            name:      "not a command",
            input:     "hello world",
            wantType:  "",
            wantModel: "",
            wantMedia: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mediaType, _, params, isMedia := ParseMediaCommand(tt.input)
            if isMedia != tt.wantMedia {
                t.Errorf("isMedia = %v, want %v", isMedia, tt.wantMedia)
            }
            if mediaType != tt.wantType {
                t.Errorf("mediaType = %v, want %v", mediaType, tt.wantType)
            }
            if model, ok := params["model"].(string); ok && model != tt.wantModel {
                t.Errorf("model = %v, want %v", model, tt.wantModel)
            }
        })
    }
}
```

---

## Documentation Standards

### README Updates

When adding features, update README.md sections:

1. **Features List** - Add bullet point with description
2. **Configuration** - Document new config options
3. **Usage Examples** - Show command examples
4. **Troubleshooting** - Add common issues

### Code Documentation

Every exported function must have doc comments:

```go
// GenerateImage generates an image using Venice.ai's full-featured endpoint.
// It uses the provided config for authentication and model selection.
// Parameters can override defaults like width, height, steps, and cfg_scale.
//
// Returns a MediaResponse with either a file path (for base64 images) or
// an error if generation fails. The response includes success status and
// any error messages from the API.
func GenerateImage(config Config, prompt string, params map[string]interface{}) (*MediaResponse, error) {
    // ...
}
```

### Help Text Standards

When updating `/help` output:

- Keep lines under 80 characters
- Use consistent formatting
- Group related commands
- Show examples for complex features
- Document default values
- Include tips and best practices

```
Example Format:

Media Generation Commands:
  image: <prompt>              Generate images with current model
                               Example: image: cyberpunk cityscape

Model Management:
  /set-model <model>           Set default image generation model
                               Example: /set-model wai-Illustrious
                               Run without args to see all models

Current Configuration:
  • Endpoint: Venice.ai (https://api.venice.ai/api/v1)
  • Image Model: Use /set-model to configure
  • Downloads: ~/Downloads
  • Quality: 40 steps, CFG 12.0, PNG format
```

---

## Code Quality Checks

### Before Every Commit

Run this checklist:

```bash
# 1. Format all code
gofmt -w ./cmd

# 2. Run vet
go vet ./...

# 3. Check for common issues
go mod tidy
go mod verify

# 4. Build successfully
go build -o celeste ./cmd/celeste

# 5. Test binary
./Celeste --help

# 6. Install and smoke test
cp Celeste ~/.local/bin/
celeste chat
# Type: "hello" and Ctrl+C to exit

# 7. Check logs for errors
tail -20 ~/.celeste/logs/celeste_$(date +%Y-%m-%d).log
```

### Code Review Standards

When reviewing code:

- **Error Handling**: Every error must be handled or explicitly ignored with `_ =`
- **Logging**: Important operations must log with context
- **Magic Numbers**: Use named constants
- **Duplication**: Extract repeated code into functions
- **Comments**: Explain why, not what
- **Naming**: Use descriptive names, avoid abbreviations

---

## NSFW Mode Features

### Architecture

NSFW mode is a complete mode switch that:
1. Changes endpoint from OpenAI to Venice.ai
2. Disables function calling (Venice uncensored doesn't support it)
3. Enables prefix-based media generation commands
4. Shows different help menu
5. Persists state in session

### Command Flow

```
User: /nsfw
  ↓
CommandResult with StateChange{NSFWMode: true, ImageModel: "lustify-sdxl"}
  ↓
AppModel.Update() handles CommandResultMsg
  ↓
- Sets m.nsfwMode = true
- Sets m.imageModel = "lustify-sdxl"
- Calls SwitchEndpoint("venice")
- Updates header with SetNSFWMode(true)
- Updates header with SetImageModel("lustify-sdxl")
- Persists session
  ↓
Header displays: 🔥 NSFW • img:lustify-sdxl
```

### Image Generation Flow

```
User: anime: magical girl with sword
  ↓
ParseMediaCommand() detects "anime:" prefix
  ↓
Returns: type="image", prompt="magical girl with sword", params={model: "wai-Illustrious"}
  ↓
GenerateMediaMsg sent with ImageModel from app state
  ↓
loadVeniceConfig() loads API key from skills.json
  ↓
Model priority: msg.ImageModel > veniceConfig.ImageModel > "lustify-sdxl"
  ↓
venice.GenerateImage() calls /image/generate endpoint
  ↓
Response saved to ~/Downloads/celeste_image_TIMESTAMP.png
  ↓
MediaResultMsg with Path shown in chat
```

### Testing NSFW Mode

Complete test flow:

```bash
# 1. Enter NSFW mode
celeste chat
/nsfw
# Verify: Header shows "🔥 NSFW • img:lustify-sdxl"

# 2. Test default model
image: test prompt
# Verify: Generates with lustify-sdxl, saves to ~/Downloads

# 3. Test anime shortcut
anime: anime test prompt
# Verify: Uses wai-Illustrious model

# 4. Test dream shortcut
dream: dream test prompt
# Verify: Uses hidream model

# 5. Test custom model syntax
image[venice-sd35]: custom model test
# Verify: Uses venice-sd35

# 6. Set explicit model
/set-model wai-Illustrious
# Verify: Header shows "img:wai-Illustrious"

# 7. Generate with set model
image: another test
# Verify: Uses wai-Illustrious (from header state)

# 8. Test help menu
/help
# Verify: Shows NSFW-specific help with model list

# 9. Return to safe mode
/safe
# Verify: Header shows OpenAI, skills available again

# 10. Persistence test
Ctrl+C
celeste chat
# Verify: Not in NSFW mode (because we ran /safe)

/nsfw
/set-model hidream
Ctrl+C
celeste chat
# Verify: Still in NSFW mode, header shows "img:hidream"
```

---

## Venice.ai Integration

### API Endpoints

CelesteCLI uses Venice.ai's full-featured endpoints:

**Chat** (NSFW mode):
- Endpoint: `/chat/completions` (OpenAI-compatible)
- Model: `venice-uncensored`
- Function calling: **Not supported** (sends 0 tools)

**Image Generation**:
- Endpoint: `/image/generate` (Venice-specific)
- Models: `lustify-sdxl`, `wai-Illustrious`, `hidream`, etc.
- Parameters: `width`, `height`, `steps`, `cfg_scale`, `format`

**Image Upscaling**:
- Endpoint: `/image/upscale`
- Parameters: `scale`, `enhance`, `enhanceCreativity`

### Configuration

Venice.ai configuration lives in `~/.celeste/skills.json`:

```json
{
  "venice_api_key": "your-key-here",
  "venice_base_url": "https://api.venice.ai/api/v1",
  "venice_model": "venice-uncensored",
  "venice_image_model": "lustify-sdxl",
  "downloads_dir": "~/Downloads"
}
```

### Image Quality Parameters

Default settings for high-quality generation:

```go
width: 1024         // 1-1280
height: 1024        // 1-1280
steps: 40           // 1-50 (more = higher quality)
cfg_scale: 12.0     // 0-20 (higher = stronger prompt adherence)
format: "png"       // jpeg/png/webp
safe_mode: false    // Disable NSFW blurring
variants: 1         // 1-4 images
```

### Error Handling

Common Venice.ai errors:

**400 - Bad Request**:
- Wrong parameters for endpoint
- Invalid model name
- Prompt too long (>1500 chars)

**401 - Unauthorized**:
- Invalid API key
- Expired key

**429 - Rate Limited**:
- Too many requests
- Wait and retry

**500 - Server Error**:
- Venice.ai service issue
- Retry after delay

### Logging Venice Requests

All Venice.ai requests should log:

```go
LogInfo(fmt.Sprintf("→ Starting %s generation with prompt: '%s'", mediaType, prompt))
LogInfo("Loading Venice config from skills.json")
LogInfo(fmt.Sprintf("✓ Loaded Venice config: baseURL=%s, imageModel=%s", config.BaseURL, config.ImageModel))
LogInfo(fmt.Sprintf("Using model: %s for %s generation", modelToUse, mediaType))
LogInfo(fmt.Sprintf("Calling Venice.ai API for %s generation", mediaType))

// On success:
LogInfo(fmt.Sprintf("✓ Media generation successful: URL=%s, Path=%s", response.URL, response.Path))

// On error:
LogInfo(fmt.Sprintf("❌ Media generation error: %v", err))
```

---

## Validation Checklist

Before marking a feature complete:

### Code Quality
- [ ] `gofmt -l ./cmd` returns nothing
- [ ] `go vet ./...` passes
- [ ] `go build` succeeds
- [ ] No hardcoded secrets or API keys
- [ ] Error messages are helpful
- [ ] Logging is comprehensive

### Documentation
- [ ] README.md updated with new feature
- [ ] Help text updated (if user-facing)
- [ ] Code comments on exported functions
- [ ] CHANGELOG.md entry added

### Testing
- [ ] Manual testing completed (checklist above)
- [ ] Happy path works
- [ ] Error cases handled gracefully
- [ ] Logs contain useful debug info
- [ ] No regressions in existing features

### User Experience
- [ ] Command syntax is intuitive
- [ ] Error messages are actionable
- [ ] Help text is clear
- [ ] Examples provided
- [ ] Edge cases handled

### Git
- [ ] Commit message follows conventions
- [ ] Changes are focused (one feature)
- [ ] No unrelated changes
- [ ] Binary not committed (in .gitignore)

---

## Common Pitfalls

### 1. Not Reading Files Before Editing

Always use `Read` tool before `Edit` or `Write`. The tools will fail otherwise.

### 2. Not Rebuilding After Changes

After editing code:
```bash
go build -o celeste ./cmd/celeste
/bin/rm -f ~/.local/bin/Celeste
cp Celeste ~/.local/bin/Celeste
chmod +x ~/.local/bin/Celeste
```

### 3. Not Testing with Real API

Mock testing is good, but always test with real Venice.ai API to catch:
- Parameter validation issues
- Response format changes
- Rate limiting behavior
- Network timeouts

### 4. Not Checking Logs

Logs are in `~/.celeste/logs/celeste_YYYY-MM-DD.log`. Always check them after testing.

### 5. Forgetting Session Persistence

Test that state persists:
1. Change settings
2. Exit (Ctrl+C)
3. Restart
4. Verify settings retained

---

## Best Practices

1. **Incremental Development**: Small, testable changes
2. **Logging First**: Add logging before debugging
3. **Error Context**: Wrap errors with `fmt.Errorf("context: %w", err)`
4. **User Feedback**: Show progress and results clearly
5. **Graceful Degradation**: Handle missing configs/keys gracefully
6. **Documentation**: Update docs alongside code
7. **Git Hygiene**: Clear commits, descriptive messages

---

## Resources

- **Go Documentation**: https://go.dev/doc/
- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **Venice.ai Docs**: https://docs.venice.ai/
- **OpenAI API**: https://platform.openai.com/docs/api-reference

---

This document is a living guide. Update it as the project evolves!
