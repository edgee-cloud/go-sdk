package edgee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Save original environment variables
	originalAPIKey := os.Getenv("EDGEE_API_KEY")
	originalBaseURL := os.Getenv("EDGEE_BASE_URL")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("EDGEE_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("EDGEE_API_KEY")
		}
		if originalBaseURL != "" {
			os.Setenv("EDGEE_BASE_URL", originalBaseURL)
		} else {
			os.Unsetenv("EDGEE_BASE_URL")
		}
	}()

	t.Run("with string API key", func(t *testing.T) {
		os.Unsetenv("EDGEE_API_KEY")
		os.Unsetenv("EDGEE_BASE_URL")

		client, err := NewClient("test-api-key")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})

	t.Run("with empty string API key", func(t *testing.T) {
		os.Unsetenv("EDGEE_API_KEY")
		os.Unsetenv("EDGEE_BASE_URL")

		_, err := NewClient("")
		if err == nil {
			t.Fatal("Expected error for empty API key")
		}
		if !strings.Contains(err.Error(), "EDGEE_API_KEY is not set") {
			t.Errorf("Expected error about EDGEE_API_KEY, got %v", err)
		}
	})

	t.Run("with Config struct", func(t *testing.T) {
		os.Unsetenv("EDGEE_API_KEY")
		os.Unsetenv("EDGEE_BASE_URL")

		config := &Config{
			APIKey:  "test-key",
			BaseURL: "https://custom.example.com",
		}
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})

	t.Run("with Config struct and empty baseURL uses default", func(t *testing.T) {
		os.Unsetenv("EDGEE_API_KEY")
		os.Unsetenv("EDGEE_BASE_URL")

		config := &Config{
			APIKey: "test-key",
		}
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})

	t.Run("with nil uses environment variables", func(t *testing.T) {
		os.Setenv("EDGEE_API_KEY", "env-api-key")
		os.Setenv("EDGEE_BASE_URL", "https://env-base-url.example.com")

		client, err := NewClient(nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})

	t.Run("with nil and no env vars fails", func(t *testing.T) {
		os.Unsetenv("EDGEE_API_KEY")
		os.Unsetenv("EDGEE_BASE_URL")

		_, err := NewClient(nil)
		if err == nil {
			t.Fatal("Expected error when no API key provided")
		}
		if !strings.Contains(err.Error(), "EDGEE_API_KEY is not set") {
			t.Errorf("Expected error about EDGEE_API_KEY, got %v", err)
		}
	})

	t.Run("with nil and only EDGEE_API_KEY uses default baseURL", func(t *testing.T) {
		os.Setenv("EDGEE_API_KEY", "env-api-key")
		os.Unsetenv("EDGEE_BASE_URL")

		client, err := NewClient(nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})

	t.Run("with unsupported config type", func(t *testing.T) {
		_, err := NewClient(123)
		if err == nil {
			t.Fatal("Expected error for unsupported config type")
		}
		if !strings.Contains(err.Error(), "unsupported config type") {
			t.Errorf("Expected error about unsupported config type, got %v", err)
		}
	})

	t.Run("with Config struct and empty APIKey falls back to env", func(t *testing.T) {
		os.Setenv("EDGEE_API_KEY", "env-api-key")
		os.Unsetenv("EDGEE_BASE_URL")

		config := &Config{
			APIKey: "",
		}
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client == nil {
			t.Fatal("Expected client, got nil")
		}
	})
}

func TestClient_Send(t *testing.T) {
	t.Run("with string input", func(t *testing.T) {
		mockResponse := SendResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Hello, world!",
					},
					FinishReason: stringPtr("stop"),
				},
			},
			Usage: &Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/chat/completions" {
				t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
			}

			var req Request
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if req.Model != "gpt-4" {
				t.Errorf("Expected model gpt-4, got %s", req.Model)
			}
			if len(req.Messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(req.Messages))
			}
			if req.Messages[0].Role != "user" || req.Messages[0].Content != "Hello" {
				t.Errorf("Expected user message 'Hello', got %+v", req.Messages[0])
			}
			if req.Stream {
				t.Error("Expected stream=false")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		response, err := client.Send("gpt-4", "Hello")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if response.Choices[0].Message.Content != "Hello, world!" {
			t.Errorf("Expected 'Hello, world!', got %s", response.Choices[0].Message.Content)
		}
		if response.Usage.TotalTokens != 15 {
			t.Errorf("Expected 15 total tokens, got %d", response.Usage.TotalTokens)
		}
	})

	t.Run("with InputObject", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)

			if len(req.Messages) != 2 {
				t.Errorf("Expected 2 messages, got %d", len(req.Messages))
			}
			if req.Messages[0].Role != "system" {
				t.Errorf("Expected first message role 'system', got %s", req.Messages[0].Role)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		input := InputObject{
			Messages: []Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
		}

		response, err := client.Send("gpt-4", input)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if response.Choices[0].Message.Content != "Response" {
			t.Errorf("Expected 'Response', got %s", response.Choices[0].Message.Content)
		}
	})

	t.Run("with InputObject pointer", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		input := &InputObject{
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		_, err := client.Send("gpt-4", input)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("with map[string]any", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)

			if len(req.Messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(req.Messages))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		input := map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": "Hello"},
			},
		}

		_, err := client.Send("gpt-4", input)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("with tools", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "",
						ToolCalls: []ToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: FunctionCall{
									Name:      "get_weather",
									Arguments: `{"location": "San Francisco"}`,
								},
							},
						},
					},
					FinishReason: stringPtr("tool_calls"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)

			if len(req.Tools) != 1 {
				t.Errorf("Expected 1 tool, got %d", len(req.Tools))
			}
			if req.Tools[0].Function.Name != "get_weather" {
				t.Errorf("Expected tool name 'get_weather', got %s", req.Tools[0].Function.Name)
			}
			if req.ToolChoice != "auto" {
				t.Errorf("Expected tool_choice 'auto', got %v", req.ToolChoice)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		tools := []Tool{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_weather",
					Description: stringPtr("Get the weather for a location"),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
					},
				},
			},
		}

		input := InputObject{
			Messages: []Message{
				{Role: "user", Content: "What is the weather?"},
			},
			Tools:      tools,
			ToolChoice: "auto",
		}

		response, err := client.Send("gpt-4", input)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(response.Choices[0].Message.ToolCalls) == 0 {
			t.Error("Expected tool calls in response")
		}
	})

	t.Run("with tool_choice object", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "",
					},
					FinishReason: stringPtr("tool_calls"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)

			toolChoiceMap, ok := req.ToolChoice.(map[string]any)
			if !ok {
				t.Errorf("Expected tool_choice to be map, got %T", req.ToolChoice)
			}
			if toolChoiceMap["type"] != "function" {
				t.Errorf("Expected tool_choice type 'function', got %v", toolChoiceMap["type"])
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		input := InputObject{
			Messages: []Message{
				{Role: "user", Content: "Test"},
			},
			ToolChoice: map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": "specific_function",
				},
			},
		}

		_, err := client.Send("gpt-4", input)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("without usage field", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		response, err := client.Send("gpt-4", "Test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if response.Usage != nil {
			t.Error("Expected nil usage")
		}
		if len(response.Choices) != 1 {
			t.Errorf("Expected 1 choice, got %d", len(response.Choices))
		}
	})

	t.Run("with multiple choices", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "First response",
					},
					FinishReason: stringPtr("stop"),
				},
				{
					Index: 1,
					Message: &Message{
						Role:    "assistant",
						Content: "Second response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		response, err := client.Send("gpt-4", "Test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(response.Choices) != 2 {
			t.Errorf("Expected 2 choices, got %d", len(response.Choices))
		}
		if response.Choices[0].Message.Content != "First response" {
			t.Errorf("Expected 'First response', got %s", response.Choices[0].Message.Content)
		}
		if response.Choices[1].Message.Content != "Second response" {
			t.Errorf("Expected 'Second response', got %s", response.Choices[1].Message.Content)
		}
	})

	t.Run("with custom baseURL", func(t *testing.T) {
		mockResponse := SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: stringPtr("stop"),
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		_, err := client.Send("gpt-4", "Test")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("with API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		_, err := client.Send("gpt-4", "Test")
		if err == nil {
			t.Fatal("Expected error for 401 status")
		}
		if !strings.Contains(err.Error(), "API error 401") {
			t.Errorf("Expected error about API error 401, got %v", err)
		}
	})

	t.Run("with 500 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		_, err := client.Send("gpt-4", "Test")
		if err == nil {
			t.Fatal("Expected error for 500 status")
		}
		if !strings.Contains(err.Error(), "API error 500") {
			t.Errorf("Expected error about API error 500, got %v", err)
		}
	})

	t.Run("with unsupported input type", func(t *testing.T) {
		client, _ := NewClient(&Config{
			APIKey: "test-api-key",
		})

		_, err := client.Send("gpt-4", 123)
		if err == nil {
			t.Fatal("Expected error for unsupported input type")
		}
		if !strings.Contains(err.Error(), "unsupported input type") {
			t.Errorf("Expected error about unsupported input type, got %v", err)
		}
	})
}

