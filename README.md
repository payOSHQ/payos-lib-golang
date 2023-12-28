# PayOs Go Package

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

### Confirm Wehook

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
