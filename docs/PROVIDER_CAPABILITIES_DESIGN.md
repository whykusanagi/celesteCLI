# Provider Capabilities & Dynamic Model Selection

## Overview

This document describes the design for a provider capability detection system that enables:
- Dynamic model listing and selection per provider
- Automatic detection of function calling support
- UI indicators showing skill availability
- Unified `/set-model` command for all providers
- Smart defaults with user override capability

## Problem Statement

Currently, users must manually know:
1. Which models support function calling (skills)
2. Which model to use for optimal tool calling (e.g., `grok-4-1-fast` vs `grok-4-latest`)
3. Whether their current endpoint supports skills at all
4. How to switch models for different providers

This leads to confusion and suboptimal experiences.

## Design Goals

1. **Automatic Capability Detection** - System knows if skills work
2. **Smart Model Selection** - Auto-select best tool-calling model
3. **User Override** - Allow manual model selection
4. **Clear UI Feedback** - Show capability status in header/UI
5. **Provider Agnostic** - Same UX across all providers
6. **Graceful Degradation** - Disable skills UI when unsupported

## Architecture

### 1. Provider Metadata System

Create a provider registry with capability information:

```go
// cmd/Celeste/providers/registry.go
package providers

type ProviderCapabilities struct {
    Name              string
    BaseURL           string
    SupportsFunctionCalling bool
    SupportsModelListing   bool
    DefaultModel      string
    PreferredToolModel string // Best model for function calling
    RequiresAPIKey    bool
}

type ModelInfo struct {
    ID          string
    Name        string
    SupportsTools bool
    ContextWindow int
    Description string
}

var ProviderRegistry = map[string]ProviderCapabilities{
    "openai": {
        Name:              "OpenAI",
        BaseURL:           "https://api.openai.com/v1",
        SupportsFunctionCalling: true,
        SupportsModelListing:   true,
        DefaultModel:      "gpt-4o-mini",
        PreferredToolModel: "gpt-4o-mini",
        RequiresAPIKey:    true,
    },
    "grok": {
        Name:              "xAI Grok",
        BaseURL:           "https://api.x.ai/v1",
        SupportsFunctionCalling: true,
        SupportsModelListing:   true,
        DefaultModel:      "grok-4-1-fast",
        PreferredToolModel: "grok-4-1-fast", // Best for tool calling
        RequiresAPIKey:    true,
    },
    "venice": {
        Name:              "Venice.ai",
        BaseURL:           "https://api.venice.ai/api/v1",
        SupportsFunctionCalling: false, // venice-uncensored doesn't support it
        SupportsModelListing:   true,
        DefaultModel:      "venice-uncensored",
        PreferredToolModel: "", // No tool calling support
        RequiresAPIKey:    true,
    },
    "digitalocean": {
        Name:              "DigitalOcean Gradient",
        BaseURL:           "https://agent-*.ondigitalocean.app/api/v1",
        SupportsFunctionCalling: false, // Cloud functions only
        SupportsModelListing:   false,
        DefaultModel:      "gpt-4o-mini",
        PreferredToolModel: "",
        RequiresAPIKey:    true,
    },
}
```

### 2. Model Listing Service

```go
// cmd/Celeste/providers/models.go
package providers

import (
    "context"
    "github.com/sashabaranov/go-openai"
)

type ModelService struct {
    client *openai.Client
    provider string
}

func NewModelService(apiKey, baseURL, provider string) *ModelService {
    config := openai.DefaultConfig(apiKey)
    config.BaseURL = baseURL
    return &ModelService{
        client: openai.NewClientWithConfig(config),
        provider: provider,
    }
}

// ListModels fetches available models from the provider
func (s *ModelService) ListModels(ctx context.Context) ([]ModelInfo, error) {
    caps, ok := ProviderRegistry[s.provider]
    if !ok || !caps.SupportsModelListing {
        return nil, fmt.Errorf("provider does not support model listing")
    }

    // Call /v1/models endpoint
    models, err := s.client.ListModels(ctx)
    if err != nil {
        return nil, err
    }

    // Convert to our ModelInfo structure
    var result []ModelInfo
    for _, m := range models.Models {
        result = append(result, ModelInfo{
            ID:   m.ID,
            Name: m.ID,
            // Provider-specific logic to detect tool support
            SupportsTools: s.detectToolSupport(m.ID),
        })
    }

    return result, nil
}

// detectToolSupport determines if a model supports function calling
func (s *ModelService) detectToolSupport(modelID string) bool {
    // Provider-specific heuristics
    switch s.provider {
    case "openai":
        // All gpt-4* and gpt-3.5-turbo* support tools
        return strings.Contains(modelID, "gpt-4") ||
               strings.Contains(modelID, "gpt-3.5-turbo")

    case "grok":
        // grok-4-1-fast, grok-4-1, grok-beta support tools
        return strings.Contains(modelID, "grok-4") ||
               strings.Contains(modelID, "grok-beta")

    case "venice":
        // Check model metadata from Venice API
        // Some models support tools, but venice-uncensored doesn't
        return !strings.Contains(modelID, "uncensored")

    default:
        return false
    }
}

// GetBestToolModel returns the optimal model for function calling
func (s *ModelService) GetBestToolModel() string {
    caps, ok := ProviderRegistry[s.provider]
    if !ok {
        return ""
    }
    return caps.PreferredToolModel
}
```

