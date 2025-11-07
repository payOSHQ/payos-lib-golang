package payos

import (
	"context"

	"github.com/payOSHQ/payos-lib-golang/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/internal/crypto"
)

// ========================
// Webhook Types
// ========================

// ConfirmWebhookRequest represents the request to confirm a webhook URL
type ConfirmWebhookRequest struct {
	WebhookUrl string `json:"webhookUrl"`
}

// ConfirmWebhookResponse represents the response after confirming a webhook
type ConfirmWebhookResponse struct {
	WebhookUrl    string `json:"webhookUrl"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	Name          string `json:"name"`
	ShortName     string `json:"shortName"`
}

// Webhook represents a webhook payload
type Webhook struct {
	Code      string       `json:"code"`
	Desc      string       `json:"desc"`
	Success   *bool        `json:"success,omitempty"`
	Data      *WebhookData `json:"data"`
	Signature string       `json:"signature"`
}

// WebhookData represents the data within a webhook
type WebhookData struct {
	OrderCode              int64   `json:"orderCode"`
	Amount                 int     `json:"amount"`
	Description            string  `json:"description"`
	AccountNumber          string  `json:"accountNumber"`
	Reference              string  `json:"reference"`
	TransactionDateTime    string  `json:"transactionDateTime"`
	Currency               string  `json:"currency"`
	PaymentLinkId          string  `json:"paymentLinkId"`
	Code                   string  `json:"code"`
	Desc                   string  `json:"desc"`
	CounterAccountBankId   *string `json:"counterAccountBankId"`
	CounterAccountBankName *string `json:"counterAccountBankName"`
	CounterAccountName     *string `json:"counterAccountName"`
	CounterAccountNumber   *string `json:"counterAccountNumber"`
	VirtualAccountName     *string `json:"virtualAccountName"`
	VirtualAccountNumber   *string `json:"virtualAccountNumber"`
}

// ========================
// Backward Compatibility Aliases (Deprecated)
// ========================

// WebhookType is deprecated, use Webhook instead
//
// Deprecated: Use Webhook
type WebhookType = Webhook

// WebhookDataType is deprecated, use WebhookData instead
//
// Deprecated: Use WebhookData
type WebhookDataType = WebhookData

// ConfirmWebhookRequestType is deprecated, use ConfirmWebhookRequest instead
//
// Deprecated: Use ConfirmWebhookRequest
type ConfirmWebhookRequestType = ConfirmWebhookRequest

// Webhooks handles webhook operations
type Webhooks struct {
	client *Client
}

// Confirm validates the webhook URL and updates it if successful
func (w *Webhooks) Confirm(ctx context.Context, webhookUrl string) (string, error) {
	if webhookUrl == "" {
		return "", apierror.NewPayOSError("invalid params")
	}

	body := map[string]interface{}{
		"webhookUrl": webhookUrl,
	}

	_, err := w.client.Post(ctx, "/confirm-webhook", body, nil, nil)
	if err != nil {
		return "", err
	}

	return webhookUrl, nil
}

// VerifyData verifies data received via webhook after payment
func (w *Webhooks) VerifyData(ctx context.Context, webhookBody interface{}) (interface{}, error) {
	// This is a utility function that doesn't require the HTTP client
	return verifyWebhookSignature(webhookBody, w.client.checksumKey)
}

// verifyWebhookSignature is a helper function to verify webhook signatures
func verifyWebhookSignature(webhookBody interface{}, checksumKey string) (interface{}, error) {
	// Use type assertion to get webhook data
	webhook, ok := webhookBody.(map[string]interface{})
	if !ok {
		return nil, apierror.NewPayOSError("Invalid webhook body format")
	}

	data, hasData := webhook["data"]
	signature, hasSig := webhook["signature"]

	if !hasData || data == nil {
		return nil, apierror.NewPayOSError("data invalid")
	}
	if !hasSig || signature == "" {
		return nil, apierror.NewPayOSError("signature invalid")
	}

	signData, err := crypto.CreateSignatureFromObj(data, checksumKey)
	if err != nil {
		return nil, apierror.NewPayOSError("internal server error")
	}

	if signData != signature.(string) {
		return nil, apierror.NewPayOSError("data not integrity")
	}

	return data, nil
}
