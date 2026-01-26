// Example usage of Edgee Gateway SDK
//
// This example demonstrates various ways to use the SDK.
// For more focused examples, see the examples/ directory.

package main

import (
	"fmt"
	"log"

	"github.com/edgee-cloud/go-sdk/edgee"
)

func main() {
	// Create client with API key
	client, err := edgee.NewClient("your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Test 1: Simple string input
	fmt.Println("Test 1: Simple string input")
	response1, err := client.Send("devstral2", "What is the capital of France?")
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Content: %s\n", response1.Text())
		if response1.Usage != nil {
			fmt.Printf("Usage: %+v\n", response1.Usage)
		}
	}
	fmt.Println()

	// Test 2: Full input object with messages
	fmt.Println("Test 2: Full input object with messages")
	response2, err := client.Send("devstral2", map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Say hello!"},
		},
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Content: %s\n", response2.Text())
	}
	fmt.Println()

	// Test 3: Auto tool execution
	fmt.Println("Test 3: Auto tool execution")
	weatherTool := edgee.NewTool("get_weather", "Get the current weather for a location").
		AddParam("location", "string", "City name", true).
		WithHandler(func(args map[string]any) (any, error) {
			location, _ := args["location"].(string)
			fmt.Printf("  [Tool called: get_weather(%s)]\n", location)
			return map[string]any{
				"location":    location,
				"temperature": 22,
				"condition":   "sunny",
			}, nil
		})

	input := edgee.NewSimpleInput("What's the weather in Paris?", weatherTool)
	response3, err := client.Send("devstral2", input)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Content: %s\n", response3.Text())
	}
	fmt.Println()

	// Test 4: Streaming
	fmt.Println("Test 4: Streaming")
	eventChan, errChan := client.Stream("devstral2", "What is Go?")
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				fmt.Println()
				return
			}
			if event.Type == edgee.StreamEventChunk && event.Chunk != nil {
				if text := event.Chunk.Text(); text != "" {
					fmt.Print(text)
				}
			}
		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v\n", err)
			}
			return
		}
	}
}
