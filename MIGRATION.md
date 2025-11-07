# Migration guide

This guide outlines the changes and steps needed to migrate your codebase to the latest version of the PayOS Go SDK.

## Breaking changes

### Initialize client

The library now uses a structured options pattern with `PayOSOptions` for better configuration management.

```go
// before
import "github.com/payOSHQ/payos-lib-golang"

payos.Key(clientId, apiKey, checksumKey, partnerCode)

// after
import "github.com/payOSHQ/payos-lib-golang"

client, err := payos.NewPayOS(&payos.PayOSOptions{
    ClientId:    clientId,
    ApiKey:      apiKey,
    ChecksumKey: checksumKey,
    PartnerCode: partnerCode, // optional
})
```

### Methods name

All methods related to payment request now under `client.PaymentRequests`.

```go
// before
payos.CreatePaymentLink(paymentData)
payos.GetPaymentLinkInformation(orderCode)
payos.CancelPaymentLink(orderCode, cancellationReason)

// after
client.PaymentRequests.Create(context.Background(), paymentData)
client.PaymentRequests.Get(context.Background(), orderCode)
client.PaymentRequests.Cancel(context.Background(), orderCode, cancellationReason)
```

For webhook related methods, they now under `client.Webhooks`.

```go
// before
payos.ConfirmWebhook(webhookUrl)
payos.VerifyPaymentWebhookData(webhookBody)

// after
client.Webhooks.Confirm(context.Background(), webhookUrl)
client.Webhooks.VerifyData(context.Background(), webhookBody)
```

### Context support

All API methods now require a `context.Context` parameter for better control over request lifecycle.

```go
// before
result, err := payos.CreatePaymentLink(paymentData)

// after
result, err := client.PaymentRequests.Create(context.Background(), paymentData)

// with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
result, err := client.PaymentRequests.Create(ctx, paymentData)
```

### Types

Some types have been renamed for better clarity and consistency.

```go
// before
var paymentData payos.CheckoutRequestType
var result *payos.CheckoutResponseDataType
result, err := payos.CreatePaymentLink(paymentData)

// after
var paymentData payos.CreatePaymentLinkRequest
var result *payos.CreatePaymentLinkResponse
result, err := client.PaymentRequests.Create(context.Background(), paymentData)
```

```go
// before
var paymentLinkInfo *payos.PaymentLinkDataType
paymentLinkInfo, err := payos.GetPaymentLinkInformation(orderCode)

var paymentLinkCancelled *payos.PaymentLinkDataType
paymentLinkCancelled, err := payos.CancelPaymentLink(orderCode, cancellationReason)

// after
var paymentLinkInfo *payos.PaymentLink
paymentLinkInfo, err := client.PaymentRequests.Get(context.Background(), orderCode)

var paymentLinkCancelled *payos.PaymentLink
paymentLinkCancelled, err := client.PaymentRequests.Cancel(context.Background(), orderCode, cancellationReason)
```

```go
// before
var webhookBody payos.WebhookType
var webhookData *payos.WebhookDataType
webhookData, err := payos.VerifyPaymentWebhookData(webhookBody)

// after
var webhookBody payos.Webhook
var webhookData interface{}
webhookData, err := client.Webhooks.VerifyData(context.Background(), webhookBody)
```

### Handling errors

The library now throws more specific error types for better error handling.

```go
// before
import "github.com/payOSHQ/payos-lib-golang/internal/apierror"

_, err := payos.CreatePaymentLink(paymentData)
if err != nil {
    if payosErr, ok := err.(*apierror.PayOSError); ok {
        fmt.Println(payosErr.Error())
    }
}

// after
import (
    "errors"
    "github.com/payOSHQ/payos-lib-golang/internal/apierror"
)

_, err := client.PaymentRequests.Create(context.Background(), paymentData)
if err != nil {
    var apiErr *apierror.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode)
        fmt.Println(apiErr.Code)
        fmt.Println(apiErr.Message)
    }
}
```