### 3. Command Updates

Update `/set-model` to work for all providers, not just images:

```go
// cmd/Celeste/commands/commands.go

func handleSetModel(cmd *Command, ctx *CommandContext) *CommandResult {
    // If in NSFW mode, handle image model (backward compatibility)
    if ctx.NSFWMode {
        return handleImageModel(cmd, ctx)
    }

    // Otherwise, handle chat model
    if len(cmd.Args) == 0 {
        // No args - list available models
        return listAvailableModels(ctx)
    }

    model := cmd.Args[0]

    // Validate model supports tools if skills are enabled
    if ctx.SkillsEnabled {
        supported := checkModelSupportsTools(model, ctx.Provider)
        if !supported {
            return &CommandResult{
                Success: false,
                Message: fmt.Sprintf("⚠️  Model '%s' does not support function calling.\n\nSkills will be disabled with this model. Use a tool-calling model or proceed anyway with /set-model %s --force", model, model),
                ShouldRender: true,
            }
        }
    }

    return &CommandResult{
        Success: true,
        Message: fmt.Sprintf("🤖 Model changed to: %s", model),
        ShouldRender: true,
        StateChange: &StateChange{
            Model: &model,
        },
    }
}

func listAvailableModels(ctx *CommandContext) *CommandResult {
    // Fetch models from provider
    modelService := providers.NewModelService(ctx.APIKey, ctx.BaseURL, ctx.Provider)
    models, err := modelService.ListModels(context.Background())

    if err != nil {
        return &CommandResult{
            Success: false,
            Message: fmt.Sprintf("Failed to fetch models: %v\n\nCommon models for %s:\n%s",
                err, ctx.Provider, getCommonModelsHelp(ctx.Provider)),
            ShouldRender: true,
        }
    }

    // Format model list
    var toolModels []string
    var otherModels []string

    for _, m := range models {
        if m.SupportsTools {
            toolModels = append(toolModels, fmt.Sprintf("  ✓ %s - %s", m.ID, m.Description))
        } else {
            otherModels = append(otherModels, fmt.Sprintf("    %s - %s (no skills)", m.ID, m.Description))
        }
    }

    message := "Available Models:\n\n"

    if len(toolModels) > 0 {
        message += "Function Calling Enabled (Skills Available):\n"
        message += strings.Join(toolModels, "\n") + "\n\n"
    }

    if len(otherModels) > 0 {
        message += "Other Models (Skills Disabled):\n"
        message += strings.Join(otherModels, "\n") + "\n\n"
    }

    message += "Usage: /set-model <model-id>"

    return &CommandResult{
        Success: true,
        Message: message,
        ShouldRender: true,
    }
}
```

### 4. UI Capability Indicators

Update header and skills panel to show capability status:

```go
// cmd/Celeste/tui/app.go

// Update HeaderModel to include capability info
type HeaderModel struct {
    width        int
    endpoint     string
    model        string
    imageModel   string
    nsfwMode     bool
    autoRouted   bool
    skillsEnabled bool  // NEW: Show if skills are available
}

func (m HeaderModel) View() string {
    title := HeaderTitleStyle.Render("✨ Celeste CLI")

    // Endpoint info
    endpointDisplay := m.endpoint
    if m.nsfwMode {
        endpointDisplay = "🔥 NSFW • " + endpointDisplay
    }

    // Model info with capability indicator
    modelInfo := m.model
    if m.skillsEnabled {
        modelInfo += " ✓"  // Checkmark for skills enabled
    } else {
        modelInfo = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#666")).
            Render(modelInfo + " ⚠")  // Warning for no skills
    }

    if m.imageModel != "" {
        modelInfo += " • " + ModelStyle.Render("img:" + m.imageModel)
    }

    endpoint := EndpointStyle.Render(endpointDisplay)
    model := ModelStyle.Render(modelInfo)

    endpointInfo := fmt.Sprintf("%s • %s", endpoint, model)

    // ... rest of header rendering
}

// Update skills panel to grey out when disabled
func (m AppModel) renderSkillsPanel() string {
    if !m.skillsEnabled {
        return lipgloss.NewStyle().
            Foreground(lipgloss.Color("#444")).
            Render("⚠ Skills unavailable with current model")
    }

    return m.skills.View()
}
```

### 5. Automatic Model Selection

On endpoint switch, automatically select best tool-calling model:

```go
// cmd/Celeste/tui/app.go

case CommandResultMsg:
    result := msg

    if result.StateChange.EndpointChange != nil {
        endpoint := *result.StateChange.EndpointChange

        // Get provider capabilities
        caps, ok := providers.ProviderRegistry[endpoint]
        if ok {
            // Auto-select best tool model if available
            if caps.PreferredToolModel != "" {
                m.model = caps.PreferredToolModel
                m.skillsEnabled = true
            } else {
                m.model = caps.DefaultModel
                m.skillsEnabled = caps.SupportsFunctionCalling
            }

            m.header = m.header.SetModel(m.model)
            m.header = m.header.SetSkillsEnabled(m.skillsEnabled)

            LogInfo(fmt.Sprintf("Auto-selected model: %s (skills: %v)",
                m.model, m.skillsEnabled))
        }
    }
```

