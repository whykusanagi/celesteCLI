# Building the Celeste CLI Flipper App

## Prerequisites

1. **Flipper Zero Firmware Repository**
   ```bash
   git clone --recursive https://github.com/flipperdevices/flipperzero-firmware.git
   cd flipperzero-firmware
   ```

2. **Install Build Tools**
   - Follow Flipper's official build instructions
   - Ensure you have the FBT (Flipper Build Tool) set up

## Installation Steps

### Method 1: Copy to Applications Folder

1. **Copy the app**:
   ```bash
   cp -r /path/to/celesteCLI/flipper/celeste_cli applications_user/
   ```

2. **Build just this app**:
   ```bash
   ./fbt launch_app APPSRC=celeste_cli
   ```

3. **Or build entire firmware**:
   ```bash
   ./fbt
   ```

### Method 2: Symlink (for Development)

1. **Create symlink**:
   ```bash
   ln -s /path/to/celesteCLI/flipper/celeste_cli applications_user/celeste_cli
   ```

2. **Build and test**:
   ```bash
   ./fbt launch_app APPSRC=celeste_cli
   ```

## Building

### Quick Build
```bash
./fbt launch_app APPSRC=celeste_cli
```

### Full Firmware Build
```bash
./fbt
```

### Install to Flipper
```bash
./fbt launch
```

## Testing

1. **Connect Flipper Zero via USB**
2. **Open terminal on host machine**
3. **Launch app on Flipper**: Applications → Celeste CLI
4. **Navigate and test commands**

## Troubleshooting

### Build Errors

- **Missing dependencies**: Run `./fbt update_package_index`
- **FBT not found**: Check PATH or use full path to `fbt`
- **Compilation errors**: Check C syntax and Flipper API usage

### Runtime Issues

- **App not appearing**: Check manifest.txt format
- **USB not working**: Verify USB connection and HID support
- **Commands not typing**: Check terminal focus and USB HID mode

## Development Tips

- Use `./fbt launch_app APPSRC=celeste_cli` for quick iteration
- Check Flipper logs: `./fbt cli`
- Test USB HID separately before adding commands
- Use Flipper's built-in HID test app to verify keyboard functionality

## Next Steps

Once basic functionality works:
1. Add Celeste pixel art icons
2. Implement custom command builder
3. Add command history
4. Polish UI/UX
5. Add settings menu

