# payOS Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/payOSHQ/payos-lib-golang.svg)](https://pkg.go.dev/github.com/payOSHQ/payos-lib-golang)
[![Go Version](https://img.shields.io/github/go-mod/go-version/payOSHQ/payos-lib-golang)](https://golang.org/)

The payOS Go library provides convenient access to the payOS Merchant API from applications written in Go.

To learn how to use payOS Merchant API, checkout our [API Reference](https://payos.vn/docs/api) and [Documentation](https://payos.vn/docs). We also have some examples in [Examples](./examples/).

## Requirements

Go 1.21 or higher.

## Installation

```bash
go get github.com/payOSHQ/payos-lib-golang
```

> [!IMPORTANT]
> If update from v1, check [Migration guide](./MIGRATION.md) for detail migration.

## Usage

### Basic usage

First you need initialize the client to interacting with payOS Merchant API.

```go
import (
    "github.com/payOSHQ/payos-lib-golang"
)

client, err := payos.NewPayOS(&payos.PayOSOptions{
    ClientId:    "your-client-id",
    ApiKey:      "your-api-key",
    ChecksumKey: "your-checksum-key",
    // ... other options
})
if err != nil {
    log.Fatal(err)
}
```

Then you can interact with payOS Merchant API, example create a payment link using `PaymentRequests.Create()`.

```go
paymentLink, err := client.PaymentRequests.Create(context.Background(), payos.CreatePaymentLinkRequest{
    OrderCode:   123,
    Amount:      2000,
    Description: "payment",
    ReturnUrl:   "https://your-url.com",
    CancelUrl:   "https://your-url.com",
})
if err != nil {
    log.Fatal(err)
}
```

### Webhook verification

You can register an endpoint to receive the payment webhook.

```go
confirmResult, err := client.Webhooks.Confirm(context.Background(), "https://your-url.com/payos-webhook")
if err != nil {
    log.Fatal(err)
}
```

Then using `Webhooks.VerifyData()` to verify and receive webhook data.

```go
webhookBody := map[string]interface{}{
    "code": "00",
    "desc": "success",
    "success": true,
    "data": map[string]interface{}{
        "orderCode": 123,
        "amount": 3000,
        "description": "VQRIO123",
        "accountNumber": "12345678",
        "reference": "TF230204212323",
        "transactionDateTime": "2023-02-04 18:25:00",
        "currency": "VND",
        "paymentLinkId": "124c33293c43417ab7879e14c8d9eb18",
        "code": "00",
        "desc": "Thành công",
        "counterAccountBankId": "",
        "counterAccountBankName": "",
        "counterAccountName": "",
        "counterAccountNumber": "",
        "virtualAccountName": "",
        "virtualAccountNumber": "",
    },
    "signature": "8d8640d802576397a1ce45ebda7f835055768ac7ad2e0bfb77f9b8f12cca4c7f",
}

webhookData, err := client.Webhooks.VerifyData(webhookBody, client.ChecksumKey)
if err != nil {
    log.Fatal(err)
}
```

For more information about webhooks, see [the API doc](https://payos.vn/docs/api/#tag/payment-webhook/operation/payment-webhook).

### Handling errors

When the API return a non-success status code (i.e, 4xx or 5xx response) or non-success code data (any code except '00'), an error will be returned:

```go
_, err := client.PaymentRequests.Get(context.Background(), "not-found-order-code")
if err != nil {
    var apiErr *apierror.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode)
        fmt.Println(apiErr.Code)
        fmt.Println(apiErr.Message)
        fmt.Println(apiErr.Headers)
    } else {
        log.Fatal(err)
    }
}
```

### Auto pagination

List method in the payOS Merchant API are paginated. You can use the iterator to automatically fetch all pages:

```go
// Auto pagination
limit := 3
iterator := client.Payouts.ListAutoPaging(context.Background(), &payos.GetPayoutListParams{
    Limit: &limit,
})

for iterator.Next() {
    payout := iterator.Current()
    fmt.Printf("Payout ID: %s\n", payout.Id)
}

if err := iterator.Err(); err != nil {
    log.Fatal(err)
}
```

Or you can request single page at a time:

```go
// Manual pagination
limit := 3
page, err := client.Payouts.List(context.Background(), &payos.GetPayoutListParams{
    Limit: &limit,
})
if err != nil {
    log.Fatal(err)
}

for _, payout := range page.Data {
    fmt.Printf("Payout ID: %s\n", payout.Id)
}

for page.HasNextPage() {
    page, err = page.GetNextPage()
    if err != nil {
        log.Fatal(err)
    }
    for _, payout := range page.Data {
        fmt.Printf("Payout ID: %s\n", payout.Id)
    }
}
```

### Advanced usage

#### Custom configuration

You can customize the payOS client with various options:

```go
client, err := payos.NewPayOS(&payos.PayOSOptions{
    ClientId:    "your-client-id",
    ApiKey:      "your-api-key",
    ChecksumKey: "your-checksum-key",
    PartnerCode: "your-partner-code", // Optional partner code
    BaseURL:     "https://api-merchant.payos.vn", // Custom base URL
    Timeout:     30 * time.Second, // Request timeout (default: 60s)
    MaxRetries:  3, // Maximum retry attempts (default: 2)
    HTTPClient:  &http.Client{}, // Custom HTTP client
    DebugLogger: log.New(os.Stderr, "[payOS] ", log.LstdFlags), // Enable debug logging
})
```

#### Request-level options with context

You can use context for request cancellation and timeout:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

paymentLink, err := client.PaymentRequests.Create(ctx, payos.CreatePaymentLinkRequest{
    OrderCode:   123,
    Amount:      2000,
    Description: "payment",
    ReturnUrl:   "https://your-url.com",
    CancelUrl:   "https://your-url.com",
})

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(5 * time.Second)
    cancel() // Cancel after 5 seconds
}()

paymentLink, err := client.PaymentRequests.Create(ctx, paymentData)
```

#### Middleware support

You can add custom middleware to intercept and modify HTTP requests:

```go
// Custom logging middleware
loggingMiddleware := func(next payos.RequestHandler) payos.RequestHandler {
    return func(ctx context.Context, req *http.Request) (*http.Response, error) {
        start := time.Now()
        log.Printf("Request: %s %s", req.Method, req.URL.Path)

        resp, err := next(ctx, req)

        duration := time.Since(start)
        if err != nil {
            log.Printf("Request failed after %v: %v", duration, err)
        } else {
            log.Printf("Request completed in %v: %d", duration, resp.StatusCode)
        }

        return resp, err
    }
}

client, err := payos.NewPayOS(&payos.PayOSOptions{
    ClientId:    "your-client-id",
    ApiKey:      "your-api-key",
    ChecksumKey: "your-checksum-key",
    Middlewares: []payos.Middleware{loggingMiddleware},
})
```

#### Logging and debugging

Enable debug logging to see detailed request and response information:

```go
import (
    "log"
    "os"
)

client, err := payos.NewPayOS(&payos.PayOSOptions{
    ClientId:    "your-client-id",
    ApiKey:      "your-api-key",
    ChecksumKey: "your-checksum-key",
    DebugLogger: log.New(os.Stderr, "[payOS Debug] ", log.LstdFlags),
})
```

#### Direct API access

For advanced use cases, you can make direct API calls with signature options:

```go
// GET request with response signature verification
result, err := client.Client.Get(
    context.Background(),
    "/v2/payment-requests",
    nil, // query parameters
    nil, // headers
    &payos.SignatureOpts{Response: "body"}, // verify response signature from body
)

// POST request with request and response signatures
result, err := client.Client.Post(
    context.Background(),
    "/v2/payment-requests",
    requestData,
    &payos.SignatureOpts{
        Request:  "create-payment-link", // sign request
        Response: "body",                 // verify response signature from body
    },
    nil, // headers
)

// POST request with header signature
result, err := client.Client.Post(
    context.Background(),
    "/v1/payouts",
    payoutData,
    &payos.SignatureOpts{
        Request:  "header", // sign request in header
        Response: "body",   // verify response signature from body
    },
    map[string]string{"x-idempotency-key": "unique-key"},
)

// With custom options using RequestOptions
result, err := client.Client.Request(context.Background(), &payos.RequestOptions{
    Method: "POST",
    Path:   "/v2/payment-requests",
    Body:   requestData,
    SignatureOpts: &payos.SignatureOpts{
        Request:  "create-payment-link",
        Response: "body",
    },
})
```

The `SignatureOpts` struct has two fields:

- `Request`: Signature type for request - can be `"create-payment-link"`, `"body"`, or `"header"`
- `Response`: Signature type for response verification - can be `"body"` or `"header"`

#### Signature

The signature can be manually created using crypto functions:

```go
import (
    "github.com/payOSHQ/payos-lib-golang/internal/crypto"
)

// For create-payment-link signature
signature, err := crypto.CreateSignatureOfPaymentRequest(data, checksumKey)

// For payment-requests and webhook signature
signature, err := crypto.CreateSignatureFromObj(data, checksumKey)

// For payouts signature (used in headers)
signature, err := crypto.CreateSignature(checksumKey, data, nil)
```

## Contributing

See [the contributing documentation](./CONTRIBUTING.md).
