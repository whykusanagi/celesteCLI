// Package llm provides the LLM client for Celeste CLI.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/whykusanagi/celesteCLI/cmd/celeste/skills"
	"github.com/whykusanagi/celesteCLI/cmd/celeste/tui"
)

// Client wraps the OpenAI client for Celeste's needs.
type Client struct {
	client       *openai.Client
	config       *Config
	registry     *skills.Registry
	systemPrompt string
}

// Config holds LLM client configuration.
type Config struct {
	APIKey            string
	BaseURL           string
	Model             string
	Timeout           time.Duration
	SkipPersonaPrompt bool
	SimulateTyping    bool
	TypingSpeed       int // chars per second
}

// NewClient creates a new LLM client.
func NewClient(config *Config, registry *skills.Registry) *Client {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	return &Client{
		client:   openai.NewClientWithConfig(clientConfig),
		config:   config,
		registry: registry,
	}
}

// SetSystemPrompt sets the system prompt (Celeste persona).
func (c *Client) SetSystemPrompt(prompt string) {
	c.systemPrompt = prompt
}

// UpdateConfig updates the client configuration and recreates the OpenAI client.
// This allows dynamic endpoint/model switching during runtime.
func (c *Client) UpdateConfig(config *Config) {
	c.config = config

	// Recreate OpenAI client with new config
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	c.client = openai.NewClientWithConfig(clientConfig)
}

// GetConfig returns the current configuration.
func (c *Client) GetConfig() *Config {
	return c.config
}

// ChatCompletionResult holds the result of a chat completion.
type ChatCompletionResult struct {
	Content      string
	ToolCalls    []ToolCallResult
	FinishReason string
	Error        error
}

// ToolCallResult holds a tool call from the LLM.
type ToolCallResult struct {
	ID        string
	Name      string
	Arguments string
}

// SendMessageSync sends a message synchronously and returns the result.
func (c *Client) SendMessageSync(ctx context.Context, messages []tui.ChatMessage, tools []tui.SkillDefinition) (*ChatCompletionResult, error) {
	// Convert messages to OpenAI format
	openAIMessages := c.convertMessages(messages)

	// Convert tools to OpenAI format
	openAITools := c.convertTools(tools)

	// Create request
	req := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: openAIMessages,
		Stream:   true,
	}

	if len(openAITools) > 0 {
		req.Tools = openAITools
	}

	// Create streaming request
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	result := &ChatCompletionResult{}
	var toolCalls []openai.ToolCall

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			result.Error = err
			return result, err
		}

		for _, choice := range response.Choices {
			// Handle content delta
			if choice.Delta.Content != "" {
				result.Content += choice.Delta.Content
			}

			// Handle tool calls
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Index != nil {
					idx := *tc.Index
					for len(toolCalls) <= idx {
						toolCalls = append(toolCalls, openai.ToolCall{})
					}
					if tc.ID != "" {
						toolCalls[idx].ID = tc.ID
					}
					if tc.Type != "" {
						toolCalls[idx].Type = tc.Type
					}
					if tc.Function.Name != "" {
						toolCalls[idx].Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						toolCalls[idx].Function.Arguments += tc.Function.Arguments
					}
				}
			}

			// Check finish reason
			if choice.FinishReason != "" {
				result.FinishReason = string(choice.FinishReason)
			}
		}
	}

	// Convert tool calls
	for _, tc := range toolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCallResult{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return result, nil
}

// StreamCallback is called for each chunk during streaming.
type StreamCallback func(chunk StreamChunk)

// StreamChunk represents a streaming chunk.
// TokenUsage holds token usage information from API response
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type StreamChunk struct {
	Content      string
	IsFirst      bool
	IsFinal      bool
	FinishReason string
	ToolCalls    []ToolCallResult
	Usage        *TokenUsage // Only populated on final chunk with stream_options
}

