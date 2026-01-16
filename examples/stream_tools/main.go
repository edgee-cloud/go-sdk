// Streaming with automatic tool execution example
//
// This example shows how to:
// - Combine streaming with automatic tool execution
// - Receive real-time events for chunks, tool calls, and results
// - Display the response progressively while tools are being executed
//
// This is useful when you want to show progress to the user while
// tools are being executed in the background.
//
// Run with: go run examples/stream_tools/main.go

package main

import (
	"fmt"
	"log"

	"github.com/edgee-cloud/go-sdk/edgee"
)

func main() {
	// Create the Edgee client
	client, err := edgee.NewClient("your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Define a tool using the NewTool builder
	weatherTool := edgee.NewTool("get_weather", "Get the current weather for a location").
		AddParam("location", "string", "The city name", true).
		WithHandler(func(args map[string]any) (any, error) {
			location, _ := args["location"].(string)

			// Simulate an API call
			return map[string]any{
				"location":    location,
				"temperature": 22,
				"unit":        "celsius",
				"condition":   "sunny",
			}, nil
		})

	fmt.Println("Streaming request with auto tool execution...")
	fmt.Println()

	// Create SimpleInput for auto tool execution
	input := edgee.NewSimpleInput("What's the weather in Paris?", weatherTool)

	// Start a streaming request with tools
	eventChan, errChan := client.Stream("devstral2", input)

	// Process events as they arrive
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, we're done
				fmt.Println("\n\nDone!")
				return
			}

			switch event.Type {
			case edgee.StreamEventChunk:
				// Text chunks from the model
				if event.Chunk != nil {
					if text := event.Chunk.Text(); text != "" {
						fmt.Print(text)
					}
				}

			case edgee.StreamEventToolStart:
				// A tool is about to be executed
				fmt.Printf("\n[Calling tool: %s]\n", event.ToolCall.Function.Name)

			case edgee.StreamEventToolResult:
				// A tool has finished executing
				fmt.Printf("[Tool result: %s returned %v]\n", event.ToolName, event.Result)

			case edgee.StreamEventIterationComplete:
				// An iteration of the agentic loop is complete
				fmt.Printf("[Iteration %d complete]\n", event.Iteration)
			}

		case err := <-errChan:
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
			return
		}
	}
}
