package payos

import (
	"context"
	"fmt"

	"github.com/payOSHQ/payos-lib-golang/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/internal/apijson"
)

// ========================
// Payment Request Types
// ========================

// PaymentLinkStatus represents the status of a payment link
type PaymentLinkStatus string

const (
	PaymentLinkStatusPending    PaymentLinkStatus = "PENDING"
	PaymentLinkStatusCancelled  PaymentLinkStatus = "CANCELLED"
	PaymentLinkStatusUnderpaid  PaymentLinkStatus = "UNDERPAID"
	PaymentLinkStatusPaid       PaymentLinkStatus = "PAID"
	PaymentLinkStatusExpired    PaymentLinkStatus = "EXPIRED"
	PaymentLinkStatusProcessing PaymentLinkStatus = "PROCESSING"
	PaymentLinkStatusFailed     PaymentLinkStatus = "FAILED"
)

// TaxPercentage represents tax percentage values
type TaxPercentage int

const (
	TaxPercentageNegTwo TaxPercentage = -2
	TaxPercentageNegOne TaxPercentage = -1
	TaxPercentageZero   TaxPercentage = 0
	TaxPercentageFive   TaxPercentage = 5
	TaxPercentageTen    TaxPercentage = 10
)

// PaymentLinkItem represents an item in a payment link
type PaymentLinkItem struct {
	Name          string         `json:"name"`
	Quantity      int            `json:"quantity"`
	Price         int            `json:"price"`
	Unit          *string        `json:"unit,omitempty"`
	TaxPercentage *TaxPercentage `json:"taxPercentage,omitempty"`
}

// InvoiceRequest represents invoice configuration for a payment
type InvoiceRequest struct {
	BuyerNotGetInvoice *bool          `json:"buyerNotGetInvoice,omitempty"`
	TaxPercentage      *TaxPercentage `json:"taxPercentage,omitempty"`
}

// CreatePaymentLinkRequest represents the request to create a payment link
type CreatePaymentLinkRequest struct {
	OrderCode        int64             `json:"orderCode"`
	Amount           int               `json:"amount"`
	Description      string            `json:"description"`
	CancelUrl        string            `json:"cancelUrl"`
	ReturnUrl        string            `json:"returnUrl"`
	Signature        *string           `json:"signature,omitempty"`
	Items            []PaymentLinkItem `json:"items,omitempty"`
	BuyerName        *string           `json:"buyerName,omitempty"`
	BuyerCompanyName *string           `json:"buyerCompanyName,omitempty"`
	BuyerTaxCode     *string           `json:"buyerTaxCode,omitempty"`
	BuyerEmail       *string           `json:"buyerEmail,omitempty"`
	BuyerPhone       *string           `json:"buyerPhone,omitempty"`
	BuyerAddress     *string           `json:"buyerAddress,omitempty"`
	Invoice          *InvoiceRequest   `json:"invoice,omitempty"`
	ExpiredAt        *int              `json:"expiredAt,omitempty"`
}

// CreatePaymentLinkResponse represents the response when creating a payment link
type CreatePaymentLinkResponse struct {
	Bin           string            `json:"bin"`
	AccountNumber string            `json:"accountNumber"`
	AccountName   string            `json:"accountName"`
	Amount        int               `json:"amount"`
	Description   string            `json:"description"`
	OrderCode     int64             `json:"orderCode"`
	Currency      string            `json:"currency"`
	PaymentLinkId string            `json:"paymentLinkId"`
	Status        PaymentLinkStatus `json:"status"`
	ExpiredAt     *int              `json:"expiredAt,omitempty"`
	CheckoutUrl   string            `json:"checkoutUrl"`
	QrCode        string            `json:"qrCode"`
}

// CancelPaymentLinkRequest represents the request to cancel a payment link
type CancelPaymentLinkRequest struct {
	CancellationReason *string `json:"cancellationReason,omitempty"`
}

// Transaction represents a payment transaction
type Transaction struct {
	Reference              string  `json:"reference"`
	Amount                 int     `json:"amount"`
	AccountNumber          string  `json:"accountNumber"`
	Description            string  `json:"description"`
	TransactionDateTime    string  `json:"transactionDateTime"`
	VirtualAccountName     *string `json:"virtualAccountName"`
	VirtualAccountNumber   *string `json:"virtualAccountNumber"`
	CounterAccountBankId   *string `json:"counterAccountBankId"`
	CounterAccountBankName *string `json:"counterAccountBankName"`
	CounterAccountName     *string `json:"counterAccountName"`
	CounterAccountNumber   *string `json:"counterAccountNumber"`
}

