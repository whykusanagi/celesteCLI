# CelesteCLI - Enterprise RPG Menu UI Specification

## Design Principles

1. **Active Feedback**: Every interaction provides immediate visual feedback
2. **Contextual Help**: Information appears when needed, not all at once
3. **Clear State**: User always knows: what model, what it can do, how to use it
4. **Progressive Disclosure**: Show essentials first, details on demand
5. **Consistent Navigation**: RPG-style menu with clear patterns

## Menu Panel Structure

The menu panel (bottom section) has **3 display modes**:

### Mode 1: STATUS (Default - Always Visible)
```
╭─────────────────────────────────────────────────────────────╮
│ ⚡ SYSTEM STATUS                                            │
│ Provider: openai                                            │
│ Model: gpt-4o-mini                                          │
│ Skills: ✓ Enabled (18 available)        NSFW: ✗ Disabled   │
│                                                             │
│ 💡 TIP: Type /menu to see commands • /skills to see tools  │
╰─────────────────────────────────────────────────────────────╯
```

**Key Info:**
- Current provider/endpoint
- Current model name
- Skills status: ✓ Enabled (count) or ✗ Disabled (with reason)
- NSFW mode indicator
- Quick tip for navigation

### Mode 2: COMMANDS MENU (/menu command)
```
╭─────────────────────────────────────────────────────────────╮
│ 📋 COMMANDS MENU                                            │
│                                                             │
│ /help          Show detailed help                           │
│ /menu          Toggle this menu                             │
│ /skills        View available AI skills                     │
│ /config        List configuration profiles                  │
│ /endpoint      Switch API provider                          │
│ /model         Change current model                         │
│ /nsfw          Enable uncensored mode                       │
│ /clear         Clear chat history                           │
│                                                             │
│ 💡 Type command name to see details as you type            │
╰─────────────────────────────────────────────────────────────╯
```

### Mode 3: SKILLS MENU (/skills command)
```
╭─────────────────────────────────────────────────────────────╮
│ ✨ AVAILABLE SKILLS (18 total)                              │
│                                                             │
│ ⏳ get_weather         Getting weather... (EXECUTING)       │
│ ○ convert_timezone     Convert times between timezones     │
│ ○ get_youtube_videos   Fetch recent YouTube videos         │
│ ○ check_twitch_live    Check if Twitch streamer is live    │
│ ○ generate_password    Generate secure passwords           │
│ ○ tarot_reading        Get tarot card reading              │
│ ○ save_note            Save notes to file                  │
│ ...                                                         │
│                                                             │
│ 💡 Type skill name to see full description                 │
╰─────────────────────────────────────────────────────────────╯
```

## Contextual Help System

### As User Types:

**Typing "/hel":**
```
╭─────────────────────────────────────────────────────────────╮
│ ⚡ SYSTEM STATUS                                            │
│ Provider: openai • Model: gpt-4o-mini • Skills: ✓          │
│                                                             │
│ 💡 HELP for: /help                                          │
│ Shows comprehensive help menu with all commands, skills,    │
│ and usage examples. Includes keyboard shortcuts and tips.   │
╰─────────────────────────────────────────────────────────────╯
```

**Typing "weather":**
```
╭─────────────────────────────────────────────────────────────╮
│ ⚡ SYSTEM STATUS                                            │
│ Provider: openai • Model: gpt-4o-mini • Skills: ✓          │
│                                                             │
│ 💡 SKILL: get_weather                                       │
│ Get current weather information for any city worldwide.     │
│ Just ask: "What's the weather in Tokyo?"                    │
╰─────────────────────────────────────────────────────────────╯
```

**Typing "/config gr":**
```
╭─────────────────────────────────────────────────────────────╮
│ ⚡ SYSTEM STATUS                                            │
│ Provider: openai • Model: gpt-4o-mini • Skills: ✓          │
│                                                             │
│ 💡 HELP for: /config                                        │
│ Switch to saved configuration profile. Example:             │
│   /config grok  - Load Grok configuration                   │
│ Available: default, grok, vertex, openrouter                │
╰─────────────────────────────────────────────────────────────╯
```

## Skill Execution Feedback

### Before Execution:
```
Skills: ✓ Enabled (18 available)
```

### During Execution:
```
Skills: ⏳ get_weather ░▒▓█▀w▄e▌a▐t■h□e▪r▫ (Executing...)
```

### After Completion:
```
Skills: ✓ Enabled (18 available)
```

### On Error:
```
Skills: ❌ get_weather failed - API timeout
```

## Commands Reference

### Navigation Commands:
- `/menu` - Toggle commands menu
- `/skills` - Toggle skills menu
- `/help` - Show full help

### Configuration Commands:
- `/config` - List available config profiles
- `/config <name>` - Load config profile (sets endpoint + model + API key)
- `/endpoint <name>` - Switch API endpoint only
- `/model <name>` - Change model within current endpoint

### Mode Commands:
- `/nsfw` - Enable Venice.ai uncensored mode (disables skills)
- `/safe` - Return to safe mode (enables skills)

### Utility Commands:
- `/clear` - Clear chat history
- `/set-model <model>` - Set image generation model (NSFW mode only)

## UI Behavior Rules

### Rule 1: Always Show Active Status
The STATUS section MUST always display:
1. Current provider/endpoint
2. Current model
3. Skills status with count or reason disabled
4. NSFW mode indicator

### Rule 2: Contextual Help Priority
When user is typing:
1. If starts with "/", show command help
2. If matches skill name, show skill description
3. If generic text, show "Ask me anything" tip

### Rule 3: Menu Toggling
- `/menu` toggles commands list
- `/skills` toggles skills list
- Only one menu visible at a time
- ESC or typing returns to STATUS view

### Rule 4: Execution Feedback
- Executing skills show corruption animation
- Status line updates in real-time
- Completion returns to normal display
- Errors shown with clear message

### Rule 5: Skills Availability
Skills panel MUST indicate:
- ✓ Enabled (count) - Skills are available
- ✗ Disabled (NSFW Mode) - Reason: Venice doesn't support functions
- ✗ Disabled (Model) - Reason: Current model doesn't support tools
- ⚠️ Limited Support - Some features may not work

## Implementation Notes

### State Management:
```go
type MenuState int

const (
    MenuStateStatus MenuState = iota  // Default - show status
    MenuStateCommands                  // /menu - show commands
    MenuStateSkills                    // /skills - show skills
)

type SkillsModel struct {
    menuState MenuState
    currentInput string
    // ... existing fields
}
```

### Update Flow:
1. User types → SetCurrentInput(value)
2. App.View() → calls skills.SetConfig() → updates STATUS
3. skills.View() → renders based on menuState + currentInput
4. Contextual help overlays on STATUS when typing

### Height Allocation:
- Menu panel: 10-12 lines minimum
- STATUS mode: 6 lines
- COMMANDS menu: 12 lines
- SKILLS menu: Dynamic (scrollable if >12)

## Success Criteria

User can answer these questions at a glance:
1. ✅ What model am I using? → STATUS line
2. ✅ Can I use skills? → Skills: ✓/✗ with reason
3. ✅ What commands are available? → /menu
4. ✅ What skills can I use? → /skills
5. ✅ How do I use X? → Type X, see contextual help
6. ✅ Is something happening? → Corruption animation
