package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/payOSHQ/payos-lib-golang"
)

func main() {
	client, err := payos.NewPayOS(payos.NewPayOSOptions(nil))
	if err != nil {
		log.Fatalf("Failed to create PayOS client: %v", err)
	}

	// Create a payment link
	paymentData := payos.CreatePaymentLinkRequest{
		OrderCode:   time.Now().UnixMilli(),
		Amount:      2000,
		Description: "payment",
		ReturnUrl:   "https://your-domain.com/return",
		CancelUrl:   "https://your-domain.com/cancel",
		Items: []payos.PaymentLinkItem{
			{
				Name:     "Product 1",
				Quantity: 1,
				Price:    2000,
			},
		},
	}

	result, err := client.PaymentRequests.Create(context.TODO(), paymentData)
	if err != nil {
		log.Fatalf("Failed to create payment link: %v", err)
	}

	fmt.Printf("Payment Link Created:\n")
	fmt.Printf("  Checkout URL: %s\n", result.CheckoutUrl)
	fmt.Printf("  QR Code: %s\n", result.QrCode)
	fmt.Printf("  Order Code: %d\n", result.OrderCode)
	fmt.Printf("  Status: %s\n", result.Status)
}
