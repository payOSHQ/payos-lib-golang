package payos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
	"github.com/payOSHQ/payos-lib-golang/v2/internal/crypto"
)

// Deprecated: Use NewPayOS() constructor instead
var PayOSClientId string

// Deprecated: Use NewPayOS() constructor instead
var PayOSApiKey string

// Deprecated: Use NewPayOS() constructor instead
var PayOSChecksumKey string

// Deprecated: Use NewPayOS() constructor instead
var PayOSPartnerCode string

// PayOS is the main client for interacting with PayOS API
type PayOS struct {
	Client          *Client
	PaymentRequests *PaymentRequests
	Webhooks        *Webhooks
	Payouts         *Payouts
	PayoutsAccount  *PayoutsAccount
}

// NewPayOS creates a new PayOS client with the provided options
// This is the recommended way to create a PayOS client
func NewPayOS(opts *PayOSOptions) (*PayOS, error) {
	client, err := NewClient(opts)
	if err != nil {
		return nil, err
	}

	payos := &PayOS{
		Client: client,
	}

	// Initialize resources
	payos.PaymentRequests = newPaymentRequests(client)
	payos.Webhooks = &Webhooks{client: client}
	payos.Payouts = newPayouts(client)
	payos.PayoutsAccount = newPayoutsAccount(client)

	return payos, nil
}

// Key sets the global client credentials
//
// Deprecated: Use NewPayOS() constructor with PayOSOptions instead
// Set ClientId, APIKey, ChecksumKey and PartnerCode(a string, optional)
func Key(clientId string, apiKey string, checksumKey string, partnerCode ...string) error {
	if clientId == "" || apiKey == "" || checksumKey == "" {
		return errors.New("invalid key")
	}
	if len(partnerCode) > 1 {
		return errors.New("invalid partner code")
	}
	PayOSClientId = clientId
	PayOSApiKey = apiKey
	PayOSChecksumKey = checksumKey
	if len(partnerCode) == 1 {
		PayOSPartnerCode = partnerCode[0]
	}
	return nil
}

// CreatePaymentLink creates a payment link for the order data passed in the parameter
//
// Deprecated: Use NewPayOS() and client.PaymentRequests.Create() instead
// Example:
//
//	client, _ := payos.NewPayOS(&payos.PayOSOptions{
//	    ClientId: "your-client-id",
//	    ApiKey: "your-api-key",
//	    ChecksumKey: "your-checksum-key",
//	})
//	result, err := client.PaymentRequests.Create(context.Background(), paymentData)
func CreatePaymentLink(paymentData CheckoutRequestType) (*CheckoutResponseDataType, error) {
	if paymentData.OrderCode == 0 || paymentData.Amount == 0 || paymentData.Description == "" || paymentData.CancelUrl == "" || paymentData.ReturnUrl == "" {
		requiredPaymentData := CheckoutRequestType{
			OrderCode:   paymentData.OrderCode,
			Amount:      paymentData.Amount,
			ReturnUrl:   paymentData.ReturnUrl,
			CancelUrl:   paymentData.CancelUrl,
			Description: paymentData.Description,
		}
		requiredKeys := []string{"OrderCode", "Amount", "ReturnUrl", "CancelUrl", "Description"}
		keysError := []string{}
		for _, key := range requiredKeys {
			switch key {
			case "OrderCode":
				if requiredPaymentData.OrderCode == 0 {
					keysError = append(keysError, key)
				}
			case "Amount":
				if requiredPaymentData.Amount == 0 {
					keysError = append(keysError, key)
				}
			case "ReturnUrl":
				if requiredPaymentData.ReturnUrl == "" {
					keysError = append(keysError, key)
				}
			case "CancelUrl":
				if requiredPaymentData.CancelUrl == "" {
					keysError = append(keysError, key)
				}
			case "Description":
				if requiredPaymentData.Description == "" {
					keysError = append(keysError, key)
				}
			}
		}

		if len(keysError) > 0 {
			msgError := fmt.Sprintf("%s must not be undefined or null.", strings.Join(keysError, ", "))
			return nil, apierror.NewPayOSError(msgError)
		}
	}

	// orderCode in range [-2^53+1, 2^53 -1]
	if paymentData.OrderCode < -9007199254740991 || paymentData.OrderCode > 9007199254740991 {
		return nil, apierror.NewPayOSError("order code out of range")
	}

	url := fmt.Sprintf("%s/v2/payment-requests", PayOSBaseUrl)
	signaturePaymentRequest, _ := crypto.CreateSignatureOfPaymentRequest(paymentData, PayOSChecksumKey)
	paymentData.Signature = &signaturePaymentRequest
	checkoutRequest, err := json.Marshal(paymentData)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(checkoutRequest))
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")
	if PayOSPartnerCode != "" {
		req.Header.Set("x-partner-code", PayOSPartnerCode)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}
	defer res.Body.Close()

	var paymentLinkRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &paymentLinkRes)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	if paymentLinkRes.Code == "00" {
		paymentLinkResSignature, _ := crypto.CreateSignatureFromObj(paymentLinkRes.Data, PayOSChecksumKey)
		if paymentLinkResSignature != *paymentLinkRes.Signature {
			return nil, apierror.NewPayOSError("data not integrity")
		}
		if paymentLinkRes.Data != nil {
			jsonData, err := json.Marshal(paymentLinkRes.Data)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			var paymentLinkData CheckoutResponseDataType
			err = json.Unmarshal(jsonData, &paymentLinkData)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			return &paymentLinkData, nil
		}
	}

	return nil, apierror.NewPayOSError(paymentLinkRes.Desc)
}

