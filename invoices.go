package payos

import (
	"context"
	"fmt"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/apijson"
)

// Invoices handles invoice operations for payment links
type Invoices struct {
	client *Client
}

// NewInvoices creates a new Invoices instance
func newInvoices(client *Client) *Invoices {
	return &Invoices{
		client: client,
	}
}

// Get retrieves invoices of a payment link by payment link ID or order code
func (inv *Invoices) Get(ctx context.Context, id interface{}) (*InvoicesInfo, error) {
	idStr, err := encodePathSegment("id", id)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v2/payment-requests/%s/invoices", idStr)
	result, err := inv.client.Get(ctx, path, nil, nil, &SignatureOpts{Response: "body"})
	if err != nil {
		return nil, err
	}

	var response InvoicesInfo
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}

// Download downloads an invoice in PDF format by invoice ID and payment link ID or order code
func (inv *Invoices) Download(ctx context.Context, invoiceId string, id interface{}) (*FileDownloadResponse, error) {
	idStr, err := encodePathSegment("id", id)
	if err != nil {
		return nil, err
	}

	invoiceStr, err := encodePathSegment("invoiceId", invoiceId)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v2/payment-requests/%s/invoices/%s/download", idStr, invoiceStr)
	return inv.client.DownloadFile(ctx, path)
}
