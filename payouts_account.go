package payos

import (
	"context"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/apijson"
)

// ========================
// Payout Account Types
// ========================

// PayoutAccountInfo represents payout account information
type PayoutAccountInfo struct {
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
	Currency      string `json:"currency"`
	Balance       string `json:"balance"`
}

// PayoutsAccount handles payout account operations
type PayoutsAccount struct {
	client *Client
}

// newPayoutsAccount creates a new PayoutsAccount instance
func newPayoutsAccount(client *Client) *PayoutsAccount {
	return &PayoutsAccount{
		client: client,
	}
}

// Balance retrieves the current payout account balance
func (pa *PayoutsAccount) Balance(ctx context.Context) (*PayoutAccountInfo, error) {
	result, err := pa.client.Get(ctx, "/v1/payouts-account/balance", nil, nil, &SignatureOpts{Response: "header"})
	if err != nil {
		return nil, err
	}

	var response PayoutAccountInfo
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}
