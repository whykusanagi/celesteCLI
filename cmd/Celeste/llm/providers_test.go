// Package llm provides tests for LLM provider function calling compatibility.
package llm

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAI_FunctionCalling tests OpenAI's native function calling support.
// This verifies that the LLM actually calls the function (not hallucinating).
func TestOpenAI_FunctionCalling(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping OpenAI function calling test")
	}

	client := openai.NewClient(apiKey)

	// Define a simple test skill
	tools := []openai.Tool{{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_weather",
			Description: "Get current weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or zip code",
					},
				},
				"required": []string{"location"},
			},
		},
	}}

	// Ask LLM to call the function
	resp, err := client.CreateChatCompletion(t.Name(), openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What's the weather in New York City?",
			},
		},
		Tools: tools,
	})

	require.NoError(t, err, "OpenAI API call failed")
	require.NotEmpty(t, resp.Choices, "No choices returned from OpenAI")

	choice := resp.Choices[0]
	require.NotEmpty(t, choice.Message.ToolCalls, "LLM did not call any functions - possible hallucination")

	// Verify the function call
	toolCall := choice.Message.ToolCalls[0]
	assert.Equal(t, "get_weather", toolCall.Function.Name, "Wrong function called")

	// Parse and verify arguments
	var args map[string]interface{}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	require.NoError(t, err, "Failed to parse function arguments")

	location, ok := args["location"].(string)
	require.True(t, ok, "Location argument not found or wrong type")
	assert.Contains(t, strings.ToLower(location), "new york", "LLM didn't extract location correctly")

	t.Logf("✅ OpenAI function calling works! Called %s with location=%s", toolCall.Function.Name, location)
}

// TestGrok_FunctionCalling tests Grok (xAI) function calling support.
// Grok uses OpenAI-compatible API, so this should work similarly.
func TestGrok_FunctionCalling(t *testing.T) {
	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		t.Skip("GROK_API_KEY not set, skipping Grok function calling test")
	}

	// Grok uses OpenAI-compatible API
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.x.ai/v1"
	client := openai.NewClientWithConfig(config)

	// Define a simple test skill
	tools := []openai.Tool{{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_weather",
			Description: "Get current weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or zip code",
					},
				},
				"required": []string{"location"},
			},
		},
	}}

	// Ask LLM to call the function
	resp, err := client.CreateChatCompletion(t.Name(), openai.ChatCompletionRequest{
		Model: "grok-beta",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What's the weather in San Francisco?",
			},
		},
		Tools: tools,
	})

	require.NoError(t, err, "Grok API call failed")
	require.NotEmpty(t, resp.Choices, "No choices returned from Grok")

	choice := resp.Choices[0]
	require.NotEmpty(t, choice.Message.ToolCalls, "Grok did not call any functions - possible hallucination or unsupported feature")

	// Verify the function call
	toolCall := choice.Message.ToolCalls[0]
	assert.Equal(t, "get_weather", toolCall.Function.Name, "Wrong function called")

	// Parse and verify arguments
	var args map[string]interface{}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	require.NoError(t, err, "Failed to parse function arguments")

	location, ok := args["location"].(string)
	require.True(t, ok, "Location argument not found or wrong type")
	assert.Contains(t, strings.ToLower(location), "san francisco", "Grok didn't extract location correctly")

	t.Logf("✅ Grok function calling works! Called %s with location=%s", toolCall.Function.Name, location)
}

// TestDigitalOcean_FunctionCalling documents DigitalOcean's limitations.
// DigitalOcean requires cloud-hosted functions with route attachment.
func TestDigitalOcean_FunctionCalling(t *testing.T) {
	t.Skip(`DigitalOcean AI Agent requires cloud-hosted functions with route attachment.

Skills will NOT work with DigitalOcean without custom setup:
1. Deploy each skill as a cloud function
2. Attach function URLs to the agent via API
3. Agent calls the URLs directly (not local execution)

This is fundamentally different from OpenAI/Grok's approach where:
- Functions are defined in the API request
- LLM decides when to call them
- Functions execute locally

For DigitalOcean, use alternative approaches:
- Manual skill invocation (not AI-driven)
- Migrate to OpenAI-compatible provider
- Deploy skills as cloud functions`)
}

// TestVeniceAI_FunctionCalling tests Venice.ai function calling support.
func TestVeniceAI_FunctionCalling(t *testing.T) {
	apiKey := os.Getenv("VENICE_API_KEY")
	if apiKey == "" {
		t.Skip("VENICE_API_KEY not set, skipping Venice.ai function calling test")
	}

	// Venice.ai may or may not support OpenAI-compatible function calling
	// This test will determine compatibility
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.venice.ai/api/v1"
	client := openai.NewClientWithConfig(config)

	tools := []openai.Tool{{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_weather",
			Description: "Get current weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or zip code",
					},
				},
				"required": []string{"location"},
			},
		},
	}}

	resp, err := client.CreateChatCompletion(t.Name(), openai.ChatCompletionRequest{
		Model: "venice-uncensored",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What's the weather in Los Angeles?",
			},
		},
		Tools: tools,
	})

	if err != nil {
		t.Logf("⚠️ Venice.ai function calling failed: %v", err)
		t.Logf("Venice.ai may not support OpenAI-style function calling")
		t.Skip("Venice.ai function calling not supported or requires different format")
	}

	require.NotEmpty(t, resp.Choices, "No choices returned from Venice.ai")

	choice := resp.Choices[0]
	if len(choice.Message.ToolCalls) == 0 {
		t.Log("⚠️ Venice.ai returned response but did not call functions")
		t.Log("Venice.ai likely does not support function calling")
		t.Skip("Venice.ai does not support function calling")
	}

	t.Logf("✅ Venice.ai function calling works!")
}
