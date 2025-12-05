# Implementation Plan: Phases 4-7

## Phase 4: TUI Integration with Provider System

### Goal
Wire up the provider system into the existing TUI (app.go) so commands can access provider capabilities.

### Changes Required

#### 1. Add Provider Field to AppModel
**File**: `cmd/Celeste/tui/app.go`

**Location**: Around line 35-40 (AppModel struct)

```go
type AppModel struct {
    // ... existing fields ...
    provider      string // NEW: Current provider (grok, openai, venice, etc.)
    skillsEnabled bool   // NEW: Whether skills are currently available
}
```

#### 2. Initialize Provider on App Start
**Location**: `NewAppModel()` function

```go
func NewAppModel(config *config.Config, ...) AppModel {
    // ... existing code ...

    // Detect provider from config
    provider := providers.DetectProvider(config.BaseURL)

    // Check if provider supports function calling
    caps, _ := providers.GetProvider(provider)
    skillsEnabled := caps.SupportsFunctionCalling

    return AppModel{
        // ... existing fields ...
        provider:      provider,
        skillsEnabled: skillsEnabled,
    }
}
```

#### 3. Update CommandContext Creation
**Location**: `Update()` function, where commands are executed (around line 250)

**Find**: Call to `commands.Execute(cmd, ctx)`

**Update**:
```go
// Create command context with provider info
ctx := &commands.CommandContext{
    NSFWMode:      m.nsfwMode,
    Provider:      m.provider,
    CurrentModel:  m.model,
    APIKey:        m.config.APIKey,
    BaseURL:       m.config.BaseURL,
    SkillsEnabled: m.skillsEnabled,
}

result := commands.Execute(cmd, ctx)
```

#### 4. Update Provider on Endpoint Switch
**Location**: `Update()` function, CommandResultMsg handler

```go
case CommandResultMsg:
    result := msg

    if result.StateChange.EndpointChange != nil {
        endpoint := *result.StateChange.EndpointChange
        m.endpoint = endpoint

        // NEW: Detect and update provider
        m.provider = providers.DetectProvider(m.config.BaseURL)

        // NEW: Update skills availability
        caps, ok := providers.GetProvider(m.provider)
        if ok {
            m.skillsEnabled = caps.SupportsFunctionCalling
        }
    }
```

### Testing
- Compile check: `go build -o Celeste cmd/Celeste/main.go`
- Verify provider detection works
- Verify context passes to commands correctly

---

## Phase 5: UI Capability Indicators

### Goal
Show visual indicators in the UI when skills are enabled vs disabled.

### Changes Required

#### 1. Add SkillsEnabled to HeaderModel
**File**: `cmd/Celeste/tui/app.go`

**Location**: HeaderModel struct (around line 803)

```go
type HeaderModel struct {
    width         int
    endpoint      string
    model         string
    imageModel    string
    nsfwMode      bool
    autoRouted    bool
    skillsEnabled bool // NEW: Show if skills work
}
```

#### 2. Add SetSkillsEnabled Method
**Location**: After other HeaderModel methods

```go
// SetSkillsEnabled sets whether skills are available.
func (m HeaderModel) SetSkillsEnabled(enabled bool) HeaderModel {
    m.skillsEnabled = enabled
    return m
}
```

#### 3. Update Header View with Indicators
**Location**: `HeaderModel.View()` function (around line 854)

**Find**: Model rendering code

**Update**:
```go
// Model info with capability indicator
modelInfo := m.model
if m.skillsEnabled {
    modelInfo += " ✓"  // Checkmark for skills enabled
} else if m.model != "" {
    modelInfo = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#666")).
        Render(modelInfo + " ⚠")  // Warning for no skills
}

if m.imageModel != "" {
    modelInfo += " • " + ModelStyle.Render("img:" + m.imageModel)
}
```

#### 4. Update Header When Skills State Changes
**Location**: `Update()` function, multiple places

