package skills

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUUIDGeneratorHandler tests UUID generation
func TestUUIDGeneratorHandler(t *testing.T) {
	result, err := UUIDGeneratorHandler(map[string]interface{}{})

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	uuid, ok := resultMap["uuid"].(string)
	require.True(t, ok, "uuid field should be a string")
	assert.Len(t, uuid, 36, "UUID should be 36 characters long")
	assert.Contains(t, uuid, "-", "UUID should contain hyphens")

	// Test that multiple calls generate different UUIDs
	result2, err := UUIDGeneratorHandler(map[string]interface{}{})
	require.NoError(t, err)
	resultMap2 := result2.(map[string]interface{})
	uuid2 := resultMap2["uuid"].(string)
	assert.NotEqual(t, uuid, uuid2, "UUIDs should be unique")
}

// TestBase64EncodeHandler tests base64 encoding
func TestBase64EncodeHandler(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
		hasError bool
	}{
		{
			name:     "simple string",
			input:    map[string]interface{}{"text": "Hello, World!"},
			expected: "SGVsbG8sIFdvcmxkIQ==",
			hasError: false,
		},
		{
			name:     "empty string",
			input:    map[string]interface{}{"text": ""},
			expected: "",
			hasError: true, // Should return error response for empty text
		},
		{
			name:     "missing text parameter",
			input:    map[string]interface{}{},
			expected: "",
			hasError: true,
		},
		{
			name:     "special characters",
			input:    map[string]interface{}{"text": "Hello! @#$%^&*()"},
			expected: "SGVsbG8hIEAjJCVeJiooKQ==",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Base64EncodeHandler(tt.input)
			require.NoError(t, err) // Handler returns formatted errors, not Go errors

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			if tt.hasError {
				// Check for error response format
				_, hasError := resultMap["error_type"]
				assert.True(t, hasError, "should contain error_type field")
			} else {
				encoded, ok := resultMap["encoded"].(string)
				require.True(t, ok, "encoded field should be a string")
				assert.Equal(t, tt.expected, encoded)

				original, ok := resultMap["original"].(string)
				require.True(t, ok, "original field should be a string")
				assert.Equal(t, tt.input["text"], original)
			}
		})
	}
}

// TestBase64DecodeHandler tests base64 decoding
func TestBase64DecodeHandler(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
		hasError bool
	}{
		{
			name:     "simple string",
			input:    map[string]interface{}{"encoded": "SGVsbG8sIFdvcmxkIQ=="},
			expected: "Hello, World!",
			hasError: false,
		},
		{
			name:     "invalid base64",
			input:    map[string]interface{}{"encoded": "not-valid-base64!!!"},
			expected: "",
			hasError: true,
		},
		{
			name:     "missing encoded parameter",
			input:    map[string]interface{}{},
			expected: "",
			hasError: true,
		},
		{
			name:     "special characters",
			input:    map[string]interface{}{"encoded": "SGVsbG8hIEAjJCVeJiooKQ=="},
			expected: "Hello! @#$%^&*()",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Base64DecodeHandler(tt.input)
			require.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			if tt.hasError {
				_, hasError := resultMap["error_type"]
				assert.True(t, hasError, "should contain error_type field")
			} else {
				decoded, ok := resultMap["decoded"].(string)
				require.True(t, ok, "decoded field should be a string")
				assert.Equal(t, tt.expected, decoded)
			}
		})
	}
}

// TestBase64RoundTrip tests encoding and then decoding
func TestBase64RoundTrip(t *testing.T) {
	originalText := "The quick brown fox jumps over the lazy dog"

	// Encode
	encodeResult, err := Base64EncodeHandler(map[string]interface{}{"text": originalText})
	require.NoError(t, err)
	encodeMap := encodeResult.(map[string]interface{})
	encoded := encodeMap["encoded"].(string)

	// Decode
	decodeResult, err := Base64DecodeHandler(map[string]interface{}{"encoded": encoded})
	require.NoError(t, err)
	decodeMap := decodeResult.(map[string]interface{})
	decoded := decodeMap["decoded"].(string)

	assert.Equal(t, originalText, decoded, "round trip should preserve original text")
}

// TestHashGeneratorHandler tests hash generation
func TestHashGeneratorHandler(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		algorithm string
		expected  string // For known test vectors
		hasError  bool
	}{
		{
			name:      "MD5",
			input:     map[string]interface{}{"text": "Hello", "algorithm": "md5"},
			algorithm: "md5",
			expected:  "8b1a9953c4611296a827abf8c47804d7",
			hasError:  false,
		},
		{
			name:      "SHA256",
			input:     map[string]interface{}{"text": "Hello", "algorithm": "sha256"},
			algorithm: "sha256",
			expected:  "185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
			hasError:  false,
		},
		{
			name:      "SHA512",
			input:     map[string]interface{}{"text": "Hello", "algorithm": "sha512"},
			algorithm: "sha512",
			expected:  "3615f80c9d293ed7402687f94b22d58e529b8cc7916f8fac7fddf7fbd5af4cf777d3d795a7a00a16bf7e7f3fb9561ee9baae480da9fe7a18769e71886b03f315",
			hasError:  false,
		},
		{
			name:     "missing text",
			input:    map[string]interface{}{"algorithm": "md5"},
			hasError: true,
		},
		{
			name:     "missing algorithm",
			input:    map[string]interface{}{"text": "Hello"},
			hasError: true,
		},
		{
			name:     "invalid algorithm",
			input:    map[string]interface{}{"text": "Hello", "algorithm": "invalid"},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HashGeneratorHandler(tt.input)
			require.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			if tt.hasError {
				_, hasError := resultMap["error_type"]
				assert.True(t, hasError, "should contain error_type field")
			} else {
				hash, ok := resultMap["hash"].(string)
				require.True(t, ok, "hash field should be a string")
				assert.Equal(t, tt.expected, hash)

				algorithm, ok := resultMap["algorithm"].(string)
				require.True(t, ok, "algorithm field should be a string")
				assert.Equal(t, tt.algorithm, algorithm)
			}
		})
	}
}

