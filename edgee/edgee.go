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
	// DefaultMaxIterations is the default max iterations for the agentic loop
	DefaultMaxIterations = 10
)

// ToolHandler is a function that handles tool execution
type ToolHandler func(args map[string]any) (any, error)

// ExecutableTool represents a tool with an executable handler
type ExecutableTool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     ToolHandler
}

// NewTool creates a new ExecutableTool with a builder pattern
func NewTool(name, description string) *ExecutableTool {
	return &ExecutableTool{
		Name:        name,
		Description: description,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		},
	}
}

// AddParam adds a parameter to the tool
func (t *ExecutableTool) AddParam(name, paramType, description string, required bool) *ExecutableTool {
	props := t.Parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        paramType,
		"description": description,
	}

	if required {
		req := t.Parameters["required"].([]string)
		t.Parameters["required"] = append(req, name)
	}

	return t
}

// AddEnumParam adds an enum parameter to the tool
func (t *ExecutableTool) AddEnumParam(name string, enumValues []string, description string, required bool) *ExecutableTool {
	props := t.Parameters["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "string",
		"enum":        enumValues,
		"description": description,
	}

	if required {
		req := t.Parameters["required"].([]string)
		t.Parameters["required"] = append(req, name)
	}

	return t
}

// WithHandler sets the handler function for the tool
func (t *ExecutableTool) WithHandler(handler ToolHandler) *ExecutableTool {
	t.Handler = handler
	return t
}

// ToTool converts ExecutableTool to a Tool for API requests
func (t *ExecutableTool) ToTool() Tool {
	desc := t.Description
	return Tool{
		Type: "function",
		Function: FunctionDefinition{
			Name:        t.Name,
			Description: &desc,
			Parameters:  t.Parameters,
		},
	}
}

// SimpleInput represents input with executable tools for automatic tool execution
type SimpleInput struct {
	Text          string
	Tools         []*ExecutableTool
	MaxIterations int
}

// NewSimpleInput creates a new SimpleInput with tools
func NewSimpleInput(text string, tools ...*ExecutableTool) *SimpleInput {
	return &SimpleInput{
		Text:          text,
		Tools:         tools,
		MaxIterations: DefaultMaxIterations,
	}
}

// WithMaxIterations sets the maximum number of agentic loop iterations
func (s *SimpleInput) WithMaxIterations(max int) *SimpleInput {
	s.MaxIterations = max
	return s
}

// StreamEventType represents the type of stream event
type StreamEventType string

const (
	StreamEventChunk             StreamEventType = "chunk"
	StreamEventToolStart         StreamEventType = "tool_start"
	StreamEventToolResult        StreamEventType = "tool_result"
	StreamEventIterationComplete StreamEventType = "iteration_complete"
)

// StreamEvent represents an event during streaming with auto tool execution
type StreamEvent struct {
	Type      StreamEventType
	Chunk     *StreamChunk
	ToolCall  *ToolCall
	ToolName  string
	Result    any
	Iteration int
}

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
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// InputObject represents structured input for chat completion
type InputObject struct {
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice any       `json:"tool_choice,omitempty"` // string or object
}

// Request represents the request body for chat completions
type Request struct {
	Model      string    `json:"model"`
	Messages   []Message `json:"messages"`
	Stream     bool      `json:"stream,omitempty"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice any       `json:"tool_choice,omitempty"`
}

// StreamDelta represents a streaming chunk delta
type StreamDelta struct {
	Role      *string    `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	FinishReason *string  `json:"finish_reason,omitempty"`
}