**After endpoint switches**:
```go
m.header = m.header.SetSkillsEnabled(m.skillsEnabled)
```

**After model changes**:
```go
if result.StateChange.Model != nil {
    m.model = *result.StateChange.Model
    m.header = m.header.SetModel(m.model)

    // NEW: Update skills enabled based on model
    detector := providers.NewModelDetection(m.provider)
    m.skillsEnabled = detector.SupportsTools(m.model)
    m.header = m.header.SetSkillsEnabled(m.skillsEnabled)
}
```

#### 5. Grey Out Skills Panel When Disabled
**Location**: Main `View()` function, skills panel rendering (around line 720)

**Find**: Skills panel rendering

**Update**:
```go
// Skills panel
if !m.skillsEnabled {
    // Show disabled state
    disabledStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#444"))
    sections = append(sections,
        disabledStyle.Render("⚠ Skills unavailable with current provider/model"))
} else {
    sections = append(sections, m.skills.View())
}
```

### Testing
- Visual check: Header shows `✓` for tool models
- Visual check: Header shows `⚠` for non-tool models
- Visual check: Skills panel greys out when unavailable

---

## Phase 6: Auto-Selection of Best Tool Model

### Goal
Automatically select the best tool-calling model when switching providers.

### Changes Required

#### 1. Auto-Select on Endpoint Switch
**File**: `cmd/Celeste/tui/app.go`

**Location**: `Update()` function, CommandResultMsg handler for EndpointChange

**Find**: Existing endpoint change handling

**Update**:
```go
if result.StateChange.EndpointChange != nil {
    endpoint := *result.StateChange.EndpointChange
    m.endpoint = endpoint

    // Update provider
    m.provider = providers.DetectProvider(m.config.BaseURL)

    // NEW: Auto-select best tool model
    caps, ok := providers.GetProvider(m.provider)
    if ok {
        if caps.PreferredToolModel != "" {
            // Auto-select best tool model
            m.model = caps.PreferredToolModel
            m.skillsEnabled = true
            m.header = m.header.SetModel(m.model)

            LogInfo(fmt.Sprintf("Auto-selected model: %s (optimized for tool calling)", m.model))
        } else {
            // Use default model
            m.model = caps.DefaultModel
            m.skillsEnabled = caps.SupportsFunctionCalling
            m.header = m.header.SetModel(m.model)

            LogInfo(fmt.Sprintf("Using default model: %s", m.model))
        }

        m.header = m.header.SetSkillsEnabled(m.skillsEnabled)
    }

    m.header = m.header.SetEndpoint(m.endpoint)
}
```

#### 2. Update LLM Client Configuration
**Location**: After model auto-selection

**Ensure**: LLM client uses the new model

```go
// Update LLM client with new model
llmConfig := &llm.Config{
    APIKey:            m.config.APIKey,
    BaseURL:           m.config.BaseURL,
    Model:             m.model, // Use auto-selected model
    Timeout:           m.config.GetTimeout(),
    SkipPersonaPrompt: m.config.SkipPersonaPrompt,
    SimulateTyping:    m.config.SimulateTyping,
    TypingSpeed:       m.config.TypingSpeed,
}
m.llm.UpdateConfig(llmConfig)
```

#### 3. Add User Notification
**Location**: After auto-selection

```go
// Show status message about auto-selection
if caps.PreferredToolModel != "" {
    m.status = m.status.SetText(
        fmt.Sprintf("✓ Auto-selected %s (optimized for skills)", m.model))
}
```

### Testing
- Switch to Grok → Should auto-select `grok-4-1-fast`
- Switch to OpenAI → Should auto-select `gpt-4o-mini`
- Switch to Venice → Should use `venice-uncensored`
- Verify LLM client uses correct model

---

## Phase 7: Testing and Validation

### Goal
Ensure everything compiles, integrates correctly, and follows expected workflows.

### Tasks

