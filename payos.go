package payos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const PayOSBaseUrl = "https://dev.api-merchant.payos.vn"

var PayOSClientId string
var PayOSApiKey string
var PayOSChecksumKey string

// Set ClientId, APIKey, ChecksumKey
func Key(clientId string, apiKey string, checksumKey string) error {
	if clientId == "" || apiKey == "" || checksumKey == "" {
		return errors.New("invalid key")
	}
	PayOSClientId = clientId
	PayOSApiKey = apiKey
	PayOSChecksumKey = checksumKey
	return nil
}

// Create a payment link for the order data passed in the parameter
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
			msgError := fmt.Sprintf("%s %s must not be undefined or null.", InvalidParameterErrorMessage, strings.Join(keysError, ", "))
			return nil, NewPayOSError(InvalidParameterErrorCode, msgError)
		}
	}
	url := fmt.Sprintf("%s/v2/payment-requests", PayOSBaseUrl)
	signaturePaymentRequest, _ := CreateSignatureOfPaymentRequest(paymentData, PayOSChecksumKey)
	paymentData.Signature = &signaturePaymentRequest
	checkoutRequest, err := json.Marshal(paymentData)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(checkoutRequest))
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}
	defer res.Body.Close()

	var paymentLinkRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &paymentLinkRes)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	if paymentLinkRes.Code == "00" {
		paymentLinkResSignatute, _ := CreateSignatureFromObj(paymentLinkRes.Data, PayOSChecksumKey)
		if paymentLinkResSignatute != *paymentLinkRes.Signature {
			return nil, NewPayOSError(DataNotIntegrityErrorCode, DataNotIntegrityErrorMessage)
		}
		if paymentLinkRes.Data != nil {
			jsonData, err := json.Marshal(paymentLinkRes.Data)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			var paymentLinkData CheckoutResponseDataType
			err = json.Unmarshal(jsonData, &paymentLinkData)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			return &paymentLinkData, nil
		}
	}

	return nil, NewPayOSError(paymentLinkRes.Code, paymentLinkRes.Desc)
}

// Get payment information of an order that has created a payment link
func GetPaymentLinkInformation(orderCode string) (*PaymentLinkDataType, error) {
	if len(orderCode) == 0 {
		return nil, NewPayOSError(InvalidParameterErrorCode, InvalidParameterErrorMessage)
	}

	url := fmt.Sprintf("%s/v2/payment-requests/%s", PayOSBaseUrl, orderCode)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}
	defer res.Body.Close()

	var paymentLinkInfoRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &paymentLinkInfoRes)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	if paymentLinkInfoRes.Code == "00" {
		paymentLinkInfoResSignature, _ := CreateSignatureFromObj(paymentLinkInfoRes.Data, PayOSChecksumKey)
		if paymentLinkInfoResSignature != *paymentLinkInfoRes.Signature {
			return nil, NewPayOSError(DataNotIntegrityErrorCode, DataNotIntegrityErrorMessage)
		}

		if paymentLinkInfoRes.Data != nil {
			jsonData, err := json.Marshal(paymentLinkInfoRes.Data)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			var paymentLinkInfoData PaymentLinkDataType
			err = json.Unmarshal(jsonData, &paymentLinkInfoData)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			return &paymentLinkInfoData, nil
		}
	}

	return nil, NewPayOSError(paymentLinkInfoRes.Code, paymentLinkInfoRes.Desc)
}

// Cancel the payment link of the order
func CancelPaymentLink(orderCode string, cancellationReason *string) (*PaymentLinkDataType, error) {
	if len(orderCode) == 0 {
		return nil, NewPayOSError(InvalidParameterErrorCode, InvalidParameterErrorMessage)
	}

	data := CancelPaymentLinkRequestType{
		CancellationReason: cancellationReason,
	}
	cancelRequest, err := json.Marshal(data)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	url := fmt.Sprintf("%s/v2/payment-requests/%s/cancel", PayOSBaseUrl, orderCode)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(cancelRequest))
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}
	defer res.Body.Close()

	var cancelPaymentLinkRes PayOSResponseType
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &cancelPaymentLinkRes)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	if cancelPaymentLinkRes.Code == "00" {
		paymentLinkResSignatute, _ := CreateSignatureFromObj(cancelPaymentLinkRes.Data, PayOSChecksumKey)
		if paymentLinkResSignatute != *cancelPaymentLinkRes.Signature {
			return nil, NewPayOSError(DataNotIntegrityErrorCode, DataNotIntegrityErrorMessage)
		}
		if cancelPaymentLinkRes.Data != nil {
			jsonData, err := json.Marshal(cancelPaymentLinkRes.Data)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			var cancelPaymentLinkData PaymentLinkDataType
			err = json.Unmarshal(jsonData, &cancelPaymentLinkData)
			if err != nil {
				return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
			}

			return &cancelPaymentLinkData, nil
		}
	}

	return nil, NewPayOSError(cancelPaymentLinkRes.Code, cancelPaymentLinkRes.Desc)
}

// Validate the Webhook URL of a payment channel and add or update the Webhook URL for that Payment Channel if successful
func ConfirmWebhook(webhookUrl string) (string, error) {
	if len(webhookUrl) == 0 {
		return "", NewPayOSError(InvalidParameterErrorCode, InvalidParameterErrorMessage)
	}

	data := ConfirmWebhookRequestType{
		WebhookUrl: webhookUrl,
	}
	webhookRequest, err := json.Marshal(data)
	if err != nil {
		return "", NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	url := fmt.Sprintf("%v/confirm-webhook", PayOSBaseUrl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(webhookRequest))
	if err != nil {
		return "", NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}

	req.Header.Set("x-client-id", PayOSClientId)
	req.Header.Set("x-api-key", PayOSApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", NewPayOSError(InternalServerErrorErrorCode, err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode == 400 {
		return "", NewPayOSError(WebhookURLInvalidErrorCode, WebhookURLInvalidErrorMessage)
	} else if res.StatusCode == 401 {
		return "", NewPayOSError(UnauthorizedErrorCode, UnauthorizedErrorMessage)
	} else if res.StatusCode >= 500 {
		return "", NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
	}

	return webhookUrl, nil
}

// Verify data received via webhook after payment
func VerifyPaymentWebhookData(webhookBody WebhookType) (*WebhookDataType, error) {
	if webhookBody.Data == nil {
		return nil, NewPayOSError(NoDataErrorCode, NoDataErrorMessage)
	}
	if webhookBody.Signature == "" {
		return nil, NewPayOSError(NoSignatureErrorCode, NoSignatureErrorMessage)
	}

	signData, err := CreateSignatureFromObj(webhookBody.Data, PayOSChecksumKey)
	if err != nil {
		return nil, NewPayOSError(InternalServerErrorErrorCode, InternalServerErrorErrorMessage)
	}

	if signData != webhookBody.Signature {
		return nil, NewPayOSError(DataNotIntegrityErrorCode, DataNotIntegrityErrorMessage)
	}

	return webhookBody.Data, nil
}
