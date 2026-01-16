// Auto tool execution example
//
// This example shows how to:
// - Define tools using the NewTool builder
// - Let the SDK automatically execute tools when the model calls them
// - Get the final response after all tool calls are processed
//
// The SDK handles the agentic loop automatically: when the model requests
// a tool call, the SDK executes your handler and sends the result back
// to the model until a final response is generated.
//
// Run with: go run examples/tools_auto/main.go

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
	// The builder creates an ExecutableTool with:
	// - name: the function name the model will call
	// - description: helps the model understand when to use this tool
	// - parameters: defined using AddParam/AddEnumParam
	// - handler: function that executes when the tool is called
	weatherTool := edgee.NewTool("get_weather", "Get the current weather for a location").
		AddParam("location", "string", "The city name", true).
		WithHandler(func(args map[string]any) (any, error) {
			// This handler is called automatically when the model uses this tool
			location, _ := args["location"].(string)

			// In a real app, you would call an actual weather API here
			fmt.Printf("[Tool executed: get_weather for %s]\n", location)

			return map[string]any{
				"location":    location,
				"temperature": 22,
				"unit":        "celsius",
				"condition":   "sunny",
			}, nil
		})

	// Create a SimpleInput with your prompt and tools
	// The SDK will automatically handle tool execution
	input := edgee.NewSimpleInput("What's the weather in Paris?", weatherTool)

	fmt.Println("Sending request with auto tool execution...")
	fmt.Println()

	// Send the request - the SDK handles the agentic loop automatically
	response, err := client.Send("devstral2", input)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print the final response (after all tools have been executed)
	fmt.Printf("\nFinal response: %s\n", response.Text())
}
