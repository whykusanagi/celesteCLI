# Celeste Interactive Mode - Asset Display Demo

This document demonstrates how to use the embedded pixel art assets in the new interactive mode.

## Quick Start

### Launch Interactive Mode

```bash
./Celeste --interactive
```

Or with the installed binary:

```bash
Celeste --interactive
```

This launches an interactive chat session with Celeste that displays the embedded pixel art assets.

---

## What You'll See

### Welcome Screen

When you launch interactive mode, you'll see:

```
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║             Welcome to Celeste Interactive Mode                ║
║                                                                ║
║  Type 'help' for commands, 'exit' to quit                      ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝

    ┌─────────────────┐
    │   ✨ Celeste ✨  │
    │  (╯°□°)╯︵ ┻━┻  │
    └─────────────────┘

✅ Celeste is ready to chat~
```

The ASCII art representation of **pixel_wink** (Celeste winking) appears right away!

---

## Interactive Commands

### Display Assets

Show Celeste winking:
```
/show pixel_wink
```

Show Kusanagi abyss corruption:
```
/show kusanagi
```

Output:
```
    ┌─────────────────────┐
    │  🌑 Kusanagi Abyss 🌑│
    │    c0rrupt3d...    │
    │   深淵への堕落...    │
    └─────────────────────┘
```

### List Assets

```
/asset
```

Shows:
```
📦 Available Assets

  1. pixel_wink - Celeste pixel art winking
  2. kusanagi - Kusanagi/abyss themed artwork

📖 Docs: https://docs.Celeste.io
```

### Switch Visual Theme

Switch to friendly theme:
```
/theme normal
```

Switch to corrupted abyss theme:
```
/theme corrupted
```

The theme switch displays the corresponding asset!

### Change Operation Mode

```
/mode tarot        # Switch to 🔮 Tarot Reading Mode
/mode nsfw         # Switch to ⚡ NSFW Mode
/mode twitter      # Switch to 🐦 Twitter Mode
/mode normal       # Switch to ✨ Normal Mode
```

### View Status

```
/status
```

Shows current configuration:
```
Active Configuration
  Mode: Interactive Chat
  Theme: Corrupted Abyss
  Assets: 2 embedded
  Status: Ready
  Time: 14:32:45
```

### Clear Screen

```
/clear
```

Clears the terminal and redisplays Celeste's header.

### Show Help

```
/help
```

Shows all available commands.

---

## Regular Chat

Just type your message and press Enter:

```
Enter your message or command:
Tell me about Celeste
```

This will:
1. Show phase indicator: `[✓] Processing your message...`
2. Display the **demonic eye animation** showing Celeste thinking
3. Simulate a response generation
4. Display the response

Output example:
```
👁️   Celeste is thinking... c0rrupt1on d33ps...

[✓ ✓ ✓] Response ready!
Celeste: You said: Tell me about Celeste
```

---

## Asset Display Locations

### 1. On Startup
The **pixel_wink** asset displays when you first launch interactive mode.

### 2. When Switching Themes
- `/theme normal` → Shows pixel_wink (friendly Celeste)
- `/theme corrupted` → Shows kusanagi (abyss corruption)

### 3. When Displaying Specific Assets
- `/show pixel_wink` → Shows Celeste winking
- `/show kusanagi` → Shows abyss corruption

### 4. On Exit
When you type `exit` or `quit`, the **kusanagi** asset displays as a farewell.

---

## How It Works

The interactive mode uses the embedded pixel art assets from `assets.go`:

1. **asset.go** contains:
   - `pixel_wink` - 960x960 pixel art animation
   - `kusanagi` - 364x560 abyss-themed artwork

2. **interactive.go** functions:
   - `displayCelesteHeader()` - Shows pixel_wink on startup
   - `displayASCIIArtRepresentation()` - Converts GIFs to ASCII art for terminal display
   - `DisplayPixelArt()` - Main function to display assets
   - `showAsset()` - Command handler for `/show`

3. **animation.go** provides:
   - Demonic eye animation for thinking states
   - Processing indicators
   - Corruption text overlay