// GetPaymentLinkInformation gets payment information of an order that has created a payment link
//
// Deprecated: Use NewPayOS() and client.PaymentRequests.Get() instead
// Example:
//
//	client, _ := payos.NewPayOS(&payos.PayOSOptions{
//	    ClientId: "your-client-id",
//	    ApiKey: "your-api-key",
//	    ChecksumKey: "your-checksum-key",
//	})
//	result, err := client.PaymentRequests.Get(context.Background(), orderCode)
func GetPaymentLinkInformation(orderCode string) (*PaymentLinkDataType, error) {
	if len(orderCode) == 0 {
		return nil, apierror.NewPayOSError("invalid params")
	}

	url := fmt.Sprintf("%s/v2/payment-requests/%s", PayOSBaseUrl, orderCode)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}
	defer res.Body.Close()

	var paymentLinkInfoRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &paymentLinkInfoRes)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	if paymentLinkInfoRes.Code == "00" {
		paymentLinkInfoResSignature, _ := crypto.CreateSignatureFromObj(paymentLinkInfoRes.Data, PayOSChecksumKey)
		if paymentLinkInfoResSignature != *paymentLinkInfoRes.Signature {
			return nil, apierror.NewPayOSError("data not integrity")
		}

		if paymentLinkInfoRes.Data != nil {
			jsonData, err := json.Marshal(paymentLinkInfoRes.Data)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			var paymentLinkInfoData PaymentLinkDataType
			err = json.Unmarshal(jsonData, &paymentLinkInfoData)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			return &paymentLinkInfoData, nil
		}
	}

	return nil, apierror.NewPayOSError(paymentLinkInfoRes.Desc)
}

