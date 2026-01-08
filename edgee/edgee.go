// Package edgee provides a lightweight Go SDK for Edgee AI Gateway
package edgee

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	// DefaultBaseURL is the default base URL for the Edgee API
	DefaultBaseURL = "https://api.edgee.ai"
	// APIEndpoint is the chat completions endpoint
	APIEndpoint = "/v1/chat/completions"
)

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       *string    `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID *string    `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call request from the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a function tool definition
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a function tool
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// InputObject represents structured input for chat completion
type InputObject struct {
	Messages   []Message   `json:"messages"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"` // string or object
}

// ChatCompletionRequest represents the request body for chat completions
type ChatCompletionRequest struct {
	Model      string      `json:"model"`
	Messages   []Message   `json:"messages"`
	Stream     bool        `json:"stream,omitempty"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// ChatCompletionDelta represents a streaming chunk delta
type ChatCompletionDelta struct {
	Role      *string    `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatCompletionChoice represents a choice in the response
type ChatCompletionChoice struct {
	Index        int                  `json:"index"`
	Message      *Message             `json:"message,omitempty"`
	Delta        *ChatCompletionDelta `json:"delta,omitempty"`
	FinishReason *string              `json:"finish_reason,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SendResponse represents the response from a non-streaming request
type SendResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *Usage                 `json:"usage,omitempty"`
}

// Text returns the text content from the first choice (convenience method)
func (r *SendResponse) Text() string {
	if len(r.Choices) > 0 && r.Choices[0].Message != nil {
		return r.Choices[0].Message.Content
	}
	return ""
}

// MessageContent returns the message from the first choice (convenience method)
func (r *SendResponse) MessageContent() *Message {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message
	}
	return nil
}

// FinishReason returns the finish reason from the first choice (convenience method)
func (r *SendResponse) FinishReason() string {
	if len(r.Choices) > 0 && r.Choices[0].FinishReason != nil {
		return *r.Choices[0].FinishReason
	}
	return ""
}

// ToolCalls returns the tool calls from the first choice (convenience method)
func (r *SendResponse) ToolCalls() []ToolCall {
	if len(r.Choices) > 0 && r.Choices[0].Message != nil {
		return r.Choices[0].Message.ToolCalls
	}
	return nil
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

// Text returns the text content from the first choice (convenience method)
func (c *StreamChunk) Text() string {
	if len(c.Choices) > 0 && c.Choices[0].Delta != nil && c.Choices[0].Delta.Content != nil {
		return *c.Choices[0].Delta.Content
	}
	return ""
}

// Role returns the role from the first choice (convenience method)
func (c *StreamChunk) Role() string {
	if len(c.Choices) > 0 && c.Choices[0].Delta != nil && c.Choices[0].Delta.Role != nil {
		return *c.Choices[0].Delta.Role
	}
	return ""
}

// FinishReason returns the finish reason from the first choice (convenience method)
func (c *StreamChunk) FinishReason() string {
	if len(c.Choices) > 0 && c.Choices[0].FinishReason != nil {
		return *c.Choices[0].FinishReason
	}
	return ""
}

// Config represents configuration for the Edgee client
type Config struct {
	APIKey  string
	BaseURL string
}

// Client represents an Edgee AI Gateway client
type Client struct {
	apiKey  string
	baseURL string
}

// NewClient creates a new Edgee client with flexible configuration:
// - Pass a string to set the API key directly
// - Pass a *Config to set both API key and base URL
// - Pass nil to use environment variables (EDGEE_API_KEY, EDGEE_BASE_URL)
func NewClient(config interface{}) (*Client, error) {
	var apiKey, baseURL string

	switch v := config.(type) {
	case string:
		// String input: use as API key
		apiKey = v
	case *Config:
		// Config struct
		apiKey = v.APIKey
		baseURL = v.BaseURL
	case nil:
		// Use environment variables
		apiKey = os.Getenv("EDGEE_API_KEY")
		baseURL = os.Getenv("EDGEE_BASE_URL")
	default:
		return nil, fmt.Errorf("unsupported config type: %T", config)
	}

	// Fall back to environment variables if not set
	if apiKey == "" {
		apiKey = os.Getenv("EDGEE_API_KEY")
	}
	if baseURL == "" {
		baseURL = os.Getenv("EDGEE_BASE_URL")
	}

	// Use default base URL if still not set
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// API key is required
	if apiKey == "" {
		return nil, fmt.Errorf("EDGEE_API_KEY is not set")
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

// Send sends a chat completion request with flexible input:
// - Pass a string for simple user input
// - Pass an InputObject for full control
// - Pass a map[string]interface{} with "messages", "tools", "tool_choice" keys
func (c *Client) Send(model string, input interface{}, stream bool) (interface{}, error) {
	req, err := c.buildRequest(model, input, stream)
	if err != nil {
		return nil, err
	}

	if stream {
		return c.handleStreamingResponse(req)
	}
	return c.handleNonStreamingResponse(req)
}

// ChatCompletion sends a non-streaming chat completion request (convenience method)
func (c *Client) ChatCompletion(model string, input interface{}) (*SendResponse, error) {
	result, err := c.Send(model, input, false)
	if err != nil {
		return nil, err
	}
	return result.(*SendResponse), nil
}

// Stream sends a streaming chat completion request (convenience method)
func (c *Client) Stream(model string, input interface{}) (<-chan *StreamChunk, <-chan error) {
	result, err := c.Send(model, input, true)
	if err != nil {
		errChan := make(chan error, 1)
		errChan <- err
		close(errChan)
		chunkChan := make(chan *StreamChunk)
		close(chunkChan)
		return chunkChan, errChan
	}

	streamResult := result.(struct {
		ChunkChan <-chan *StreamChunk
		ErrChan   <-chan error
	})
	return streamResult.ChunkChan, streamResult.ErrChan
}

// StreamText sends a streaming request and returns only text content (convenience method)
func (c *Client) StreamText(model string, input interface{}) (<-chan string, <-chan error) {
	textChan := make(chan string, 10)
	errChan := make(chan error, 1)

	chunkChan, chunkErrChan := c.Stream(model, input)

	go func() {
		defer close(textChan)
		defer close(errChan)

		for {
			select {
			case chunk, ok := <-chunkChan:
				if !ok {
					return
				}
				if text := chunk.Text(); text != "" {
					textChan <- text
				}
			case err, ok := <-chunkErrChan:
				if ok && err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	return textChan, errChan
}

func (c *Client) buildRequest(model string, input interface{}, stream bool) (*ChatCompletionRequest, error) {
	req := &ChatCompletionRequest{
		Model:  model,
		Stream: stream,
	}

	switch v := input.(type) {
	case string:
		// Simple string input
		req.Messages = []Message{{Role: "user", Content: v}}
	case InputObject:
		req.Messages = v.Messages
		req.Tools = v.Tools
		req.ToolChoice = v.ToolChoice
	case *InputObject:
		req.Messages = v.Messages
		req.Tools = v.Tools
		req.ToolChoice = v.ToolChoice
	case map[string]interface{}:
		// Map input
		if messages, ok := v["messages"]; ok {
			msgBytes, err := json.Marshal(messages)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal messages: %w", err)
			}
			if err := json.Unmarshal(msgBytes, &req.Messages); err != nil {
				return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
			}
		}
		if tools, ok := v["tools"]; ok {
			toolBytes, err := json.Marshal(tools)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tools: %w", err)
			}
			if err := json.Unmarshal(toolBytes, &req.Tools); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
			}
		}
		if toolChoice, ok := v["tool_choice"]; ok {
			req.ToolChoice = toolChoice
		}
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}

	return req, nil
}

func (c *Client) handleNonStreamingResponse(req *ChatCompletionRequest) (*SendResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) handleStreamingResponse(req *ChatCompletionRequest) (interface{}, error) {
	chunkChan := make(chan *StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		body, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequest("POST", c.baseURL+APIEndpoint, bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			errChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("error reading stream: %w", err)
				return
			}

			lineStr := strings.TrimSpace(string(line))
			if lineStr == "" {
				continue
			}

			if strings.HasPrefix(lineStr, "data: ") {
				data := strings.TrimPrefix(lineStr, "data: ")

				if data == "[DONE]" {
					return
				}

				var chunk StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					// Skip malformed JSON
					continue
				}

				chunkChan <- &chunk
			}
		}
	}()

	return struct {
		ChunkChan <-chan *StreamChunk
		ErrChan   <-chan error
	}{ChunkChan: chunkChan, ErrChan: errChan}, nil
}
