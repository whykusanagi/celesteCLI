# Celeste One-Shot Commands Reference

All Celeste functionality is now available as one-shot CLI commands without entering the TUI. Perfect for scripting, testing, and quick operations.

## Core Commands

### Session & Context

```bash
# Show token usage from most recent session
./celeste context
./celeste context status

# Display analytics dashboard with corruption theme
./celeste stats

# Export session data
./celeste export          # Export most recent session
./celeste export sessions # Export all sessions
```

### Session Management

```bash
# List all sessions
./celeste session --list

# Load a specific session
./celeste session --load <session-id>

# Clear all sessions
./celeste session --clear
```

### Configuration

```bash
# Show current configuration
./celeste config --show

# List all config profiles
./celeste config --list

# Initialize new config
./celeste config --init <name>

# Set API key
./celeste config --set-key <key>
```

## Skill Execution

All 18 built-in skills can be executed directly:

### No Arguments

```bash
# Generate UUID
./celeste skill generate_uuid

# Generate password (default 16 chars)
./celeste skill generate_password

# List all notes
./celeste skill list_notes

# List all reminders
./celeste skill list_reminders
```

### With Arguments

```bash
# Generate password with custom length
./celeste skill generate_password --length 20

# Get weather for specific zip
./celeste skill get_weather --zip 90210

# Convert units
./celeste skill convert_units --value 100 --from fahrenheit --to celsius

# Save a note
./celeste skill save_note --title "My Note" --content "Note content here"

# Get a note
./celeste skill get_note --title "My Note"

# Set reminder
./celeste skill set_reminder --message "Call mom" --time "2024-12-15T14:00:00Z"

# Tarot reading
./celeste skill tarot_reading --spread three_card

# Generate QR code
./celeste skill generate_qr_code --text "https://example.com"

# Base64 decode
./celeste skill base64_decode --data "SGVsbG8gV29ybGQ="

# Get YouTube videos
./celeste skill get_youtube_videos --channel_id "@someChannel"

# Check Twitch status
./celeste skill check_twitch_live --username "someTwitchUser"
```

### Alternative Syntax

```bash
# Using --exec flag
./celeste skills --exec generate_uuid
./celeste skills --exec generate_password --length 20
```

## Available Skills

Run `./celeste skills --list` to see all 18 built-in skills:

1. **set_reminder** - Set a reminder with time and message
2. **get_youtube_videos** - Get recent videos from a YouTube channel
3. **list_reminders** - List all active reminders
4. **save_note** - Save a note with optional title
5. **tarot_reading** - Generate tarot card reading (three-card or celtic cross)
6. **base64_decode** - Decode a base64 string
7. **generate_uuid** - Generate random UUID (v4)
8. **generate_password** - Generate secure random password
9. **get_note** - Retrieve a note by title
10. **list_notes** - List all saved notes
11. **generate_qr_code** - Generate QR code from text/URL
12. **get_weather** - Get weather forecast for location
13. **convert_units** - Convert between units
14. **convert_currency** - Convert between currencies
15. **convert_timezone** - Convert between timezones
16. **hash_data** - Generate cryptographic hash
17. **encode_base64** - Encode data to base64
18. **check_twitch_live** - Check if Twitch streamer is live

## Testing & Development

One-shot commands are perfect for:

```bash
# Quick testing of token tracking
./celeste context

# Testing corruption rendering
./celeste stats

# Testing skill execution
./celeste skill generate_uuid

# Scripting with skills
UUID=$(./celeste skill generate_uuid | jq -r '.uuid')
echo "Generated: $UUID"

# Export data for backup
./celeste export

# Check session history
./celeste session --list
```

## Combining with TUI

You can still use the TUI for interactive work:

```bash
# Enter interactive mode
./celeste chat

# Or send a single message
./celeste message "What's the weather?"
```

All slash commands in the TUI (/context, /stats, /export, etc.) are now also available as standalone CLI commands for maximum flexibility!