// CancelPaymentLink cancels the payment link of the order
//
// Deprecated: Use NewPayOS() and client.PaymentRequests.Cancel() instead
// Example:
//
//	client, _ := payos.NewPayOS(&payos.PayOSOptions{
//	    ClientId: "your-client-id",
//	    ApiKey: "your-api-key",
//	    ChecksumKey: "your-checksum-key",
//	})
//	result, err := client.PaymentRequests.Cancel(context.Background(), orderCode, cancellationReason)
func CancelPaymentLink(orderCode string, cancellationReason *string) (*PaymentLinkDataType, error) {
	if len(orderCode) == 0 {
		return nil, apierror.NewPayOSError("invalid params")
	}

	data := CancelPaymentLinkRequestType{
		CancellationReason: cancellationReason,
	}
	cancelRequest, err := json.Marshal(data)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	url := fmt.Sprintf("%s/v2/payment-requests/%s/cancel", PayOSBaseUrl, orderCode)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(cancelRequest))
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}
	defer res.Body.Close()

	var cancelPaymentLinkRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &cancelPaymentLinkRes)
	if err != nil {
		return nil, apierror.NewPayOSError(err.Error())
	}

	if cancelPaymentLinkRes.Code == "00" {
		paymentLinkResSignature, _ := crypto.CreateSignatureFromObj(cancelPaymentLinkRes.Data, PayOSChecksumKey)
		if paymentLinkResSignature != *cancelPaymentLinkRes.Signature {
			return nil, apierror.NewPayOSError("data not integrity")
		}
		if cancelPaymentLinkRes.Data != nil {
			jsonData, err := json.Marshal(cancelPaymentLinkRes.Data)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			var cancelPaymentLinkData PaymentLinkDataType
			err = json.Unmarshal(jsonData, &cancelPaymentLinkData)
			if err != nil {
				return nil, apierror.NewPayOSError("internal server error")
			}

			return &cancelPaymentLinkData, nil
		}
	}

	return nil, apierror.NewPayOSError(cancelPaymentLinkRes.Desc)
}

// ConfirmWebhook validates the Webhook URL of a payment channel and adds or updates the Webhook URL for that Payment Channel if successful
//
// Deprecated: Use NewPayOS() and client.Webhooks.Confirm() instead
// Example:
//
//	client, _ := payos.NewPayOS(&payos.PayOSOptions{
//	    ClientId: "your-client-id",
//	    ApiKey: "your-api-key",
//	    ChecksumKey: "your-checksum-key",
//	})
//	result, err := client.Webhooks.Confirm(context.Background(), webhookUrl)
func ConfirmWebhook(webhookUrl string) (string, error) {
	if len(webhookUrl) == 0 {
		return "", apierror.NewPayOSError("invalid params")
	}

	data := ConfirmWebhookRequestType{
		WebhookUrl: webhookUrl,
	}
	webhookRequest, err := json.Marshal(data)
	if err != nil {
		return "", apierror.NewPayOSError(err.Error())
	}

	url := fmt.Sprintf("%v/confirm-webhook", PayOSBaseUrl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(webhookRequest))
	if err != nil {
		return "", apierror.NewPayOSError(err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", apierror.NewPayOSError(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode == 400 {
		return "", apierror.NewPayOSError("webhook URL invalid")
	} else if res.StatusCode == 401 {
		return "", apierror.NewPayOSError("unauthorize")
	} else if res.StatusCode >= 500 {
		return "", apierror.NewPayOSError("internal server error")
	}

	return webhookUrl, nil
}

// VerifyPaymentWebhookData verifies data received via webhook after payment
//
// Deprecated: Use NewPayOS() and client.Webhooks.VerifyData() instead
// Example:
//
//	client, _ := payos.NewPayOS(&payos.PayOSOptions{
//	    ClientId: "your-client-id",
//	    ApiKey: "your-api-key",
//	    ChecksumKey: "your-checksum-key",
//	})
//	result, err := client.Webhooks.VerifyWebhookData(webhookBody, client.checksumKey)
func VerifyPaymentWebhookData(webhookBody WebhookType) (*WebhookDataType, error) {
	if webhookBody.Data == nil {
		return nil, apierror.NewPayOSError("data invalid")
	}
	if webhookBody.Signature == "" {
		return nil, apierror.NewPayOSError("signature invalid")
	}

	signData, err := crypto.CreateSignatureFromObj(webhookBody.Data, PayOSChecksumKey)
	if err != nil {
		return nil, apierror.NewPayOSError("internal server error")
	}

	if signData != webhookBody.Signature {
		return nil, apierror.NewPayOSError("data not integrity")
	}

	return webhookBody.Data, nil
}