4. **ui.go** provides:
   - Color-coded messages
   - Progress phase indicators
   - Visual separators
   - Configuration display

---

## Build Instructions

Build with interactive mode:

```bash
go build -o Celeste main.go scaffolding.go animation.go ui.go assets.go interactive.go
```

Or install to PATH:

```bash
go build -o Celeste main.go scaffolding.go animation.go ui.go assets.go interactive.go
cp Celeste ~/.local/bin/
chmod +x ~/.local/bin/Celeste
```

---

## Example Session

Here's what a typical interactive session looks like:

```bash
$ Celeste --interactive

╔════════════════════════════════════════════════════════════════╗
║             Welcome to Celeste Interactive Mode                ║
║  Type 'help' for commands, 'exit' to quit                      ║
╚════════════════════════════════════════════════════════════════╝

    ┌─────────────────┐
    │   ✨ Celeste ✨  │
    │  (╯°□°)╯︵ ┻━┻  │
    └─────────────────┘

✅ Celeste is ready to chat~

📋 Enter your message or command:
Hey Celeste, what's up?

[✓] Processing your message...
👁️   Celeste is thinking... c0rrupt1on d33ps...

[✓ ✓ ✓] Response ready!
Celeste: You said: Hey Celeste, what's up?

📋 Enter your message or command:
/theme corrupted

✅ Switched to friendly theme

    ┌─────────────────────┐
    │  🌑 Kusanagi Abyss 🌑│
    │    c0rrupt3d...    │
    │   深淵への堕落...    │
    └─────────────────────┘

📋 Enter your message or command:
/help

📚 Celeste Interactive Commands
════════════════════════════════════════════════════════════════
  /show [pixel_wink|kusanagi]     Display pixel art asset
  /asset                          List all available assets
  /theme [normal|corrupted]       Switch visual theme
  /mode [tarot|nsfw|twitter|no... Change operation mode
  /status                         Show current status
  /clear                          Clear screen
  /help                           Show this help menu
  exit/quit                       Exit interactive mode
────────────────────────────────────────────────────────────────

📋 Enter your message or command:
exit

📋 Thanks for chatting with Celeste! Goodbye~

    ┌─────────────────────┐
    │  🌑 Kusanagi Abyss 🌑│
    │    c0rrupt3d...    │
    │   深淵への堕落...    │
    └─────────────────────┘
```

---

## Features

✅ **Embedded Assets** - GIF assets compiled directly into binary
✅ **ASCII Art Display** - Pixel art rendered as text for terminal compatibility
✅ **Theme Switching** - Display different assets based on mode
✅ **Rich UI** - Color-coded messages, progress indicators, animations
✅ **Thinking Animation** - Demonic eye animation during processing
✅ **Interactive Commands** - Full command set for asset control
✅ **User-Friendly** - Clear prompts and help system

---

## Integration Points

The interactive mode integrates:

1. **assets.go** - Embedded pixel art
2. **animation.go** - Eye animations and thinking indicators
3. **ui.go** - Color-coded messages and visual feedback
4. **interactive.go** - Interactive session management (NEW)

When you run `Celeste --interactive`, all these components work together to create a rich, visual terminal experience with embedded pixel art assets.

---

## Troubleshooting

### Assets not displaying?
- Make sure you built with: `go build ... assets.go interactive.go`
- Check that emoji/Unicode support is enabled in your terminal
- ASCII fallback should still show

### Colors not showing?
- Verify terminal supports ANSI colors (most modern terminals do)
- Try using a different terminal emulator
- The text-only version still works

### Commands not recognized?
- Type `/help` to see available commands
- Make sure to include the `/` prefix for commands
- Regular messages don't need a prefix

---

## Next Steps

The interactive mode provides:
- Beautiful terminal interface with embedded pixel art
- Real-time thinking animations
- Command-driven asset display
- Theme switching with visual feedback

You can now:
1. Launch the CLI with `Celeste --interactive`
2. See Celeste's pixel art on startup
3. Type messages or commands
4. Switch between themes and see different assets
5. Experience the full premium UI with animations

Enjoy chatting with Celeste! 🌙✨
