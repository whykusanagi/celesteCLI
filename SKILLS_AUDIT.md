# Skills Audit - December 4, 2025

## Issue Discovered

User asked for "twitter post" but the LLM called `get_youtube_videos` instead, indicating skill routing problems.

## Root Cause

Legacy Node.js skill JSON files in `~/.celeste/skills/` were loaded as available skills, but they had no Go handler implementations. When the LLM tried to call these skills, they would fail or cause incorrect routing.

**Affected Skills:**
- `twitter_post` - Had JSON definition but no Go handler
- `generate_content` - Had JSON definition but no Go handler
- `nsfw_mode` - Had JSON definition but no Go handler (NSFW handled via `/nsfw` command)

These skills showed up in the skills list, confusing the LLM about which skills were actually available.

## Solution

### 1. Disabled Unimplemented Skills

Renamed legacy skill files to prevent them from being loaded:
```bash
~/.celeste/skills/
├── content.json.disabled  (was generate_content skill)
├── content.js.disabled
├── twitter.json.disabled  (was twitter_post skill)
├── twitter.js.disabled
├── nsfw.json.disabled     (was nsfw_mode skill)
├── nsfw.js.disabled
├── tarot.json             (✓ WORKS - has Go handler)
└── tarot.js               (legacy, not used)
```

### 2. Updated YouTube Default Channel

Set `youtube_default_channel` to "whykusanagi" in `~/.celeste/skills.json`.

The config loader already has "whykusanagi" hardcoded as the fallback (config.go:450), so this ensures consistency.

## Working Skills (18 total)

All skills in `cmd/Celeste/skills/builtin.go` with Go implementations:

### Divination & Entertainment
- ✅ **tarot_reading** - Three-card or Celtic Cross spreads

### Information Services
- ✅ **get_weather** - Current conditions and forecasts (wttr.in)
- ✅ **convert_currency** - Real-time exchange rates
- ✅ **check_twitch_live** - Check if streamers are online
- ✅ **get_youtube_videos** - Get recent uploads from channels (default: whykusanagi)

### Utilities
- ✅ **convert_units** - Length, weight, temperature, volume
- ✅ **convert_timezone** - Convert times between zones
- ✅ **generate_hash** - MD5, SHA256, SHA512
- ✅ **base64_encode** - Encode text to base64
- ✅ **base64_decode** - Decode base64 to text
- ✅ **generate_uuid** - Generate random UUIDs
- ✅ **generate_password** - Secure random passwords
- ✅ **generate_qr_code** - Create QR codes from text/URLs

### Productivity
- ✅ **set_reminder** - Set reminders (local storage)
- ✅ **list_reminders** - List all reminders
- ✅ **save_note** - Save notes (local storage)
- ✅ **get_note** - Retrieve specific note
- ✅ **list_notes** - List all notes

## Disabled Skills (No Go Handlers)

These skills were defined in JSON but had no working Go implementations:

- ❌ **twitter_post** - Post tweets (requires Twitter API v2 implementation)
- ❌ **generate_content** - Platform-specific content generation (was LLM template helper)
- ❌ **nsfw_mode** - Enable NSFW mode (now handled via `/nsfw` chat command)

## Implementation Notes

### Why These Skills Don't Work

1. **JavaScript Handlers**: The JSON files reference `.js` handler files (e.g., `"handler": "./twitter.js"`) which were from the Node.js implementation. The Go implementation doesn't execute JavaScript.

2. **No Go Implementation**: Skills like `twitter_post` would require:
   - Twitter API v2 client implementation
   - OAuth 2.0 authentication
   - Tweet posting endpoint
   - Error handling for rate limits

3. **Architecture Change**: `nsfw_mode` was a skill in Node.js but became a chat command (`/nsfw`) in the Go rewrite for better UX.

### Future Work

If you want to re-implement these skills:

1. **twitter_post**: Implement Twitter API v2 client in Go
2. **generate_content**: Could be reimplemented as a template/formatting helper
3. Consider whether skills like these should be skills (function calling) or chat commands

### Testing

After this fix:
- ✅ Skills list shows only 18 skills (down from 21)
- ✅ All listed skills have working Go handlers
- ✅ LLM won't try to call non-existent skills
- ✅ YouTube skill uses "whykusanagi" as default channel

## Configuration Changes

**File**: `~/.celeste/skills.json`
```json
{
  "youtube_default_channel": "whykusanagi"
}
```

This ensures YouTube skill defaults to your channel when no channel is specified in the skill call.

## Verification

Run `Celeste chat` and type:
```
/help
```

You should see only 18 skills listed in the skills panel (no twitter_post, generate_content, or nsfw_mode).

Ask the LLM:
```
What skills do you have access to?
```

It should only list the 18 working skills.