func TestSendResponse_ConvenienceMethods(t *testing.T) {
	t.Run("Text method", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:    "assistant",
						Content: "Hello, world!",
					},
				},
			},
		}

		if response.Text() != "Hello, world!" {
			t.Errorf("Expected 'Hello, world!', got %s", response.Text())
		}
	})

	t.Run("Text method with empty choices", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{},
		}

		if response.Text() != "" {
			t.Errorf("Expected empty string, got %s", response.Text())
		}
	})

	t.Run("Text method with nil message", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{
				{
					Index:   0,
					Message: nil,
				},
			},
		}

		if response.Text() != "" {
			t.Errorf("Expected empty string, got %s", response.Text())
		}
	})

	t.Run("MessageContent method", func(t *testing.T) {
		msg := &Message{
			Role:    "assistant",
			Content: "Hello, world!",
		}
		response := &SendResponse{
			Choices: []Choice{
				{
					Index:   0,
					Message: msg,
				},
			},
		}

		if response.MessageContent() != msg {
			t.Error("Expected message to match")
		}
	})

	t.Run("MessageContent method with empty choices", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{},
		}

		if response.MessageContent() != nil {
			t.Error("Expected nil message")
		}
	})

	t.Run("FinishReason method", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{
				{
					Index:        0,
					FinishReason: stringPtr("stop"),
				},
			},
		}

		if response.FinishReason() != "stop" {
			t.Errorf("Expected 'stop', got %s", response.FinishReason())
		}
	})

	t.Run("FinishReason method with nil", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{
				{
					Index:        0,
					FinishReason: nil,
				},
			},
		}

		if response.FinishReason() != "" {
			t.Errorf("Expected empty string, got %s", response.FinishReason())
		}
	})

	t.Run("ToolCalls method", func(t *testing.T) {
		toolCalls := []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "get_weather",
					Arguments: `{"location": "San Francisco"}`,
				},
			},
		}
		response := &SendResponse{
			Choices: []Choice{
				{
					Index: 0,
					Message: &Message{
						Role:      "assistant",
						ToolCalls: toolCalls,
					},
				},
			},
		}

		result := response.ToolCalls()
		if len(result) != 1 {
			t.Errorf("Expected 1 tool call, got %d", len(result))
		}
		if result[0].ID != "call_123" {
			t.Errorf("Expected tool call ID 'call_123', got %s", result[0].ID)
		}
	})

	t.Run("ToolCalls method with empty choices", func(t *testing.T) {
		response := &SendResponse{
			Choices: []Choice{},
		}

		if response.ToolCalls() != nil {
			t.Error("Expected nil tool calls")
		}
	})
}

