package payos

import (
	"context"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/apijson"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/crypto"
)

// Batch handles batch payout operations
type Batch struct {
	client *Client
}

// newBatch creates a new Batch instance
func newBatch(client *Client) *Batch {
	return &Batch{
		client: client,
	}
}

// Create creates a batch payout
func (b *Batch) Create(ctx context.Context, payoutData PayoutBatchRequest, idempotencyKey *string) (*Payout, error) {
	// Generate idempotency key if not provided
	key := ""
	if idempotencyKey != nil {
		key = *idempotencyKey
	} else {
		key = crypto.GenerateUUID()
	}

	result, err := b.client.Post(ctx, "/v1/payouts/batch", payoutData, &SignatureOpts{
		Request:  "header",
		Response: "header",
	}, map[string]string{"x-idempotency-key": key})
	if err != nil {
		return nil, err
	}

	var response Payout
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}
