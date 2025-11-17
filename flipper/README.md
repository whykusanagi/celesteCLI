# Flipper Zero - Celeste CLI Remote Controller

This directory contains a Flipper Zero application that acts as a USB HID keyboard remote controller for the CelesteAI CLI tool.

## Overview

The Flipper Zero app allows you to:
- Navigate a menu of pre-configured Celeste CLI commands
- Send commands to a host machine via USB HID keyboard
- Execute tarot readings, content generation, and NSFW commands
- All with a Celeste-themed interface

## Quick Start

1. **Copy to Flipper firmware**:
   ```bash
   cp -r flipper/celeste_cli /path/to/flipperzero-firmware/applications_user/
   ```

2. **Build and install**:
   ```bash
   cd /path/to/flipperzero-firmware
   ./fbt launch_app APPSRC=celeste_cli
   ```

3. **Use it**:
   - Connect Flipper to laptop via USB
   - Open terminal on laptop
   - Launch "Celeste CLI" app on Flipper
   - Navigate and select commands

## Directory Structure

```
flipper/
└── celeste_cli/
    ├── manifest.txt      # App metadata
    ├── celeste_cli.c     # Main application
    └── README.md         # Detailed documentation
```

## Features

- ✅ Menu-based command selection
- ✅ USB HID keyboard emulation
- ✅ Pre-configured command templates
- ✅ Celeste-themed UI
- ✅ Multiple command categories

## Current Status

**Prototype** - Basic functionality working, ready for testing and expansion.

## Next Steps

1. Test with actual Flipper Zero device
2. Add Celeste pixel art icons
3. Implement custom command builder
4. Add command history
5. Polish UI/UX

See `celeste_cli/README.md` for detailed documentation.

