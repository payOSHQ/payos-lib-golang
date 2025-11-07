package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/payOSHQ/payos-lib-golang"
)

func main() {
	// Create a new PayOS client
	client, err := payos.NewPayOS(nil)
	if err != nil {
		log.Fatalf("Failed to create PayOS client: %v", err)
	}

	// Setup webhook handler
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse webhook body
		var webhookData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&webhookData); err != nil {
			log.Printf("Failed to decode webhook: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify webhook data
		verifiedData, err := client.Webhooks.VerifyData(context.Background(), webhookData)
		if err != nil {
			log.Printf("Failed to verify webhook: %v", err)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Process verified webhook data
		fmt.Printf("Webhook received: %+v\n", verifiedData)

		// Respond to webhook
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	fmt.Println("Webhook server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	// Confirm your webhook url https://your-tunnel.com/webhook
}
