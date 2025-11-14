# CelesteCLI UI/UX Integration Guide

This guide explains the new premium UI system added to CelesteCLI and how to use the new `ui.go` module in your code.

## Overview

The UI module (`ui.go`) provides a comprehensive system for displaying messages, progress indicators, and formatted output that maintains the premium Apple-quality design while preserving the demonic/corrupted abyss aesthetic.

## Components

### 1. Message Types

There are five standardized message types with associated emojis and colors:

```go
const (
    INFO    MessageType = "INFO"     // 📋 Cyan
    SUCCESS MessageType = "SUCCESS"  // ✅ Green
    WARN    MessageType = "WARN"     // ⚠️  Yellow
    ERROR   MessageType = "ERROR"    // ❌ Bright Red
    DEBUG   MessageType = "DEBUG"    // 🔍 Cyan
)
```

### 2. Operation Modes

Different CLI modes have distinct visual themes:

```go
const (
    NORMAL    OperationMode = "NORMAL"     // Default cyan
    TAROT     OperationMode = "TAROT"      // Bright magenta
    NSFW      OperationMode = "NSFW"       // Bright yellow
    TWITTER   OperationMode = "TWITTER"    // Bright blue
    STREAMING OperationMode = "STREAMING"  // Bright green
)
```

Each mode automatically selects appropriate colors for its context.

### 3. Separator Styles

Visual separators for organizing output:

```go
const (
    HEAVY    SeparatorStyle = "HEAVY"      // ═══ (double line)
    LIGHT    SeparatorStyle = "LIGHT"      // ─── (single line)
    DASHED   SeparatorStyle = "DASHED"     // ╌╌╌ (dashed)
    CORRUPTED SeparatorStyle = "CORRUPTED" // ≈≈≈ (wavy/corrupted)
)
```

## Core Functions

### PrintMessage

Displays a formatted message with type-specific emoji and color:

```go
PrintMessage(INFO, "Generating content...")
// Output: 📋 Generating content... (in cyan)

PrintMessage(SUCCESS, "Tweet posted!")
// Output: ✅ Tweet posted! (in green)

PrintMessage(ERROR, "API key missing!")
// Output: ❌ API key missing! (in bright red)
```

### PrintMessagef

Printf-style formatted messages:

```go
PrintMessagef(INFO, "Downloading %d tweets...", count)
// Output: 📋 Downloading 500 tweets... (in cyan)
```

### PrintPhase

Shows progress through operation steps:

```go
PrintPhase(1, 4, "Loading configuration...")
// Output: [✓ ● ○ ○] Loading configuration...

PrintPhase(3, 4, "Generating response...")
// Output: [✓ ✓ ✓ ● ○] Generating response...
```

### PrintSuccess

Displays success with operation metrics:

```go
metadata := map[string]string{
    "tokens": "145",
    "cost": "$0.12",
}
PrintSuccess("Content generated", duration, metadata)
// Output: ✅ Content generated in 2.34s | tokens: 145 | cost: $0.12
```

### PrintError

Shows errors with optional hints:

```go
PrintError("API Request", err, "Check your API key in ~/.celesteAI")
// Output:
// ❌ API Request: context deadline exceeded
// 💡 Hint: Check your API key in ~/.celesteAI
```

### PrintErrorBox

Displays formatted error boxes with resolution instructions:

```go
hints := []string{
    "1. Visit https://developer.twitter.com",
    "2. Generate a Bearer Token",
    "3. Add to ~/.celesteAI: twitter_bearer_token=...",
}
PrintErrorBox(
    "Missing Twitter Bearer Token",
    "Twitter API integration requires a Bearer Token",
    hints,
    "https://docs.celestecli.io/twitter",
)
```

Output:
```
╔══════════════════════════════════════════╗
║ ❌ Missing Twitter Bearer Token         ║
╟──────────────────────────────────────────╢
║ Twitter API integration requires a...    ║
║ HOW TO FIX:                             ║
║ 1. Visit https://developer.twitter.com   ║
║ 2. Generate a Bearer Token              ║
║ 3. Add to ~/.celesteAI: twitter_...     ║
║                                          ║
║ 📖 Docs: https://docs.celestecli...    ║
╚══════════════════════════════════════════╝
```

### PrintConfig

Displays active configuration in a header box:

```go
config := map[string]string{
    "Format": "short",
    "Platform": "twitter",
    "Tone": "teasing",
    "Mode": "NORMAL",
}
PrintConfig(config)
```

Output:
```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ Active Configuration                     ┃
┃   Format: short | Platform: twitter     ┃
┃   Tone: teasing | Mode: NORMAL          ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

### PrintHeader

Shows mode and operation details:

```go
details := map[string]string{
    "Platform": "twitter",
    "Format": "short",
}
PrintHeader(NORMAL, details)
// Output: [NORMAL] [Platform: twitter] [Format: short]
```

### PrintSeparator

Displays a visual separator:

```go
PrintSeparator(HEAVY)   // ════════════════════════════════════════════════════════
PrintSeparator(LIGHT)   // ────────────────────────────────────────────────────────
PrintSeparator(DASHED)  // ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
```

### PrintSeparatorWithText

Separator with centered text:

```go
PrintSeparatorWithText(HEAVY, "✨ Response Ready ✨")
// Output: ════════════ ✨ Response Ready ✨ ════════════
```

### PrintResponseReady

Premium visual indicator that response is complete:

```go
PrintResponseReady()
// Output:
//
// ════════════════ ✨ Response Ready ✨ ════════════════
//
```

### PrintModeIndicator

Color-coded mode indicator:

```go
PrintModeIndicator(TAROT)
// Output: [TAROT] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### PrintConfigStatus

