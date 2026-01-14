# Edgee Go SDK

Lightweight, type-safe Go SDK for the [Edgee AI Gateway](https://www.edgee.cloud).

[![Go Reference](https://pkg.go.dev/badge/github.com/edgee-cloud/go-sdk.svg)](https://pkg.go.dev/github.com/edgee-cloud/go-sdk)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Installation

```bash
go get github.com/edgee-cloud/go-sdk/edgee
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/edgee-cloud/go-sdk/edgee"
)

func main() {
    client, err := edgee.NewClient("your-api-key")
    if err != nil {
        log.Fatal(err)
    }

    response, err := client.Send("gpt-4o", "What is the capital of France?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Text())
    // "The capital of France is Paris."
}
```

## Send Method

The `Send()` method makes non-streaming chat completion requests:

```go
response, err := client.Send("gpt-4o", "Hello, world!")
if err != nil {
    log.Fatal(err)
}

// Access response
fmt.Println(response.Text())         // Text content
fmt.Println(response.FinishReason()) // Finish reason
fmt.Println(response.ToolCalls())    // Tool calls (if any)
```

## Stream Method

The `Stream()` method enables real-time streaming responses:

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
        
        if reason := chunk.FinishReason(); reason != "" {
            fmt.Printf("\nFinished: %s\n", reason)
        }
    case err := <-errChan:
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

## Features

- ✅ **Type-safe** - Strong typing with Go structs and interfaces
- ✅ **OpenAI-compatible** - Works with any model supported by Edgee
- ✅ **Streaming** - Real-time response streaming with channels
- ✅ **Tool calling** - Full support for function calling
- ✅ **Flexible input** - Accept strings, InputObject, or maps
- ✅ **Minimal dependencies** - Uses only standard library and essential packages

## Documentation

For complete documentation, examples, and API reference, visit:

**👉 [Official Go SDK Documentation](https://www.edgee.cloud/docs/sdk/go)**

The documentation includes:
- [Configuration guide](https://www.edgee.cloud/docs/sdk/go/configuration) - Multiple ways to configure the SDK
- [Send method](https://www.edgee.cloud/docs/sdk/go/send) - Complete guide to non-streaming requests
- [Stream method](https://www.edgee.cloud/docs/sdk/go/stream) - Streaming responses guide
- [Tools](https://www.edgee.cloud/docs/sdk/go/tools) - Function calling guide

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