// SendMessageStream sends a message with streaming callback.
func (c *Client) SendMessageStream(ctx context.Context, messages []tui.ChatMessage, tools []tui.SkillDefinition, callback StreamCallback) error {
	// Convert messages to OpenAI format
	openAIMessages := c.convertMessages(messages)

	// Convert tools to OpenAI format
	openAITools := c.convertTools(tools)

	// Create request
	req := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}

	if len(openAITools) > 0 {
		req.Tools = openAITools
	}

	// Create streaming request
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}
	defer stream.Close()

	var toolCalls []openai.ToolCall
	var usage *TokenUsage
	isFirst := true

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Send final chunk with usage data if available
			callback(StreamChunk{
				IsFinal:      true,
				FinishReason: "stop",
				ToolCalls:    convertToolCalls(toolCalls),
				Usage:        usage,
			})
			return nil
		}
		if err != nil {
			return err
		}

		// Capture usage data from response (only in final chunk with StreamOptions)
		if response.Usage != nil {
			usage = &TokenUsage{
				PromptTokens:     response.Usage.PromptTokens,
				CompletionTokens: response.Usage.CompletionTokens,
				TotalTokens:      response.Usage.TotalTokens,
			}
		}

		for _, choice := range response.Choices {
			chunk := StreamChunk{
				IsFirst: isFirst,
			}

			// Handle content delta
			if choice.Delta.Content != "" {
				chunk.Content = choice.Delta.Content
			}

			// Handle tool calls
			// Note: Different providers stream tool calls in different formats:
			// - OpenAI: Streams tool calls incrementally across multiple chunks with an Index
			// - Gemini: Sends complete tool calls in a single chunk without an Index
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Index != nil {
					// OpenAI format: Tool calls have an index for streaming accumulation
					idx := *tc.Index
					for len(toolCalls) <= idx {
						toolCalls = append(toolCalls, openai.ToolCall{})
					}
					if tc.ID != "" {
						toolCalls[idx].ID = tc.ID
					}
					if tc.Type != "" {
						toolCalls[idx].Type = tc.Type
					}
					if tc.Function.Name != "" {
						toolCalls[idx].Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						toolCalls[idx].Function.Arguments += tc.Function.Arguments
					}
				} else {
					// Gemini format: Tool calls come complete without an index
					// Append as a new tool call if it has an ID
					if tc.ID != "" {
						toolCalls = append(toolCalls, openai.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: openai.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						})
					}
				}
			}

			// Check finish reason
			if choice.FinishReason != "" {
				chunk.IsFinal = true
				chunk.FinishReason = string(choice.FinishReason)
				chunk.ToolCalls = convertToolCalls(toolCalls)
			}

			// Call callback
			callback(chunk)
			isFirst = false
		}
	}
}

// convertMessages converts TUI messages to OpenAI format.
func (c *Client) convertMessages(messages []tui.ChatMessage) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage

	// Add system prompt if configured
	if c.systemPrompt != "" && !c.config.SkipPersonaPrompt {
		result = append(result, openai.ChatCompletionMessage{
			Role:    "system",
			Content: c.systemPrompt,
		})
	}

	// Convert messages
	for _, msg := range messages {
		// Skip messages with empty content (except tool calls which can have empty content)
		if msg.Content == "" && len(msg.ToolCalls) == 0 && msg.Role != "tool" {
			// Skip empty messages to prevent API errors (Grok requires content field)
			continue
		}

		if msg.Role == "tool" {
			// Tool messages need special format with tool_call_id
			result = append(result, openai.ChatCompletionMessage{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
			})
		} else if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Assistant messages with tool_calls need to include ToolCalls field
			toolCalls := make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}

			// For tool-calling messages, ensure content is at least empty string (not nil)
			content := msg.Content
			if content == "" {
				content = "" // Explicitly set to empty string for serialization
			}

			result = append(result, openai.ChatCompletionMessage{
				Role:      msg.Role,
				Content:   content,
				ToolCalls: toolCalls,
			})
		} else {
			// Regular messages (user, assistant without tool_calls, system)
			result = append(result, openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return result
}

// convertTools converts TUI skill definitions to OpenAI tools.
func (c *Client) convertTools(tools []tui.SkillDefinition) []openai.Tool {
	var result []openai.Tool

	for _, tool := range tools {
		params, _ := json.Marshal(tool.Parameters)

		result = append(result, openai.Tool{
			Type: "function",
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  json.RawMessage(params),
			},
		})
	}

	return result
}

// convertToolCalls converts OpenAI tool calls to result format.
func convertToolCalls(toolCalls []openai.ToolCall) []ToolCallResult {
	var result []ToolCallResult
	for _, tc := range toolCalls {
		result = append(result, ToolCallResult{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return result
}

// GetSkills returns skill definitions for the TUI.
func (c *Client) GetSkills() []tui.SkillDefinition {
	if c.registry == nil {
		return nil
	}

	allSkills := c.registry.GetAllSkills()
	var result []tui.SkillDefinition

	for _, skill := range allSkills {
		result = append(result, tui.SkillDefinition{
			Name:        skill.Name,
			Description: skill.Description,
			Parameters:  skill.Parameters,
		})
	}

	return result
}

// ExecuteSkill executes a skill and returns the result.
func (c *Client) ExecuteSkill(ctx context.Context, name string, argsJSON string) (*skills.ExecutionResult, error) {
	if c.registry == nil {
		return nil, fmt.Errorf("no skill registry configured")
	}

	executor := skills.NewExecutor(c.registry)
	return executor.Execute(ctx, name, argsJSON)
}
