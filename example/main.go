package main

import (
	"fmt"
	"log"

	"github.com/edgee-cloud/go-sdk/edgee"
)

func main() {
	// Create client with API key from environment variable
	client, err := edgee.NewClient(nil)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Test 1: Simple string input
	fmt.Println("Test 1: Simple string input")
	response1, err := client.ChatCompletion("mistral/mistral-small-latest", "What is the capital of France?")
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
	response2, err := client.ChatCompletion("mistral/mistral-small-latest", map[string]interface{}{
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

	// Test 3: With tools
	fmt.Println("Test 3: With tools")
	response3, err := client.ChatCompletion("gpt-4o", map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "What is the weather in Paris?"},
		},
		"tools": []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_weather",
					"description": "Get the current weather for a location",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]string{
								"type":        "string",
								"description": "City name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
		"tool_choice": "auto",
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Content: %s\n", response3.Text())
		if toolCalls := response3.ToolCalls(); len(toolCalls) > 0 {
			fmt.Printf("Tool calls: %+v\n", toolCalls)
		}
	}
	fmt.Println()

	// Test 4: Streaming (simplest way - text only)
	fmt.Println("Test 4: Streaming (text only)")
	textChan, errChan := client.StreamText("mistral/mistral-small-latest", "Tell me a short story about a robot")
	for {
		select {
		case text, ok := <-textChan:
			if !ok {
				fmt.Println()
				goto nextTest
			}
			fmt.Print(text)
		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v\n", err)
				goto nextTest
			}
		}
	}

nextTest:
	fmt.Println()

	// Test 5: Streaming with more control
	fmt.Println("Test 5: Streaming with more control")
	chunkChan, errChan2 := client.Stream("mistral/mistral-small-latest", "What is Go?")
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				fmt.Println()
				return
			}
			if text := chunk.Text(); text != "" {
				fmt.Print(text)
			}
		case err := <-errChan2:
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
	}
}