// TestPasswordGeneratorHandler tests password generation
func TestPasswordGeneratorHandler(t *testing.T) {
	tests := []struct {
		name            string
		input           map[string]interface{}
		expectedLength  int
		shouldHaveUpper bool
		shouldHaveLower bool
		shouldHaveDigit bool
		shouldHaveSymbol bool
	}{
		{
			name:             "default settings",
			input:            map[string]interface{}{},
			expectedLength:   16,
			shouldHaveUpper:  true,
			shouldHaveLower:  true,
			shouldHaveDigit:  true,
			shouldHaveSymbol: true,
		},
		{
			name:             "custom length",
			input:            map[string]interface{}{"length": float64(32)},
			expectedLength:   32,
			shouldHaveUpper:  true,
			shouldHaveLower:  true,
			shouldHaveDigit:  true,
			shouldHaveSymbol: true,
		},
		{
			name:             "no symbols",
			input:            map[string]interface{}{"include_symbols": false},
			expectedLength:   16,
			shouldHaveUpper:  true,
			shouldHaveLower:  true,
			shouldHaveDigit:  true,
			shouldHaveSymbol: false,
		},
		{
			name:             "no numbers",
			input:            map[string]interface{}{"include_numbers": false},
			expectedLength:   16,
			shouldHaveUpper:  true,
			shouldHaveLower:  true,
			shouldHaveDigit:  false,
			shouldHaveSymbol: true,
		},
		{
			name:             "minimum length",
			input:            map[string]interface{}{"length": float64(8)},
			expectedLength:   8,
			shouldHaveUpper:  true,
			shouldHaveLower:  true,
			shouldHaveDigit:  true,
			shouldHaveSymbol: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PasswordGeneratorHandler(tt.input)
			require.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			password, ok := resultMap["password"].(string)
			require.True(t, ok, "password field should be a string")
			assert.Len(t, password, tt.expectedLength)

			// Check character requirements
			hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
			hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
			hasDigit := strings.ContainsAny(password, "0123456789")
			hasSymbol := strings.ContainsAny(password, "!@#$%^&*()-_=+[]{}|;:,.<>?")

			if tt.shouldHaveUpper {
				assert.True(t, hasUpper, "password should contain uppercase letters")
			}
			if tt.shouldHaveLower {
				assert.True(t, hasLower, "password should contain lowercase letters")
			}
			if tt.shouldHaveDigit {
				assert.True(t, hasDigit, "password should contain digits")
			}
			if tt.shouldHaveSymbol {
				assert.True(t, hasSymbol, "password should contain symbols")
			}

			// Test uniqueness - generate multiple passwords
			result2, err := PasswordGeneratorHandler(tt.input)
			require.NoError(t, err)
			resultMap2 := result2.(map[string]interface{})
			password2 := resultMap2["password"].(string)
			assert.NotEqual(t, password, password2, "passwords should be unique")
		})
	}
}

// TestContentHandler tests content generation
func TestContentHandler(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		platform string
		format   string
		tone     string
	}{
		{
			name:     "default settings",
			input:    map[string]interface{}{"topic": "AI"},
			platform: "twitter",
			format:   "short",
			tone:     "teasing",
		},
		{
			name:     "custom platform",
			input:    map[string]interface{}{"platform": "youtube", "topic": "Gaming"},
			platform: "youtube",
			format:   "short",
			tone:     "teasing",
		},
		{
			name:     "custom format and tone",
			input:    map[string]interface{}{"format": "long", "tone": "professional", "topic": "Tech"},
			platform: "twitter",
			format:   "long",
			tone:     "professional",
		},
		{
			name:     "tiktok content",
			input:    map[string]interface{}{"platform": "tiktok", "topic": "Dance", "tone": "fun"},
			platform: "tiktok",
			format:   "short",
			tone:     "fun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ContentHandler(tt.input)
			require.NoError(t, err)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))
			assert.Equal(t, tt.platform, resultMap["platform"])
			assert.Equal(t, tt.format, resultMap["format"])
			assert.Equal(t, tt.tone, resultMap["tone"])

			prompt, ok := resultMap["prompt"].(string)
			require.True(t, ok, "prompt should be a string")
			assert.NotEmpty(t, prompt)
			assert.Contains(t, prompt, tt.platform)
		})
	}
}

// TestFormatErrorResponse tests error response formatting
func TestFormatErrorResponse(t *testing.T) {
	result := formatErrorResponse(
		"test_error",
		"This is a test error",
		"Please fix the issue",
		map[string]interface{}{"field": "test"},
	)

	resultMap := result
	assert.Equal(t, "test_error", resultMap["error_type"])
	assert.Equal(t, true, resultMap["error"])                    // error field is boolean
	assert.Equal(t, "This is a test error", resultMap["message"]) // message field has the text
	assert.Equal(t, "Please fix the issue", resultMap["hint"])    // hint not suggestion

	// Context is merged directly into result
	assert.Equal(t, "test", resultMap["field"])
}
