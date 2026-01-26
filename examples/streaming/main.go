// Streaming example demonstrating real-time response processing
//
// This example shows how to:
// - Stream responses from a model in real-time
// - Process chunks as they arrive
// - Display text progressively (like a typing effect)
//
// Run with: go run examples/streaming/main.go

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

	fmt.Println("Asking the model to count from 1 to 10...")
	fmt.Println()

	// Start a streaming request
	eventChan, errChan := client.Stream("devstral2", "Count from 1 to 10")

	// Process each event as it arrives
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, we're done
				fmt.Println("\n\n[Stream finished]")
				return
			}

			// For regular streaming, we only get chunk events
			if event.Type == edgee.StreamEventChunk && event.Chunk != nil {
				// Print each text chunk as it arrives (no newline, for typing effect)
				if text := event.Chunk.Text(); text != "" {
					fmt.Print(text)
				}

				// Check if the stream is complete
				if reason := event.Chunk.FinishReason(); reason != "" {
					fmt.Printf("\n\n[Finish reason: %s]", reason)
				}
			}

		case err := <-errChan:
			if err != nil {
				log.Fatalf("Error during streaming: %v", err)
			}
			return
		}
	}
}