// PaymentLink represents complete payment link information
type PaymentLink struct {
	Id                 string            `json:"id"`
	OrderCode          int64             `json:"orderCode"`
	Amount             int               `json:"amount"`
	AmountPaid         int               `json:"amountPaid"`
	AmountRemaining    int               `json:"amountRemaining"`
	Status             PaymentLinkStatus `json:"status"`
	CreatedAt          string            `json:"createdAt"`
	Transactions       []Transaction     `json:"transactions"`
	CancellationReason *string           `json:"cancellationReason"`
	CanceledAt         *string           `json:"canceledAt"`
}

// Invoice represents an invoice associated with a payment
type Invoice struct {
	InvoiceId       string  `json:"invoiceId"`
	InvoiceNumber   *string `json:"invoiceNumber"`
	IssuedTimestamp *int64  `json:"issuedTimestamp"`
	IssuedDatetime  *string `json:"issuedDatetime"`
	TransactionId   *string `json:"transactionId"`
	ReservationCode *string `json:"reservationCode"`
	CodeOfTax       *string `json:"codeOfTax"`
}

// InvoicesInfo represents a collection of invoices
type InvoicesInfo struct {
	Invoices []Invoice `json:"invoices"`
}

// ========================
// Backward Compatibility Aliases (Deprecated)
// ========================

// Item is deprecated, use PaymentLinkItem instead
//
// Deprecated: Use PaymentLinkItem
type Item = PaymentLinkItem

// CheckoutRequestType is deprecated, use CreatePaymentLinkRequest instead
//
// Deprecated: Use CreatePaymentLinkRequest
type CheckoutRequestType = CreatePaymentLinkRequest

// CheckoutResponseDataType is deprecated, use CreatePaymentLinkResponse instead
//
// Deprecated: Use CreatePaymentLinkResponse
type CheckoutResponseDataType = CreatePaymentLinkResponse

// CancelPaymentLinkRequestType is deprecated, use CancelPaymentLinkRequest instead
//
// Deprecated: Use CancelPaymentLinkRequest
type CancelPaymentLinkRequestType = CancelPaymentLinkRequest

// PaymentLinkDataType is deprecated, use PaymentLink instead
//
// Deprecated: Use PaymentLink
type PaymentLinkDataType = PaymentLink

// TransactionType is deprecated, use Transaction instead
//
// Deprecated: Use Transaction
type TransactionType = Transaction

// PaymentRequests handles payment request operations
type PaymentRequests struct {
	client   *Client
	Invoices *Invoices
}

// newPaymentRequests creates a new PaymentRequests instance
func newPaymentRequests(client *Client) *PaymentRequests {
	pr := &PaymentRequests{
		client: client,
	}
	pr.Invoices = newInvoices(client)
	return pr
}

// Create creates a new payment link
func (pr *PaymentRequests) Create(ctx context.Context, data CreatePaymentLinkRequest) (*CreatePaymentLinkResponse, error) {
	// Validate required fields
	if data.OrderCode == 0 || data.Amount == 0 || data.Description == "" || data.CancelUrl == "" || data.ReturnUrl == "" {
		return nil, apierror.NewPayOSError("OrderCode, Amount, ReturnUrl, CancelUrl, Description must not be undefined or null.")
	}

	// Validate orderCode range
	if data.OrderCode < -9007199254740991 || data.OrderCode > 9007199254740991 {
		return nil, apierror.NewPayOSError("order code out of range")
	}

	result, err := pr.client.Post(ctx, "/v2/payment-requests", data, &SignatureOpts{
		Request:  "create-payment-link",
		Response: "body",
	}, nil)
	if err != nil {
		return nil, err
	}

	// Convert result to CreatePaymentLinkResponse
	var response CreatePaymentLinkResponse
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}

// Get retrieves payment link information by payment link ID or order code
func (pr *PaymentRequests) Get(ctx context.Context, id interface{}) (*PaymentLink, error) {
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

	path := fmt.Sprintf("/v2/payment-requests/%s", idStr)
	result, err := pr.client.Get(ctx, path, nil, nil, &SignatureOpts{Response: "body"})
	if err != nil {
		return nil, err
	}

	var response PaymentLink
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}

// Cancel cancels a payment link by payment link ID or order code
func (pr *PaymentRequests) Cancel(ctx context.Context, id interface{}, cancellationReason *string) (*PaymentLink, error) {
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

	path := fmt.Sprintf("/v2/payment-requests/%s/cancel", idStr)
	var body *CancelPaymentLinkRequest
	if cancellationReason != nil {
		body = &CancelPaymentLinkRequest{
			CancellationReason: cancellationReason,
		}
	}

	result, err := pr.client.Post(ctx, path, body, &SignatureOpts{Response: "body"}, nil)
	if err != nil {
		return nil, err
	}

	var response PaymentLink
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}