## User Workflows

### Workflow 1: Switch to Grok with Auto-Selection

```
User: /endpoint grok
System: 🔄 Switched to xAI Grok
        Model: grok-4-1-fast ✓ (optimized for function calling)
        Skills: 18 tools available

User: set a reminder for 10 minutes
System: [Grok calls set_reminder skill successfully]
```

### Workflow 2: Manual Model Override

```
User: /set-model
System: Available Models:

        Function Calling Enabled (Skills Available):
          ✓ grok-4-1-fast - Best for tool calling (2M context)
          ✓ grok-beta - Beta model with tool support

        Other Models (Skills Disabled):
            grok-4-latest - Latest general model (no skills)

        Usage: /set-model <model-id>

User: /set-model grok-4-latest
System: ⚠️  Model 'grok-4-latest' does not support function calling.

        Skills will be disabled with this model.
        Use /set-model grok-4-1-fast for skills support.

        Proceed anyway with /set-model grok-4-latest --force

User: /set-model grok-4-latest --force
System: 🤖 Model changed to: grok-4-latest
        ⚠ Skills disabled - model does not support function calling

[Header shows: grok • grok-4-latest ⚠]
[Skills panel shows greyed out]
```

### Workflow 3: Venice Image Models (Unchanged)

```
User: /nsfw
System: 🔥 NSFW Mode Enabled
        Switched to Venice.ai
        Image Model: lustify-sdxl

User: /set-model
System: Available Image Models:
        • lustify-sdxl (default)
        • wai-Illustrious (anime)
        • hidream (dream-like)
        ...

User: /set-model wai-Illustrious
System: 🎨 Image model changed to: wai-Illustrious
```

### Workflow 4: DigitalOcean (No Skills)

```
User: /endpoint digitalocean
System: 🔄 Switched to DigitalOcean Gradient
        ⚠ This endpoint does not support local function calling
        Skills are disabled

[Header shows: digitalocean • gpt-4o-mini ⚠]
[Skills panel shows: "⚠ Skills unavailable with this provider"]

User: What's the weather?
System: [LLM responds with text, doesn't call get_weather skill]
```

## Implementation Plan

### Phase 1: Provider Registry (Foundation)
- [ ] Create `cmd/Celeste/providers/` package
- [ ] Define `ProviderCapabilities` struct
- [ ] Add provider metadata for OpenAI, Grok, Venice, DigitalOcean
- [ ] Create provider detection logic

### Phase 2: Model Listing Service
- [ ] Implement `ModelService` with `/models` endpoint calls
- [ ] Add model capability detection heuristics
- [ ] Cache model lists (15-minute TTL)
- [ ] Handle API errors gracefully

### Phase 3: Command Updates
- [ ] Update `/set-model` to work for all providers
- [ ] Add `/list-models` as alias
- [ ] Implement `--force` flag for overriding warnings
- [ ] Add model validation logic

### Phase 4: UI Updates
- [ ] Add `skillsEnabled` field to header
- [ ] Update header view with capability indicators
- [ ] Grey out skills panel when disabled
- [ ] Add tooltip/help text for capability status

### Phase 5: Auto-Selection
- [ ] Auto-select best tool model on endpoint switch
- [ ] Log model selection reasoning
- [ ] Persist model preference per provider
- [ ] Add config option to disable auto-selection

### Phase 6: Testing & Documentation
- [ ] Test all provider workflows
- [ ] Update README with new commands
- [ ] Update LLM_PROVIDERS.md with capability matrix
- [ ] Add examples to CLAUDE.md

## API Endpoints Used

### OpenAI
- `GET /v1/models` - List all available models
- Response includes model IDs, object type, created timestamp

### xAI Grok
- `GET /v1/models` - List all available models (OpenAI-compatible)
- Visit console.x.ai for model details

### Venice.ai
- `GET /v1/models` - List all models
- Response includes capabilities (function calling, vision, etc.)

### DigitalOcean
- No model listing endpoint
- Use hardcoded capabilities (no local function calling)

## Configuration

Add to `config.json`:

```json
{
  "auto_select_tool_model": true,
  "prefer_tool_calling": true,
  "model_cache_ttl": 900,
  "warn_on_non_tool_model": true
}
```

## Benefits

1. **Better UX** - Users always know if skills work
2. **Smart Defaults** - Auto-select best models
3. **Clear Feedback** - Visual indicators for capabilities
4. **Flexibility** - Users can override when needed
5. **Consistency** - Same experience across providers
6. **Documentation** - Self-documenting through UI

## Future Enhancements

- Add model comparison view
- Show pricing/rate limits per model
- Add favorite models per provider
- Support custom model aliases
- Add model performance metrics
- Implement A/B testing for model selection

---

**Status**: Design Phase
**Owner**: @whykusanagi
**Last Updated**: 2025-12-04