func TestClient_ChatCompletion(t *testing.T) {
	mockResponse := SendResponse{
		Choices: []Choice{
			{
				Index: 0,
				Message: &Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: stringPtr("stop"),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client, _ := NewClient(&Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})

	result, err := client.ChatCompletion("gpt-4", "Hello")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Choices[0].Message.Content != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got %s", result.Choices[0].Message.Content)
	}
}

func TestClient_Stream(t *testing.T) {
	t.Run("with string input", func(t *testing.T) {
		mockChunks := []string{
			`{"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)

			if !req.Stream {
				t.Error("Expected stream=true")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			for _, chunk := range mockChunks {
				fmt.Fprintf(w, "data: %s\n\n", chunk)
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		eventChan, errChan := client.Stream("gpt-4", "Hello")

		chunks := []*StreamChunk{}
		for event := range eventChan {
			if event.Type == StreamEventChunk && event.Chunk != nil {
				chunks = append(chunks, event.Chunk)
			}
		}

		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		default:
		}

		if len(chunks) != 4 {
			t.Errorf("Expected 4 chunks, got %d", len(chunks))
			return
		}
		if chunks[0].Role() != "assistant" {
			t.Errorf("Expected role 'assistant', got %s", chunks[0].Role())
		}
		if chunks[1].Text() != "Hello" {
			t.Errorf("Expected 'Hello', got %s", chunks[1].Text())
		}
		if chunks[2].Text() != " world" {
			t.Errorf("Expected ' world', got %s", chunks[2].Text())
		}
		if chunks[3].FinishReason() != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %s", chunks[3].FinishReason())
		}
	})

	t.Run("with InputObject", func(t *testing.T) {
		mockChunk := `{"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Response"},"finish_reason":null}]}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: %s\n\n", mockChunk)
			fmt.Fprintf(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		input := InputObject{
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		eventChan, errChan := client.Stream("gpt-4", input)

		chunks := []*StreamChunk{}
		for event := range eventChan {
			if event.Type == StreamEventChunk && event.Chunk != nil {
				chunks = append(chunks, event.Chunk)
			}
		}

		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		default:
		}

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Text() != "Response" {
			t.Errorf("Expected 'Response', got %s", chunks[0].Text())
		}
	})

	t.Run("with streaming error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded"))
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		eventChan, errChan := client.Stream("gpt-4", "Hello")

		// Wait for error
		err := <-errChan
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "API error 429") {
			t.Errorf("Expected error about API error 429, got %v", err)
		}

		// Channel should be closed
		_, ok := <-eventChan
		if ok {
			t.Error("Expected event channel to be closed")
		}
	})

	t.Run("skips malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {invalid json}\n\n")
			fmt.Fprintf(w, `data: {"id":"test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Valid"},"finish_reason":null}]}`+"\n\n")
			fmt.Fprintf(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		eventChan, errChan := client.Stream("gpt-4", "Hello")

		chunks := []*StreamChunk{}
		for event := range eventChan {
			if event.Type == StreamEventChunk && event.Chunk != nil {
				chunks = append(chunks, event.Chunk)
			}
		}

		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		default:
		}

		// Should skip the malformed JSON and only return the valid chunk
		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
		if len(chunks) > 0 && chunks[0].Text() != "Valid" {
			t.Errorf("Expected 'Valid', got %s", chunks[0].Text())
		}
	})
}

