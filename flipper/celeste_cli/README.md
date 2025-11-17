# Celeste CLI - Flipper Zero App

A Flipper Zero application that acts as a USB HID keyboard remote controller for the CelesteAI CLI tool. When connected to a host machine, it can type `celestecli` commands directly into the terminal.

## Features

- **Celeste-themed UI** - Custom interface with Celeste branding
- **Quick Command Access** - One-button access to common commands
- **Menu Navigation** - Easy navigation through command categories
- **USB HID Keyboard** - Types commands directly to host terminal
- **Command Templates** - Pre-configured commands for common tasks

## Command Categories

### Tarot Readings
- 3-Card Tarot Spread
- Celtic Cross Spread
- Divine Reading (with AI interpretation)
- Divine NSFW Reading
- Parsed Output (for AI consumption)

### Content Generation
- Twitter Short Posts
- Twitter with different tones (lewd, teasing, chaotic)
- YouTube Descriptions

### NSFW Mode
- NSFW Text Generation
- NSFW Image Generation
- List Available Models

## Installation

### Prerequisites
- Flipper Zero device
- Flipper Zero firmware development environment
- USB connection to host machine

### Build Instructions

1. **Clone Flipper Zero Firmware** (if not already done):
```bash
git clone --recursive https://github.com/flipperdevices/flipperzero-firmware.git
cd flipperzero-firmware
```

2. **Copy the app to applications folder**:
```bash
cp -r /path/to/celesteCLI/flipper/celeste_cli applications_user/celeste_cli
```

3. **Build the firmware**:
```bash
./fbt launch_app APPSRC=celeste_cli
```

Or build the entire firmware:
```bash
./fbt
```

4. **Install to Flipper Zero**:
```bash
./fbt launch
```

## Usage

### Setup

1. **Connect Flipper Zero to host machine via USB**
2. **Open terminal on host machine**
3. **Ensure `celestecli` is installed and in PATH on host**

### Running Commands

1. **Launch the app** on Flipper Zero:
   - Navigate to Applications → Celeste CLI
   - Press OK

2. **Navigate menus**:
   - UP/DOWN: Navigate menu items
   - OK: Select/Execute
   - BACK: Go back/Exit

3. **Select a command**:
   - Choose category (Tarot, Content, NSFW)
   - Select specific command
   - Confirm execution

4. **Command execution**:
   - Flipper types the command into host terminal
   - Command executes on host machine
   - Results display in terminal

### Example Flow

```
1. Launch "Celeste CLI" app
2. Press OK on splash screen
3. Navigate to "Tarot Readings"
4. Select "3-Card Tarot"
5. Press OK to confirm
6. Command types: `celestecli --tarot\n`
7. Check host terminal for results
```

## Customization

### Adding New Commands

Edit `celeste_cli.c` and add to the `commands` array:

```c
{"Command Name", "celestecli --format short --platform twitter --topic \"NIKKE\"\n", MenuCategoryContent},
```

### Modifying UI

- Edit render functions in `celeste_cli.c`
- Adjust menu layouts and text
- Add custom icons/graphics

### Command Templates

Commands are stored as strings in the `commands` array. Modify them to match your preferred command patterns.

## Technical Details

### USB HID Implementation

The app uses Flipper's USB HID keyboard functionality to type commands. It:
- Enables USB HID mode
- Waits for USB connection
- Types each character with appropriate delays
- Handles special characters (quotes, dashes, etc.)
- Sends Enter key to execute

### Character Mapping

Currently supports:
- Letters (a-z, A-Z)
- Numbers (0-9)
- Space, dash, quotes
- Enter/Return

Add more special characters in the `send_char()` function as needed.

## Limitations

- **Screen Size**: 128x64 pixel display limits UI complexity
- **Input Method**: No text input for custom commands (yet)
- **USB Dependency**: Requires USB connection to host
- **Character Support**: Limited special character support (expandable)

## Future Enhancements

- [ ] Custom command builder with text input
- [ ] Command history storage
- [ ] Settings menu (defaults, delays)
- [ ] Celeste pixel art icons
- [ ] Animation support
- [ ] Bluetooth connectivity option
- [ ] Command templates editor
- [ ] Favorite commands quick access

## Troubleshooting

### USB Not Working
- Ensure Flipper is connected via USB
- Check USB cable quality
- Try different USB port
- Restart Flipper Zero

### Commands Not Typing
- Verify terminal is focused on host
- Check USB HID is enabled
- Increase delays in `send_char()` if needed

### Wrong Characters
- Add missing character mappings to `send_char()`
- Check keyboard layout (US vs International)

## Development

### Project Structure
```
celeste_cli/
├── manifest.txt          # App metadata
├── celeste_cli.c         # Main application code
└── README.md            # This file
```

### Key Functions

- `celeste_cli_app()` - Main entry point
- `render_callback()` - UI rendering
- `input_callback()` - Input handling
- `send_command()` - USB HID keyboard typing
- `send_char()` - Character typing with mapping

## License

Part of the CelesteAI ecosystem. See main project license.

## Contributing

This is a prototype. Contributions welcome for:
- UI improvements
- More command templates
- Better character support
- Custom command builder
- Celeste artwork integration

