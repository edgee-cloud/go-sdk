// Simple example demonstrating basic usage of the Edgee SDK
//
// This example shows how to:
// - Create an Edgee client with your API key
// - Send a simple text prompt to a model
// - Get the response text
//
// Run with: go run examples/simple/main.go

package main

import (
	"fmt"
	"log"

	"github.com/edgee-cloud/go-sdk/edgee"
)

func main() {
	// Create the Edgee client with your API key
	client, err := edgee.NewClient("your-api-key")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Send a simple text prompt to the model
	response, err := client.Send("devstral2", "What is the capital of France?")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print the response text
	fmt.Printf("Response: %s\n", response.Text())

	// You can also access metadata about the response
	fmt.Printf("Model used: %s\n", response.Model)

	if response.Usage != nil {
		fmt.Printf("Tokens: %d prompt + %d completion = %d total\n",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.TotalTokens,
		)
	}
}