#### 1. Compilation Test
```bash
cd /Users/kusanagi/Development/celesteCLI
go build -o Celeste cmd/Celeste/main.go
```

**Expected**: Clean build with no errors

**If errors**: Fix import paths, type mismatches, undefined references

#### 2. Static Code Analysis
```bash
go vet ./cmd/Celeste/...
gofmt -l ./cmd/Celeste/
```

**Expected**: No warnings, no formatting issues

#### 3. Manual Workflow Testing

**Test 1: Grok Provider**
```bash
Celeste chat --config grok
> /set-model
# Should list: grok-4-1-fast, grok-4-1, grok-beta, grok-4-latest
# Should show ✓ for tool models, (no skills) for others

> /set-model grok-4-1-fast
# Should succeed with ✓

> /set-model grok-4-latest
# Should warn about no skills, suggest --force
```

**Test 2: Provider Switching**
```bash
> /endpoint grok
# Header should show: grok • grok-4-1-fast ✓
# Skills panel should be active

> /endpoint digitalocean
# Header should show: digitalocean • gpt-4o-mini ⚠
# Skills panel should show "unavailable"
```

**Test 3: Venice Consistency**
```bash
> /nsfw
# Should switch to Venice, show image model

> /set-model
# Should list IMAGE models (not chat models)

> /set-model wai-Illustrious
# Should change image model

> /safe
# Should return to safe mode

> /set-model
# Should list CHAT models again
```

**Test 4: OpenAI Models**
```bash
> /endpoint openai
# Should auto-select gpt-4o-mini

> /set-model
# Should list: gpt-4o-mini, gpt-4o, gpt-4-turbo, etc.
# All should show ✓ (all support tools)
```

#### 4. Edge Case Testing

**Test 1: Unknown Provider**
```bash
> /endpoint custom
> /set-model
# Should handle gracefully, show "unknown provider"
```

**Test 2: API Failure**
```bash
# With invalid API key
> /set-model
# Should fall back to static model list
```

**Test 3: Force Override**
```bash
> /set-model grok-4-latest --force
# Should accept and disable skills
# Header should show ⚠
```

#### 5. Visual Validation

Check these visual elements:
- [ ] Header shows provider name
- [ ] Header shows model name
- [ ] Header shows ✓ when skills enabled
- [ ] Header shows ⚠ when skills disabled
- [ ] Skills panel greys out when unavailable
- [ ] Status messages are clear and helpful

#### 6. Log Validation

Check logs for:
```bash
tail -f ~/.celeste/logs/celeste_$(date +%Y-%m-%d).log
```

**Should see**:
- `Auto-selected model: grok-4-1-fast (optimized for tool calling)`
- Provider detection messages
- Model validation logs
- No errors or warnings

### Success Criteria

✅ Clean compilation
✅ No go vet warnings
✅ All 4 test workflows pass
✅ Edge cases handled gracefully
✅ Visual indicators work correctly
✅ Logs show expected behavior
✅ Venice pattern maintained
✅ Grok auto-selects `grok-4-1-fast`

---

## Execution Order

1. **Phase 4** → TUI Integration (provider detection, context passing)
2. **Phase 5** → UI Indicators (visual feedback)
3. **Phase 6** → Auto-Selection (smart defaults)
4. **Phase 7** → Testing (validation)

Each phase builds on the previous one and can be tested independently.

## Rollback Plan

If issues arise:
1. Git commit after each phase
2. Can revert individual phases if needed
3. Core system (Phases 1-3) is already stable

## Estimated Impact

- **Files Modified**: 2 (`app.go`, `styles.go` for colors)
- **Lines Added**: ~150 lines
- **Risk Level**: Low (mostly additive changes)
- **Backwards Compatible**: Yes (all existing features preserved)

---

**Ready to Execute**: Yes
**Prerequisites Met**: Yes (Phases 1-3 complete)
**Blocking Issues**: None