func TestStreamChunk_ConvenienceMethods(t *testing.T) {
	t.Run("Text method", func(t *testing.T) {
		content := "Hello"
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: &StreamDelta{
						Content: &content,
					},
				},
			},
		}

		if chunk.Text() != "Hello" {
			t.Errorf("Expected 'Hello', got %s", chunk.Text())
		}
	})

	t.Run("Text method with nil delta", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: nil,
				},
			},
		}

		if chunk.Text() != "" {
			t.Errorf("Expected empty string, got %s", chunk.Text())
		}
	})

	t.Run("Text method with nil content", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: &StreamDelta{
						Content: nil,
					},
				},
			},
		}

		if chunk.Text() != "" {
			t.Errorf("Expected empty string, got %s", chunk.Text())
		}
	})

	t.Run("Role method", func(t *testing.T) {
		role := "assistant"
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: &StreamDelta{
						Role: &role,
					},
				},
			},
		}

		if chunk.Role() != "assistant" {
			t.Errorf("Expected 'assistant', got %s", chunk.Role())
		}
	})

	t.Run("Role method with nil", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: &StreamDelta{
						Role: nil,
					},
				},
			},
		}

		if chunk.Role() != "" {
			t.Errorf("Expected empty string, got %s", chunk.Role())
		}
	})

	t.Run("FinishReason method", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index:        0,
					FinishReason: stringPtr("stop"),
				},
			},
		}

		if chunk.FinishReason() != "stop" {
			t.Errorf("Expected 'stop', got %s", chunk.FinishReason())
		}
	})

	t.Run("FinishReason method with nil", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{
				{
					Index:        0,
					FinishReason: nil,
				},
			},
		}

		if chunk.FinishReason() != "" {
			t.Errorf("Expected empty string, got %s", chunk.FinishReason())
		}
	})

	t.Run("FinishReason method with empty choices", func(t *testing.T) {
		chunk := &StreamChunk{
			Choices: []StreamChoice{},
		}

		if chunk.FinishReason() != "" {
			t.Errorf("Expected empty string, got %s", chunk.FinishReason())
		}
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