// StreamChoice represents a choice in the streaming response
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        *StreamDelta `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SendResponse represents the response from a non-streaming request
type SendResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
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
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
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
func NewClient(config any) (*Client, error) {
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
// - Pass a *SimpleInput for automatic tool execution (agentic loop)
// - Pass an InputObject for manual tool handling
// - Pass a map[string]any with "messages", "tools", "tool_choice" keys
func (c *Client) Send(model string, input any) (SendResponse, error) {
	// Check if this is a SimpleInput for auto tool execution
	switch v := input.(type) {
	case *SimpleInput:
		return c.sendWithAutoTools(model, v)
	case SimpleInput:
		return c.sendWithAutoTools(model, &v)
	default:
		req, err := c.buildRequest(model, input, false)
		if err != nil {
			return SendResponse{}, err
		}
		return c.handleNonStreamingResponse(req)
	}
}

// sendWithAutoTools implements the agentic loop for automatic tool execution
func (c *Client) sendWithAutoTools(model string, input *SimpleInput) (SendResponse, error) {
	// Convert executable tools to API tools
	tools := make([]Tool, len(input.Tools))
	toolHandlers := make(map[string]ToolHandler)
	for i, t := range input.Tools {
		tools[i] = t.ToTool()
		toolHandlers[t.Name] = t.Handler
	}

	// Build initial messages
	messages := []Message{{Role: "user", Content: input.Text}}

	// Accumulate usage across iterations
	var totalUsage *Usage

	for iteration := 0; iteration < input.MaxIterations; iteration++ {
		// Build and send request
		req := &Request{
			Model:    model,
			Messages: messages,
			Tools:    tools,
			Stream:   false,
		}

		response, err := c.handleNonStreamingResponse(req)
		if err != nil {
			return response, err
		}

		// Accumulate usage
		if response.Usage != nil {
			if totalUsage == nil {
				totalUsage = &Usage{}
			}
			totalUsage.PromptTokens += response.Usage.PromptTokens
			totalUsage.CompletionTokens += response.Usage.CompletionTokens
			totalUsage.TotalTokens += response.Usage.TotalTokens
		}

		// Check if model requested tool calls
		toolCalls := response.ToolCalls()
		if len(toolCalls) == 0 {
			// No tool calls - return final response with accumulated usage
			response.Usage = totalUsage
			return response, nil
		}

		// Add assistant message with tool calls
		if response.MessageContent() != nil {
			messages = append(messages, *response.MessageContent())
		}

		// Execute each tool and add results
		for _, toolCall := range toolCalls {
			handler, ok := toolHandlers[toolCall.Function.Name]
			if !ok {
				// Unknown tool - add error result
				toolCallID := toolCall.ID
				messages = append(messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf(`{"error": "Unknown tool: %s"}`, toolCall.Function.Name),
					ToolCallID: &toolCallID,
				})
				continue
			}

			// Parse arguments
			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				toolCallID := toolCall.ID
				messages = append(messages, Message{
					Role:       "tool",
					Content:    fmt.Sprintf(`{"error": "Failed to parse arguments: %s"}`, err.Error()),
					ToolCallID: &toolCallID,
				})
				continue
			}

			// Execute handler
			result, err := handler(args)
			var resultStr string
			if err != nil {
				resultStr = fmt.Sprintf(`{"error": "Tool execution failed: %s"}`, err.Error())
			} else {
				resultBytes, _ := json.Marshal(result)
				resultStr = string(resultBytes)
			}

			toolCallID := toolCall.ID
			messages = append(messages, Message{
				Role:       "tool",
				Content:    resultStr,
				ToolCallID: &toolCallID,
			})
		}
	}

	return SendResponse{}, fmt.Errorf("max tool iterations (%d) reached", input.MaxIterations)
}

// ChatCompletion sends a non-streaming chat completion request (convenience method)
func (c *Client) ChatCompletion(model string, input any) (SendResponse, error) {
	return c.Send(model, input)
}

// Stream sends a streaming chat completion request with flexible input:
// - Pass a string for simple streaming
// - Pass a *SimpleInput for streaming with automatic tool execution
// - Pass an InputObject or map for manual control
func (c *Client) Stream(model string, input any) (<-chan *StreamEvent, <-chan error) {
	// Check if this is a SimpleInput for auto tool execution
	switch v := input.(type) {
	case *SimpleInput:
		return c.streamWithAutoTools(model, v)
	case SimpleInput:
		return c.streamWithAutoTools(model, &v)
	default:
		// Regular streaming - wrap chunks in StreamEvent
		return c.streamRegular(model, input)
	}
}

// streamRegular handles regular streaming without auto tools
func (c *Client) streamRegular(model string, input any) (<-chan *StreamEvent, <-chan error) {
	eventChan := make(chan *StreamEvent, 10)
	errChan := make(chan error, 1)

	req, err := c.buildRequest(model, input, true)
	if err != nil {
		errChan <- err
		close(errChan)
		close(eventChan)
		return eventChan, errChan
	}

	go func() {
		defer close(eventChan)
		defer close(errChan)

		chunkChan, chunkErrChan := c.doStreamRequest(req)

		for {
			select {
			case chunk, ok := <-chunkChan:
				if !ok {
					return
				}
				eventChan <- &StreamEvent{
					Type:  StreamEventChunk,
					Chunk: chunk,
				}
			case err := <-chunkErrChan:
				if err != nil {
					errChan <- err
				}
				return
			}
		}
	}()

	return eventChan, errChan
}

// streamWithAutoTools implements streaming with automatic tool execution
func (c *Client) streamWithAutoTools(model string, input *SimpleInput) (<-chan *StreamEvent, <-chan error) {
	eventChan := make(chan *StreamEvent, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		// Convert executable tools to API tools
		tools := make([]Tool, len(input.Tools))
		toolHandlers := make(map[string]ToolHandler)
		for i, t := range input.Tools {
			tools[i] = t.ToTool()
			toolHandlers[t.Name] = t.Handler
		}

		// Build initial messages
		messages := []Message{{Role: "user", Content: input.Text}}

		for iteration := 0; iteration < input.MaxIterations; iteration++ {
			// Build request
			req := &Request{
				Model:    model,
				Messages: messages,
				Tools:    tools,
				Stream:   true,
			}

			// Stream the response and collect tool calls
			var collectedToolCalls []ToolCall
			var assistantContent strings.Builder

			chunkChan, chunkErrChan := c.doStreamRequest(req)

		streamLoop:
			for {
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						break streamLoop
					}

					// Send chunk event
					eventChan <- &StreamEvent{
						Type:  StreamEventChunk,
						Chunk: chunk,
					}

					// Collect text content
					if text := chunk.Text(); text != "" {
						assistantContent.WriteString(text)
					}

					// Collect tool calls from delta
					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
						for _, tc := range chunk.Choices[0].Delta.ToolCalls {
							// Merge or add tool call
							found := false
							for i := range collectedToolCalls {
								if collectedToolCalls[i].ID == tc.ID || (collectedToolCalls[i].ID == "" && i < len(collectedToolCalls)) {
									// Merge arguments
									collectedToolCalls[i].Function.Arguments += tc.Function.Arguments
									if tc.Function.Name != "" {
										collectedToolCalls[i].Function.Name = tc.Function.Name
									}
									if tc.ID != "" {
										collectedToolCalls[i].ID = tc.ID
									}
									if tc.Type != "" {
										collectedToolCalls[i].Type = tc.Type
									}
									found = true
									break
								}
							}
							if !found {
								collectedToolCalls = append(collectedToolCalls, tc)
							}
						}
					}

				case err := <-chunkErrChan:
					if err != nil {
						errChan <- err
						return
					}
					break streamLoop
				}
			}

			// If no tool calls, we're done
			if len(collectedToolCalls) == 0 {
				return
			}

			// Add assistant message with tool calls
			messages = append(messages, Message{
				Role:      "assistant",
				Content:   assistantContent.String(),
				ToolCalls: collectedToolCalls,
			})

			// Execute each tool
			for _, toolCall := range collectedToolCalls {
				// Send tool start event
				eventChan <- &StreamEvent{
					Type:     StreamEventToolStart,
					ToolCall: &toolCall,
				}

				handler, ok := toolHandlers[toolCall.Function.Name]
				var resultStr string
				var result any

				if !ok {
					resultStr = fmt.Sprintf(`{"error": "Unknown tool: %s"}`, toolCall.Function.Name)
					result = map[string]any{"error": fmt.Sprintf("Unknown tool: %s", toolCall.Function.Name)}
				} else {
					// Parse arguments
					var args map[string]any
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						resultStr = fmt.Sprintf(`{"error": "Failed to parse arguments: %s"}`, err.Error())
						result = map[string]any{"error": err.Error()}
					} else {
						// Execute handler
						var err error
						result, err = handler(args)
						if err != nil {
							resultStr = fmt.Sprintf(`{"error": "Tool execution failed: %s"}`, err.Error())
							result = map[string]any{"error": err.Error()}
						} else {
							resultBytes, _ := json.Marshal(result)
							resultStr = string(resultBytes)
						}
					}
				}

				// Send tool result event
				eventChan <- &StreamEvent{
					Type:     StreamEventToolResult,
					ToolName: toolCall.Function.Name,
					Result:   result,
					ToolCall: &toolCall,
				}

				toolCallID := toolCall.ID
				messages = append(messages, Message{
					Role:       "tool",
					Content:    resultStr,
					ToolCallID: &toolCallID,
				})
			}

			// Send iteration complete event
			eventChan <- &StreamEvent{
				Type:      StreamEventIterationComplete,
				Iteration: iteration + 1,
			}
		}

		errChan <- fmt.Errorf("max tool iterations (%d) reached", input.MaxIterations)
	}()

	return eventChan, errChan
}

// doStreamRequest performs the actual streaming HTTP request
func (c *Client) doStreamRequest(req *Request) (<-chan *StreamChunk, <-chan error) {
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

	return chunkChan, errChan
}

func (c *Client) buildRequest(model string, input any, stream bool) (*Request, error) {
	req := &Request{
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
	case map[string]any:
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

func (c *Client) handleNonStreamingResponse(req *Request) (response SendResponse, err error) {
	body, err := json.Marshal(req)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return response, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return response, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return response, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("failed to decode response: %w", err)
	}

	return
}