Configuration validation display:

```go
checks := map[string]bool{
    "API Key Loaded": true,
    "S3 Configured": false,
    "Twitter API": true,
}
PrintConfigStatus(checks)
// Output:
// Configuration Status:
//   ✓ API Key Loaded
//   ✗ S3 Configured
//   ✓ Twitter API
```

## Animation Integration

### Demonic Eye Animation

The new `animation.go` includes eye-based animations for processing:

```go
// Start the demonic eye animation
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startDemonicEyeAnimation(ctx, done, os.Stderr)

// Do work here...

// Stop animation when complete
cancel()
<-done // Wait for animation to finish
```

The eye animation displays:
- Alternating frames (👁️, 👀, ◉◉, ●●)
- Different looking directions
- Color pulses (magenta and red)
- Corrupted text alongside the eye

### Processing Indicator

A simple braille spinner for lighter operations:

```go
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startProcessingIndicator(ctx, done, os.Stderr, "Generating content...")

// Do work...

cancel()
<-done
```

## Color Constants

The UI module includes ANSI color codes for custom styling:

```go
ColorRed       = "\033[31m"
ColorGreen     = "\033[32m"
ColorYellow    = "\033[33m"
ColorBlue      = "\033[34m"
ColorMagenta   = "\033[35m"
ColorCyan      = "\033[36m"

ColorBrightRed    = "\033[91m"
ColorBrightGreen  = "\033[92m"
ColorBrightYellow = "\033[93m"
ColorBrightBlue   = "\033[94m"
ColorBrightMagenta = "\033[95m"
ColorBrightCyan   = "\033[96m"

Bold      = "\033[1m"
Italic    = "\033[3m"
Underline = "\033[4m"
```

## Integration Examples

### Example 1: Content Generation

```go
// Show what we're doing
config := map[string]string{
    "Format": format,
    "Platform": platform,
    "Tone": tone,
}
PrintConfig(config)

// Show progress
PrintPhase(1, 3, "Loading personality...")
time.Sleep(500 * time.Millisecond)

PrintPhase(2, 3, "Building prompt...")
time.Sleep(500 * time.Millisecond)

// Start thinking animation
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startDemonicEyeAnimation(ctx, done, os.Stderr)

// Generate content
response := generateContent(prompt)

cancel()
<-done

// Show success
metadata := map[string]string{
    "tokens": strconv.Itoa(tokenCount),
    "platform": platform,
}
PrintSuccess("Content generated", duration, metadata)

PrintResponseReady()
fmt.Println(response)
```

### Example 2: Error Handling

```go
config, err := loadTwitterConfig()
if err != nil {
    hints := []string{
        "Set TWITTER_BEARER_TOKEN environment variable, or",
        "Add twitter_bearer_token=... to ~/.celesteAI",
    }
    PrintErrorBox(
        "Twitter Configuration Missing",
        err.Error(),
        hints,
        "https://docs.celestecli.io/twitter",
    )
    os.Exit(1)
}
```

### Example 3: Operation Phases

```go
PrintPhase(1, 5, "Connecting to API...")
if !connectAPI() {
    PrintMessage(ERROR, "Failed to connect")
    os.Exit(1)
}

PrintPhase(2, 5, "Loading configuration...")
config := loadConfig()

PrintPhase(3, 5, "Preparing request...")
req := buildRequest(config)

PrintPhase(4, 5, "Sending request...")
response := makeRequest(req)

PrintPhase(5, 5, "Processing response...")
result := parseResponse(response)

PrintMessage(SUCCESS, "All operations completed!")
```

## Building with the New UI Module

```bash
# Build with the new ui.go module
go build -o celestecli main.go scaffolding.go animation.go ui.go

# Install
cp celestecli ~/.local/bin/
chmod +x ~/.local/bin/celestecli
```

## Design Philosophy

The UI system is designed with these principles:

1. **Premium Quality**: Inspired by Apple's design language - clean, minimal, consistent
2. **Corrupted Aesthetic**: The abyss theme is preserved through colors, animations, and corrupted text
3. **Clarity**: Every message type is immediately recognizable
4. **Accessibility**: Color-blind friendly with emoji fallbacks
5. **Non-Intrusive**: Animations and messages never interfere with actual content output
6. **Responsive**: Works in TTY terminals and gracefully degrades in pipes/CI
7. **Helpful**: Errors provide guidance and documentation links

## Future Enhancements

Potential improvements for future versions:

- Terminal capability detection (256-color vs 16-color)
- Responsive box sizing based on terminal width
- JSON/YAML output format option
- Custom theme support
- Progress bars for long operations
- Nested phase indicators for complex workflows
- Sound alerts for completions/errors (optional)

## API Reference

For a complete API reference, see the `ui.go` file which includes inline documentation for all functions and types.
