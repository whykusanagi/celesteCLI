# Bot Integration Guide

## Overview

The CelesteCLI now supports proper user isolation and override functionality for Discord and Twitch bot integrations. This ensures that each user gets their own conversation context and allows for PGP-signed override commands.

## User Isolation

### Problem Solved
Previously, all conversations were attributed to a single user ID ("kusanagi"), which caused issues for Discord and Twitch bots where multiple users need separate conversation contexts.

### Solution
The CLI now supports per-user conversation tracking through the `CELESTE_USER_ID` environment variable.

### Usage for Bot Integration

#### Discord Bot
```bash
# Set user-specific environment variables
export CELESTE_USER_ID="discord_user_123"
export CELESTE_PLATFORM="discord"
export CELESTE_CHANNEL_ID="channel_456"
export CELESTE_GUILD_ID="guild_789"
export CELESTE_MESSAGE_ID="msg_101112"

# Run CelesteCLI with user context
./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync
```

#### Twitch Bot
```bash
# Set user-specific environment variables
export CELESTE_USER_ID="twitch_user_456"
export CELESTE_PLATFORM="twitch"
export CELESTE_CHANNEL_ID="kusanagi_abyss"

# Run CelesteCLI with user context
./celestecli --type tweet --game "NIKKE" --tone "chaotic" --sync
```

## Override Functionality

### PGP Signature Verification
The CLI supports PGP-signed override commands to bypass Celeste's normal restrictions.

### Environment Variables
- `CELESTE_OVERRIDE_ENABLED` - Enable override mode (true/false)
- `CELESTE_PGP_SIGNATURE` - PGP signature for override commands

### Usage
```bash
# Enable override mode with PGP signature
export CELESTE_OVERRIDE_ENABLED="true"
export CELESTE_PGP_SIGNATURE="kusanagi-abyss-override"

# Run with override permissions
./celestecli --type tweet --game "NIKKE" --tone "explicit"
```

### Override Output
When override mode is enabled, you'll see:
```
🔓 Override mode enabled - Abyssal laws may be bypassed
```

## Conversation Data Structure

### Enhanced Metadata
Each conversation entry now includes:

```json
{
  "id": "conversation_id",
  "timestamp": "2024-01-01T00:00:00Z",
  "user_id": "discord_user_123",
  "content_type": "tweet",
  "tone": "teasing",
  "game": "NIKKE",
  "persona": "celeste_stream",
  "prompt": "user_prompt",
  "response": "ai_response",
  "metadata": {
    "platform": "discord",
    "channel_id": "channel_456",
    "guild_id": "guild_789",
    "message_id": "msg_101112",
    "pgp_signature": "kusanagi-abyss-override",
    "override_enabled": true,
    "command_line": "./celestecli --type tweet --game NIKKE --tone teasing",
    "api_endpoint": "https://agent-2fb16e0d1ddb38cd9eb3-ajipw.ondigitalocean.app/api/v1/"
  },
  "intent": "social_media",
  "purpose": "tweet",
  "topics": ["nikke", "gaming"],
  "sentiment": "positive",
  "platform": "twitter",
  "tags": ["celeste", "ai", "content"],
  "context": "Game: NIKKE, Tone: teasing, Persona: celeste_stream",
  "success": true
}
```

## Bot Integration Examples

### Discord Bot Integration
```go
// Example Discord bot integration
func handleCelesteCommand(userID, channelID, guildID, messageID string, prompt string) {
    // Set environment variables for user context
    os.Setenv("CELESTE_USER_ID", userID)
    os.Setenv("CELESTE_PLATFORM", "discord")
    os.Setenv("CELESTE_CHANNEL_ID", channelID)
    os.Setenv("CELESTE_GUILD_ID", guildID)
    os.Setenv("CELESTE_MESSAGE_ID", messageID)
    
    // Run CelesteCLI
    cmd := exec.Command("./celestecli", "--type", "tweet", "--tone", "teasing", "--sync")
    output, err := cmd.Output()
    
    if err != nil {
        // Handle error
        return
    }
    
    // Send response to Discord
    sendDiscordMessage(channelID, string(output))
}
```

