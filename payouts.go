package payos

import (
	"context"
	"fmt"
	"strings"

	"github.com/payOSHQ/payos-lib-golang/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/internal/apijson"
	"github.com/payOSHQ/payos-lib-golang/internal/crypto"
	"github.com/payOSHQ/payos-lib-golang/internal/pagination"
)

// ========================
// Payout Types
// ========================

// PayoutTransactionState represents the state of a payout transaction
type PayoutTransactionState string

const (
	PayoutTransactionStateReceived   PayoutTransactionState = "RECEIVED"
	PayoutTransactionStateProcessing PayoutTransactionState = "PROCESSING"
	PayoutTransactionStateCancelled  PayoutTransactionState = "CANCELLED"
	PayoutTransactionStateSucceeded  PayoutTransactionState = "SUCCEEDED"
	PayoutTransactionStateOnHold     PayoutTransactionState = "ON_HOLD"
	PayoutTransactionStateReversed   PayoutTransactionState = "REVERSED"
	PayoutTransactionStateFailed     PayoutTransactionState = "FAILED"
)

// PayoutApprovalState represents the approval state of a payout
type PayoutApprovalState string

const (
	PayoutApprovalStateDrafting         PayoutApprovalState = "DRAFTING"
	PayoutApprovalStateSubmitted        PayoutApprovalState = "SUBMITTED"
	PayoutApprovalStateApproved         PayoutApprovalState = "APPROVED"
	PayoutApprovalStateRejected         PayoutApprovalState = "REJECTED"
	PayoutApprovalStateCancelled        PayoutApprovalState = "CANCELLED"
	PayoutApprovalStateScheduled        PayoutApprovalState = "SCHEDULED"
	PayoutApprovalStateProcessing       PayoutApprovalState = "PROCESSING"
	PayoutApprovalStateFailed           PayoutApprovalState = "FAILED"
	PayoutApprovalStatePartialCompleted PayoutApprovalState = "PARTIAL_COMPLETED"
	PayoutApprovalStateCompleted        PayoutApprovalState = "COMPLETED"
)

// PayoutRequest represents a request to create a payout
type PayoutRequest struct {
	ReferenceId     string   `json:"referenceId"`
	Amount          int      `json:"amount"`
	Description     string   `json:"description"`
	ToBin           string   `json:"toBin"`
	ToAccountNumber string   `json:"toAccountNumber"`
	Category        []string `json:"category,omitempty"`
}

// PayoutTransaction represents a single payout transaction
type PayoutTransaction struct {
	Id                  string                 `json:"id"`
	ReferenceId         string                 `json:"referenceId"`
	Amount              int                    `json:"amount"`
	Description         string                 `json:"description"`
	ToBin               string                 `json:"toBin"`
	ToAccountNumber     string                 `json:"toAccountNumber"`
	ToAccountName       *string                `json:"toAccountName"`
	Reference           *string                `json:"reference"`
	TransactionDatetime *string                `json:"transactionDatetime"`
	ErrorMessage        *string                `json:"errorMessage"`
	ErrorCode           *string                `json:"errorCode"`
	State               PayoutTransactionState `json:"state"`
}

// Payout represents a payout with its transactions
type Payout struct {
	Id            string              `json:"id"`
	ReferenceId   string              `json:"referenceId"`
	Transactions  []PayoutTransaction `json:"transactions"`
	Category      []string            `json:"category"`
	ApprovalState PayoutApprovalState `json:"approvalState"`
	CreatedAt     string              `json:"createdAt"`
}

// EstimateCredit represents the estimated credit for a payout
type EstimateCredit struct {
	EstimateCredit int `json:"estimateCredit"`
}

// PayoutListResponse represents a paginated list of payouts
type PayoutListResponse struct {
	Pagination pagination.Pagination `json:"pagination"`
	Payouts    []Payout              `json:"payouts"`
}

// PayoutBatchItem represents a single item in a batch payout
type PayoutBatchItem struct {
	ReferenceId     string `json:"referenceId"`
	Amount          int    `json:"amount"`
	Description     string `json:"description"`
	ToBin           string `json:"toBin"`
	ToAccountNumber string `json:"toAccountNumber"`
}

// PayoutBatchRequest represents a request to create a batch payout
type PayoutBatchRequest struct {
	ReferenceId         string            `json:"referenceId"`
	ValidateDestination *bool             `json:"validateDestination,omitempty"`
	Category            []string          `json:"category"`
	Payouts             []PayoutBatchItem `json:"payouts"`
}

// Payouts handles payout operations
type Payouts struct {
	client *Client
	Batch  *Batch
}

