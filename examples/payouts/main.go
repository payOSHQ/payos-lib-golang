package main

import (
	"context"
	"fmt"
	"log"
	"os"

	payos "github.com/payOSHQ/payos-lib-golang"
)

func main() {
	// Create PayOS client
	client, err := payos.NewPayOS(&payos.PayOSOptions{
		ClientId:    os.Getenv("PAYOS_PAYOUT_CLIENT_ID"),
		ApiKey:      os.Getenv("PAYOS_PAYOUT_API_KEY"),
		ChecksumKey: os.Getenv("PAYOS_PAYOUT_CHECKSUM_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.TODO()

	payoutReq := payos.PayoutRequest{
		ReferenceId:     "PAYOUT-001",
		Amount:          50000,
		Description:     "Test payout",
		ToBin:           "970422",
		ToAccountNumber: "0123456789",
		Category:        []string{"test"},
	}

	payout, err := client.Payouts.Create(ctx, payoutReq, nil)
	if err != nil {
		log.Printf("Error creating payout: %v\n", err)
	} else {
		fmt.Printf("Payout created: %s\n", payout.Id)
		fmt.Printf("Approval state: %s\n", payout.ApprovalState)
	}

	limit := 10
	offset := 0
	approvalState := payos.PayoutApprovalStateCompleted
	listParams := &payos.GetPayoutListParams{
		Limit:         &limit,
		Offset:        &offset,
		ApprovalState: &approvalState,
	}
	payoutList, err := client.Payouts.List(ctx, listParams)
	if err != nil {
		log.Printf("Error listing payouts: %v\n", err)
	} else {
		fmt.Printf("Total payouts: %d\n", payoutList.Pagination.Total)
		fmt.Printf("Found %d payouts\n", len(payoutList.Data))
	}

	balance, err := client.PayoutsAccount.Balance(ctx)
	if err != nil {
		log.Printf("Error getting balance: %v\n", err)
	} else {
		fmt.Printf("Account: %s\n", balance.AccountNumber)
		fmt.Printf("Balance: %s %s\n", balance.Balance, balance.Currency)
	}
}