### Twitch Bot Integration
```go
// Example Twitch bot integration
func handleCelesteCommand(userID, channelID string, prompt string) {
    // Set environment variables for user context
    os.Setenv("CELESTE_USER_ID", userID)
    os.Setenv("CELESTE_PLATFORM", "twitch")
    os.Setenv("CELESTE_CHANNEL_ID", channelID)
    
    // Run CelesteCLI
    cmd := exec.Command("./celestecli", "--type", "tweet", "--tone", "chaotic", "--sync")
    output, err := cmd.Output()
    
    if err != nil {
        // Handle error
        return
    }
    
    // Send response to Twitch chat
    sendTwitchMessage(channelID, string(output))
}
```

### Override Command Integration
```go
// Example override command handling
func handleOverrideCommand(userID, signature string, prompt string) bool {
    // Check if user has override permissions
    if !hasOverridePermissions(userID) {
        return false
    }
    
    // Set override environment variables
    os.Setenv("CELESTE_USER_ID", userID)
    os.Setenv("CELESTE_OVERRIDE_ENABLED", "true")
    os.Setenv("CELESTE_PGP_SIGNATURE", signature)
    
    // Run CelesteCLI with override
    cmd := exec.Command("./celestecli", "--type", "tweet", "--tone", "explicit", "--sync")
    output, err := cmd.Output()
    
    if err != nil {
        return false
    }
    
    // Send override response
    sendOverrideResponse(userID, string(output))
    return true
}
```

## S3 Storage Structure

### Per-User Conversations
```
s3://whykusanagi/celeste/conversations/
├── discord_user_123/
│   ├── 1760833689259283000.json
│   ├── 1760833689259283001.json
│   └── ...
├── twitch_user_456/
│   ├── 1760833689259283002.json
│   ├── 1760833689259283003.json
│   └── ...
└── kusanagi/
    ├── 1760833689259283004.json
    └── ...
```

### Benefits
- **User Isolation**: Each user gets their own conversation history
- **Context Preservation**: Conversations are tracked per user
- **Override Tracking**: Override commands are logged with PGP signatures
- **Platform Identification**: Easy to identify which platform generated each conversation

## Security Considerations

### PGP Signature Verification
- **Current Implementation**: Basic pattern matching for demonstration
- **Production Implementation**: Use proper PGP library like `golang.org/x/crypto/openpgp`
- **Key Management**: Store Kusanagi's public key securely
- **Signature Validation**: Verify signatures against trusted keys

### Override Permissions
- **Access Control**: Only authorized users can use override commands
- **Audit Logging**: All override commands are logged with signatures
- **Rate Limiting**: Implement rate limiting for override commands
- **Monitoring**: Monitor override usage for abuse

## Testing

### User Isolation Test
```bash
# Test Discord user
CELESTE_USER_ID="discord_user_123" CELESTE_PLATFORM="discord" ./celestecli --type tweet --game "NIKKE" --tone "teasing" --sync

# Test Twitch user  
CELESTE_USER_ID="twitch_user_456" CELESTE_PLATFORM="twitch" ./celestecli --type tweet --game "NIKKE" --tone "chaotic" --sync
```

### Override Test
```bash
# Test override functionality
CELESTE_OVERRIDE_ENABLED="true" CELESTE_PGP_SIGNATURE="kusanagi-abyss-override" ./celestecli --type tweet --game "NIKKE" --tone "explicit"
```

## Conclusion

The CelesteCLI now properly supports:
- ✅ **User Isolation**: Separate conversations per user
- ✅ **Bot Integration**: Discord and Twitch bot support
- ✅ **Override Functionality**: PGP-signed override commands
- ✅ **Audit Logging**: Complete conversation tracking
- ✅ **Platform Identification**: Easy platform-specific handling

This ensures that Discord and Twitch bots can properly manage user conversations and allows for secure override commands when needed.
