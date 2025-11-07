package payos

import (
	"context"
	"fmt"

	"github.com/payOSHQ/payos-lib-golang/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/internal/apijson"
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
	var idStr string
	switch v := id.(type) {
	case string:
		idStr = v
	case int:
		idStr = fmt.Sprintf("%d", v)
	case int64:
		idStr = fmt.Sprintf("%d", v)
	default:
		return nil, apierror.NewPayOSError("id must be string or number")
	}

	if idStr == "" {
		return nil, apierror.NewPayOSError("invalid params")
	}

	path := fmt.Sprintf("/v2/payment-requests/%s/invoices", id)
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
	var idStr string
	switch v := id.(type) {
	case string:
		idStr = v
	case int:
		idStr = fmt.Sprintf("%d", v)
	case int64:
		idStr = fmt.Sprintf("%d", v)
	default:
		return nil, apierror.NewPayOSError("id must be string or number")
	}

	if invoiceId == "" || idStr == "" {
		return nil, apierror.NewPayOSError("invalid params")
	}

	path := fmt.Sprintf("/v2/payment-requests/%s/invoices/%s/download", idStr, invoiceId)
	return inv.client.DownloadFile(ctx, path)
}
