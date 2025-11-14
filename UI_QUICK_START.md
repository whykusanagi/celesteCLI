# CelesteCLI UI System - Quick Start Guide

## Building the New Version

```bash
# Build with UI module
go build -o celestecli main.go scaffolding.go animation.go ui.go

# Install (optional)
cp celestecli ~/.local/bin/
chmod +x ~/.local/bin/celestecli
```

## Using the UI Module in Code

### 1. Basic Messages

```go
// Information message (cyan)
PrintMessage(INFO, "Loading configuration...")

// Success message (green)
PrintMessage(SUCCESS, "Configuration loaded!")

// Warning message (yellow)
PrintMessage(WARN, "Using fallback settings")

// Error message (red)
PrintMessage(ERROR, "Failed to load config")

// Debug message (cyan)
PrintMessage(DEBUG, "Parsed 5 configuration entries")
```

### 2. Formatted Messages

```go
// Printf-style formatting
PrintMessagef(INFO, "Processing %d tweets...", count)
PrintMessagef(SUCCESS, "Generated %s in %.2fs", filename, duration.Seconds())
```

### 3. Progress Phases

```go
// Show multi-step progress
PrintPhase(1, 3, "Step 1: Loading...")
time.Sleep(500 * time.Millisecond)

PrintPhase(2, 3, "Step 2: Processing...")
time.Sleep(500 * time.Millisecond)

PrintPhase(3, 3, "Step 3: Complete!")

// Or simple single status
PrintPhaseSimple("done", "Configuration loaded")
PrintPhaseSimple("active", "Generating content...")
PrintPhaseSimple("pending", "Waiting for response...")
```

### 4. Success with Metrics

```go
duration := time.Now().Sub(startTime)
metadata := map[string]string{
    "tokens": "287",
    "cost": "$0.02",
    "platform": "twitter",
}
PrintSuccess("Content generated", duration, metadata)

// Output: ✅ Content generated in 1.23s | tokens: 287 | cost: $0.02 | platform: twitter
```

### 5. Errors

```go
// Simple error with hint
err := loadConfig()
if err != nil {
    PrintError("Configuration", err, "Check ~/.celesteAI exists")
}

// Or error box with multiple hints
PrintErrorBox(
    "Missing API Key",
    "CELESTE_API_KEY not found",
    []string{
        "Set environment variable: export CELESTE_API_KEY=...",
        "Or add to ~/.celesteAI: api_key=...",
    },
    "https://docs.celestecli.io/setup",
)
```

### 6. Configuration Display

```go
config := map[string]string{
    "Format": "short",
    "Platform": "twitter",
    "Tone": "teasing",
    "Mode": "NORMAL",
}
PrintConfig(config)
```

### 7. Visual Separators

```go
PrintSeparator(HEAVY)      // ═══════════════════════
PrintSeparator(LIGHT)      // ───────────────────────
PrintSeparator(DASHED)     // ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
PrintSeparator(CORRUPTED)  // ≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈≈

// With text
PrintSeparatorWithText(HEAVY, "✨ Content Ready ✨")
```

### 8. Response Ready

```go
// Before outputting final response
PrintResponseReady()
fmt.Println(generatedContent)
```

### 9. Mode and Headers

```go
PrintHeader(NORMAL, map[string]string{
    "Platform": "twitter",
    "Format": "short",
})

PrintModeIndicator(TAROT)
// Output: [TAROT] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 10. Configuration Status

```go
checks := map[string]bool{
    "API Key": true,
    "S3 Config": false,
    "Twitter": true,
}
PrintConfigStatus(checks)
// Output:
// Configuration Status:
//   ✓ API Key
//   ✗ S3 Config
//   ✓ Twitter
```

## Animations

### Demonic Eye Animation

```go
import (
    "context"
    "time"
)

// Start animation
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startDemonicEyeAnimation(ctx, done, os.Stderr)

// Do work...
PrintMessage(INFO, "Generating response...")
time.Sleep(2 * time.Second)

// Stop animation
cancel()
<-done  // Wait for cleanup

// Show result
PrintResponseReady()
fmt.Println("✨ Generated response ✨")
```

### Processing Indicator

```go
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startProcessingIndicator(ctx, done, os.Stderr, "Fetching tweets...")

// Do work...
tweets := downloadUserTweets(username, count)

cancel()
<-done

PrintMessage(SUCCESS, "Downloaded "+strconv.Itoa(len(tweets))+" tweets")
```

## Common Patterns

### Pattern 1: Simple Operation

```go
PrintMessage(INFO, "Starting operation...")

