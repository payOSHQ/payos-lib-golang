package main

import (
	"context"
	"fmt"
	"log"
	"os"

	payos "github.com/payOSHQ/payos-lib-golang"
)

func main() {
	client, err := payos.NewPayOS(&payos.PayOSOptions{
		ClientId:    os.Getenv("PAYOS_PAYOUT_CLIENT_ID"),
		ApiKey:      os.Getenv("PAYOS_PAYOUT_API_KEY"),
		ChecksumKey: os.Getenv("PAYOS_PAYOUT_CHECKSUM_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create PayOS client: %v", err)
	}

	ctx := context.Background()

	limit := 20
	iter := client.Payouts.ListAutoPaging(ctx, &payos.GetPayoutListParams{
		Limit: &limit,
	})

	count := 0
	for iter.Next() {
		payout := iter.Current()
		count++
		fmt.Printf("%d. Payout ID: %s, Reference: %s, State: %s\n",
			count, payout.Id, payout.ReferenceId, payout.ApprovalState)
	}

	if err := iter.Err(); err != nil {
		log.Fatalf("Error during iteration: %v", err)
	}

	fmt.Printf("\nTotal payouts processed: %d\n", count)
}