// GetPayoutListParams represents parameters for listing payouts
type GetPayoutListParams struct {
	ReferenceId   *string              `json:"referenceId,omitempty"`
	ApprovalState *PayoutApprovalState `json:"approvalState,omitempty"`
	Category      []string             `json:"category,omitempty"`
	FromDate      *string              `json:"fromDate,omitempty"` // ISO date string
	ToDate        *string              `json:"toDate,omitempty"`   // ISO date string
	Limit         *int                 `json:"limit,omitempty"`
	Offset        *int                 `json:"offset,omitempty"`
}

// newPayouts creates a new Payouts instance
func newPayouts(client *Client) *Payouts {
	p := &Payouts{
		client: client,
	}
	p.Batch = newBatch(client)
	return p
}

// Create creates a new payout
func (p *Payouts) Create(ctx context.Context, payoutData PayoutRequest, idempotencyKey *string) (*Payout, error) {
	// Generate idempotency key if not provided
	key := ""
	if idempotencyKey != nil {
		key = *idempotencyKey
	} else {
		key = crypto.GenerateUUID()
	}

	// Add idempotency key to request options through Request method
	result, err := p.client.Post(ctx, "/v1/payouts/", payoutData, &SignatureOpts{
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

// Get retrieves detailed information about a specific payout
func (p *Payouts) Get(ctx context.Context, payoutId string) (*Payout, error) {
	if payoutId == "" {
		return nil, apierror.NewPayOSError("invalid params")
	}

	path := fmt.Sprintf("/v1/payouts/%s", payoutId)
	result, err := p.client.Get(ctx, path, nil, nil, &SignatureOpts{Response: "header"})
	if err != nil {
		return nil, err
	}

	var response Payout
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}

// EstimateCredit estimates credit required for one or multiple payouts
func (p *Payouts) EstimateCredit(ctx context.Context, payoutData interface{}) (*EstimateCredit, error) {
	result, err := p.client.Post(ctx, "/v1/payouts/estimate-credit", payoutData, &SignatureOpts{Request: "header"}, nil)
	if err != nil {
		return nil, err
	}

	var response EstimateCredit
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	return &response, nil
}

// List retrieves a paginated list of payouts filtered by the given criteria
// Returns a Page object that supports manual pagination with GetNextPage()
func (p *Payouts) List(ctx context.Context, params *GetPayoutListParams) (*pagination.Page[Payout], error) {
	if params == nil {
		params = &GetPayoutListParams{
			Limit:  intPtr(10),
			Offset: intPtr(0),
		}
	}

	page, err := p.fetchPayoutPage(ctx, params)
	if err != nil {
		return nil, err
	}

	return page, nil
}

// ListAutoPaging returns an iterator that automatically fetches all pages
func (p *Payouts) ListAutoPaging(ctx context.Context, params *GetPayoutListParams) *pagination.PageIterator[Payout] {
	if params == nil {
		params = &GetPayoutListParams{
			Limit:  intPtr(20),
			Offset: intPtr(0),
		}
	}

	fetcher := func(ctx context.Context, params interface{}) (*pagination.Page[Payout], error) {
		return p.fetchPayoutPage(ctx, params.(*GetPayoutListParams))
	}

	return pagination.NewPageIterator(ctx, params, fetcher)
}

// fetchPayoutPage is the internal method to fetch a single page of payouts
func (p *Payouts) fetchPayoutPage(ctx context.Context, params *GetPayoutListParams) (*pagination.Page[Payout], error) {
	// Build query parameters
	query := make(map[string]interface{})
	if params.ReferenceId != nil {
		query["referenceId"] = *params.ReferenceId
	}
	if params.ApprovalState != nil {
		query["approvalState"] = string(*params.ApprovalState)
	}
	if len(params.Category) > 0 {
		query["category"] = strings.Join(params.Category, ",")
	}
	if params.FromDate != nil {
		query["fromDate"] = *params.FromDate
	}
	if params.ToDate != nil {
		query["toDate"] = *params.ToDate
	}
	if params.Limit != nil {
		query["limit"] = *params.Limit
	}
	if params.Offset != nil {
		query["offset"] = *params.Offset
	}

	result, err := p.client.Get(ctx, "/v1/payouts", query, nil, &SignatureOpts{Response: "header"})
	if err != nil {
		return nil, err
	}

	var response PayoutListResponse
	if err := apijson.ConvertInterface(result, &response); err != nil {
		return nil, apierror.NewPayOSError("failed to parse response")
	}

	// Create Page object
	fetcher := func(ctx context.Context, params interface{}) (*pagination.Page[Payout], error) {
		return p.fetchPayoutPage(ctx, params.(*GetPayoutListParams))
	}

	page := &pagination.Page[Payout]{
		Data:       response.Payouts,
		Pagination: response.Pagination,
		Ctx:        ctx,
		Params:     params,
		Fetcher:    fetcher,
	}

	return page, nil
}

// Helper functions
func intPtr(i int) *int {
	return &i
}