// Do work
result := doWork()

PrintSuccess("Operation", duration, map[string]string{
    "result": result,
})
```

### Pattern 2: Multi-Step Operation

```go
PrintHeader(NORMAL, map[string]string{"Task": "Content Generation"})

PrintPhase(1, 4, "Loading...")
loadConfig()

PrintPhase(2, 4, "Preparing...")
preparePrompt()

PrintPhase(3, 4, "Generating...")
ctx, cancel := context.WithCancel(context.Background())
done := make(chan bool)
startDemonicEyeAnimation(ctx, done, os.Stderr)

response := generateContent()

cancel()
<-done

PrintPhase(4, 4, "Complete!")

PrintResponseReady()
fmt.Println(response)
```

### Pattern 3: Error Recovery

```go
config, err := loadConfig()
if err != nil {
    PrintErrorBox(
        "Configuration Failed",
        err.Error(),
        []string{
            "Check file: ~/.celesteAI",
            "Format should be: key=value",
            "One setting per line",
        },
        "https://docs.celestecli.io/config",
    )

    // Try fallback
    PrintMessage(WARN, "Using default configuration...")
    config = getDefaultConfig()
}

PrintMessage(SUCCESS, "Configuration loaded")
```

### Pattern 4: Status Tracking

```go
checks := map[string]bool{}

// Check each requirement
if _, err := os.Stat(configPath); err == nil {
    checks["Config File"] = true
} else {
    checks["Config File"] = false
}

if apiKey := os.Getenv("CELESTE_API_KEY"); apiKey != "" {
    checks["API Key"] = true
} else {
    checks["API Key"] = false
}

PrintConfigStatus(checks)

// Exit if critical check failed
if !checks["API Key"] {
    PrintMessage(ERROR, "Cannot proceed without API key")
    os.Exit(1)
}
```

## Color Reference

```go
// Basic colors
ColorRed       = "\033[31m"      // 31m
ColorGreen     = "\033[32m"      // 32m
ColorYellow    = "\033[33m"      // 33m
ColorBlue      = "\033[34m"      // 34m
ColorMagenta   = "\033[35m"      // 35m
ColorCyan      = "\033[36m"      // 36m

// Bright colors (bold)
ColorBrightRed    = "\033[91m"   // 91m
ColorBrightGreen  = "\033[92m"   // 92m
ColorBrightYellow = "\033[93m"   // 93m

// Text styles
Bold      = "\033[1m"            // Bold
Italic    = "\033[3m"            // Italic
Underline = "\033[4m"            // Underline
```

## Tips & Tricks

### Tip 1: Mode-Aware Colors
```go
color := getColorForMode(TAROT)  // Returns bright magenta
fmt.Fprintf(os.Stderr, "%sCustom colored text%s\n", color, ColorDefault)
```

### Tip 2: Message Type Colors
```go
color := getColorForMessageType(SUCCESS)  // Returns green
fmt.Fprintf(os.Stderr, "%sSuccess in color!%s\n", color, ColorDefault)
```

### Tip 3: Wrapping Text
```go
wrapped := wrapText("Long message here...", 50)
for _, line := range wrapped {
    PrintMessage(INFO, line)
}
```

### Tip 4: Custom Padded Output
```go
padded := padRight("Status", 20)  // Pads to 20 chars
fmt.Println("[" + padded + "]")
```

## Quick Reference

```
PrintMessage(type, msg)
PrintMessagef(type, format, args...)
PrintPhase(current, total, text)
PrintPhaseSimple(status, text)
PrintSuccess(op, duration, metadata)
PrintError(op, err, hint)
PrintErrorBox(title, error, hints[], docLink)
PrintConfig(config)
PrintHeader(mode, details)
PrintSeparator(style)
PrintSeparatorWithText(style, text)
PrintResponseReady()
PrintModeIndicator(mode)
PrintConfigStatus(checks)

startDemonicEyeAnimation(ctx, done, output)
startProcessingIndicator(ctx, done, output, msg)
```

## Building Real Integration

Want to integrate UI into main.go? Start here:

1. Replace `fmt.Fprintf(os.Stderr, "Error: %v\n", err)` with `PrintError("Operation", err, "")`
2. Replace `fmt.Fprintf(os.Stderr, "Loading...")`  with `PrintPhase(1, 3, "Loading...")`
3. Add `PrintConfig(config)` at operation start
4. Add eye animation during long operations
5. Use `PrintResponseReady()` before output

That's it! The UI system will handle all the formatting and colors automatically.

---

See `UI_INTEGRATION_GUIDE.md` for complete API documentation.
