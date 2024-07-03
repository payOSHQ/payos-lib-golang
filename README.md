# PayOS Go Package

[![Go Reference](https://pkg.go.dev/badge/github.com/payOSHQ/payos-lib-golang.svg)](https://pkg.go.dev/github.com/payOSHQ/payos-lib-golang)

## Installation

```bash
go get github.com/payOSHQ/payos-lib-golang
```

## Usage

### Create Payment Link

```go
package main

import (
    "fmt"
    "log"

    "github.com/payOSHQ/payos-lib-golang"
)

func main(){
    payos.Key(clientId, apiKey,checksumKey)
    // or with your partner code
    // payos.Key(clientId, apiKey,checksumKey, partnerCode)
    body := CheckoutRequestType{
		OrderCode:   12345,
		Amount:      2000,
		Description: "Thanh toán đơn hàng",
		CancelUrl:   "http://localhost:8080/cancel/",
		ReturnUrl:   "http://localhost:8080/success/",
	}

	data, err := CreatePaymentLink(body)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(data)
}
```

### Get Payment Link Information

```go
package main

import (
    "fmt"
    "log"

    "github.com/payOSHQ/payos-lib-golang"
)

func main(){
    payos.Key(clientId, apiKey,checksumKey)
	data, err := GetPaymentLinkInformation("12345")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(data)
}
```

### Cancel Payment Link

```go
package main

import (
    "fmt"
    "log"

    "github.com/payOSHQ/payos-lib-golang"
)

func main(){
    payos.Key(clientId, apiKey,checksumKey)
    cancelReason := "Khách hàng hủy đơn hàng"
    data, err := CancelPaymentLink("12345", &cancelReason)
}
```

### Confirm Webhook

```go
package main

import (
    "fmt"
    "log"

    "github.com/payOSHQ/payos-lib-golang"
)

func main(){
    payos.Key(clientId, apiKey,checksumKey)
    data, err := ConfirmWebhook("http://yourdomain.com/webhook/")
}
```

### Verify Webhook

```go
package main

import (
    "fmt"
    "log"

    "github.com/payOSHQ/payos-lib-golang"
)

func main(){
    payos.Key(clientId, apiKey,checksumKey)
    body := WebhookType{}
	data, err := VerifyPaymentWebhookData(body)
}
```

## Development

1. Clone the repository
2. Install dependencies

```bash
go mod tidy
```

3. Run tests

```bash
go test
```

4. Commit and create new tag

```bash
git commit -m "Your message"
git tag v0.0.1
git push origin v0.0.1
```

5. Publish to [pkg.go.dev](https://pkg.go.dev/)

```bash
GOPROXY=proxy.golang.org
go list -m github.com/payOSHQ/payos-lib-golang@v0.1.0
```
