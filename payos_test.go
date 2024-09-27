package payos

import (
	"log"
	"strconv"
	"testing"
	"time"
)

const clientId = "your_client_id"
const apiKey = "your_api_key"
const checksumKey = "your_checksum_key"

const webhookUrl = "your_webhook_url"

var partnerCode = "your_partner_code"

var orderCode int64

func GenerateNumber() {
	millis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	orderCode, _ = strconv.ParseInt(millis[len(millis)-6:], 10, 64)
}

func TestKey(t *testing.T) {
	err := Key(clientId, apiKey, checksumKey)
	if err != nil {
		t.Errorf("Key returned an error: %v", err)
	}

}

func TestCreatePaymentLink(t *testing.T) {
	Key(clientId, apiKey, checksumKey)
	GenerateNumber()
	body := CheckoutRequestType{
		OrderCode:   orderCode,
		Amount:      500000,
		Description: "Thanh toan don hang",
		Items:       []Item{{"Mỳ tôm Hảo Hảo", 3, 2000}, {"Mỳ tôm Omachi", 3, 9000}},
		CancelUrl:   "http://localhost:8080/cancel/",
		ReturnUrl:   "http://localhost:8080/success/",
	}

	data, err := CreatePaymentLink(body)

	if err != nil {
		t.Errorf("CreatePaymentLink returned an error: %v", err)
	}
	if data.CheckoutUrl == "" {
		t.Errorf("CreatePaymentLink returned an empty CheckoutUrl")
	}
	t.Logf("%d", orderCode)
}

func TestGetPaymentLinkInformation(t *testing.T) {
	Key(clientId, apiKey, checksumKey)
	data, err := GetPaymentLinkInformation(strconv.FormatInt(orderCode, 10))
	if data != nil {
		t.Log(data.Transactions)
	} else {
		t.Logf("Data is nil")
	}

	if err != nil {
		t.Errorf("GetPaymentLinkInformation returned an error:%v", err)
	}
	if data.OrderCode == 0 {
		t.Errorf("GetPaymentLinkInformation returned an empty information")
	}
}

func TestCancelPaymentLink(t *testing.T) {
	Key(clientId, apiKey, checksumKey)
	cancelReason := "Huy"
	data, err := CancelPaymentLink(strconv.FormatInt(orderCode, 10), &cancelReason)

	if err != nil {
		t.Errorf("CancelPaymentLink returned an error:%v", err)
	}
	if data.OrderCode == 0 {
		t.Errorf("CancelPaymentLink returned an empty information")
	}
}

func TestConfirmWebhook(t *testing.T) {
	Key(clientId, apiKey, checksumKey)
	data, err := ConfirmWebhook(webhookUrl)

	if err != nil {
		t.Errorf("ConfirmWebhook returned an error:%v", err)
	}
	if data == "" {
		t.Errorf("ConfirmWebhook returned an empty information")
	}
}

func TestVerifyPaymentWebhookData(t *testing.T) {
	Key(clientId, apiKey, checksumKey)
	webhookData := WebhookDataType{
		AccountNumber:          "0004100033726006",
		Amount:                 2000,
		Code:                   "00",
		CounterAccountBankId:   nil,
		CounterAccountBankName: nil,
		CounterAccountName:     nil,
		CounterAccountNumber:   nil,
		Currency:               "VND",
		Desc:                   "success",
		Description:            "ND:CT DEN:416713881564 CSHN5MB59H7 Thanh toan don hang 92; tai Napas",
		OrderCode:              839378987236498,
		PaymentLinkId:          "5d292da4439446b2842a36aca961e586",
		Reference:              "FT240120B3JQ\\BNK",
		TransactionDateTime:    "2024-01-12 14:34:00",
		VirtualAccountName:     nil,
		VirtualAccountNumber:   func(s string) *string { return &s }("CAS004100033726006"),
	}
	signature, _ := CreateSignatureFromObj(webhookData, checksumKey)
	body := WebhookType{
		Code:      "00",
		Desc:      "success",
		Success:   true,
		Signature: signature,
		Data:      &webhookData,
	}
	data, err := VerifyPaymentWebhookData(body)
	if data != nil {
		t.Logf("%s", data.Description)
	}

	if err != nil {
		t.Errorf("VerifyPaymentWebhookData returned an error:%v", err)
	}
	if data == nil {
		t.Errorf("VerifyPaymentWebhookData returned an empty information")
	}
}

func TestSignature(t *testing.T) {
	body := PaymentLinkDataType{
		Id:              "47e2f8e275a44bc0bf455af261d4f2e6",
		OrderCode:       92,
		Amount:          5000,
		AmountPaid:      5000,
		AmountRemaining: 0,
		Status:          "PAID",
		CreateAt:        "2024-06-15T20:53:43+07:00",
		Transactions: []TransactionType{
			{
				Reference:              "",
				Amount:                 5000,
				AccountNumber:          "100827042003",
				Description:            "ND:CT DEN:416713881564 CSHN5MB59H7 Thanh toan don hang 92; tai Napas",
				TransactionDateTime:    "2024-06-15T20:56:00+07:00",
				VirtualAccountName:     nil,
				VirtualAccountNumber:   nil,
				CounterAccountBankId:   nil,
				CounterAccountBankName: nil,
				CounterAccountName:     nil,
				CounterAccountNumber:   nil,
			},
		},
		CancellationReason: nil,
		CancelAt:           nil,
	}

	data, err := CreateSignatureFromObj(body, checksumKey)

	if err != nil {
		t.Error(err)
	}

	log.Println(data)
}
