# Edgee Gateway SDK for Go

Lightweight Go SDK for Edgee AI Gateway.

## Installation

```bash
go get github.com/edgee-cloud/go-sdk/edgee
```

## Usage

```go
import "github.com/edgee-cloud/go-sdk/edgee"

// Create client (uses EDGEE_API_KEY environment variable)
client, err := edgee.NewClient(nil)
if err != nil {
    log.Fatal(err)
}

// Or create with explicit API key
client, err := edgee.NewClient("your-api-key")

// Or create with full config
client, err := edgee.NewClient(&edgee.Config{
    APIKey:  "your-api-key",
    BaseURL: "https://api.edgee.ai",
})
```

### Simple Input

```go
response, err := client.ChatCompletion("gpt-4o", "What is the capital of France?")
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Text())
```

### Full Input with Messages

```go
response, err := client.ChatCompletion("gpt-4o", map[string]interface{}{
    "messages": []map[string]string{
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Text())
```

### With Tools

```go
response, err := client.ChatCompletion("gpt-4o", map[string]interface{}{
    "messages": []map[string]string{
        {"role": "user", "content": "What's the weather in Paris?"},
    },
    "tools": []map[string]interface{}{
        {
            "type": "function",
            "function": map[string]interface{}{
                "name":        "get_weather",
                "description": "Get weather for a location",
                "parameters": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "location": map[string]string{"type": "string"},
                    },
                },
            },
        },
    },
    "tool_choice": "auto",
})

if toolCalls := response.ToolCalls(); len(toolCalls) > 0 {
    fmt.Printf("Tool calls: %+v\n", toolCalls)
}
```

### Streaming

#### Simple Text Streaming

The simplest way to stream text responses:

```go
textChan, errChan := client.StreamText("gpt-4o", "Tell me a story")

for {
    select {
    case text, ok := <-textChan:
        if !ok {
            return
        }
        fmt.Print(text)
    case err := <-errChan:
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

#### Streaming with More Control

Access chunk properties when you need more control:

```go
chunkChan, errChan := client.Stream("gpt-4o", "Tell me a story")

for {
    select {
    case chunk, ok := <-chunkChan:
        if !ok {
            return
        }
        if text := chunk.Text(); text != "" {
            fmt.Print(text)
        }
    case err := <-errChan:
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

#### Accessing Full Chunk Data

When you need complete access to the streaming response:

```go
chunkChan, errChan := client.Stream("gpt-4o", "Hello")

for {
    select {
    case chunk, ok := <-chunkChan:
        if !ok {
            return
        }
        if role := chunk.Role(); role != "" {
            fmt.Printf("Role: %s\n", role)
        }
        if text := chunk.Text(); text != "" {
            fmt.Print(text)
        }
        if reason := chunk.FinishReason(); reason != "" {
            fmt.Printf("\nFinish: %s\n", reason)
        }
    case err := <-errChan:
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

## Response Types

### SendResponse

```go
type SendResponse struct {
    ID      string
    Object  string
    Created int64
    Model   string
    Choices []ChatCompletionChoice
    Usage   *Usage
}

// Convenience methods for easy access
func (r *SendResponse) Text() string                   // Shortcut for Choices[0].Message.Content
func (r *SendResponse) MessageContent() *Message       // Shortcut for Choices[0].Message
func (r *SendResponse) FinishReason() string           // Shortcut for Choices[0].FinishReason
func (r *SendResponse) ToolCalls() []ToolCall          // Shortcut for Choices[0].Message.ToolCalls
```

### ChatCompletionChoice

```go
type ChatCompletionChoice struct {
    Index        int
    Message      *Message      // For non-streaming responses
    Delta        *ChatCompletionDelta  // For streaming responses
    FinishReason *string
}
```

### Message

```go
type Message struct {
    Role       string
    Content    string
    Name       *string
    ToolCalls  []ToolCall
    ToolCallID *string
}
```

### Usage

```go
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### Streaming Response

```go
type StreamChunk struct {
    ID      string
    Object  string
    Created int64
    Model   string
    Choices []ChatCompletionChoice
}

// Convenience methods for easy access
func (c *StreamChunk) Text() string           // Shortcut for Choices[0].Delta.Content
func (c *StreamChunk) Role() string           // Shortcut for Choices[0].Delta.Role
func (c *StreamChunk) FinishReason() string   // Shortcut for Choices[0].FinishReason
```

### ChatCompletionDelta

```go
type ChatCompletionDelta struct {
    Role      *string     // Only present in first chunk
    Content   *string
    ToolCalls []ToolCall
}
```

To learn more about this SDK, please refer to the [dedicated documentation](https://www.edgee.cloud/docs/sdk/go).

## License

Apache-2.0
