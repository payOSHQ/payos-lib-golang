package payos

const (
	NoSignatureErrorMessage         = "No signature."
	NoDataErrorMessage              = "No data."
	InvalidSignatureErrorMessage    = "Invalid signature."
	DataNotIntegrityErrorMessage    = "The data is unreliable because the signature of the response does not match the signature of the data."
	WebhookURLInvalidErrorMessage   = "Webhook URL invalid."
	UnauthorizedErrorMessage        = "Unauthorized."
	InternalServerErrorErrorMessage = "Internal Server Error."
	InvalidParameterErrorMessage    = "Invalid Parameter."
	OrderCodeOuOfRange              = "orderCode is out of range."
)

const (
	InternalServerErrorErrorCode = "20"
	UnauthorizedErrorCode        = "401"
	InvalidParameterErrorCode    = "21"
	NoSignatureErrorCode         = "22"
	NoDataErrorCode              = "23"
	InvalidSignatureErrorCode    = "24"
	DataNotIntegrityErrorCode    = "25"
	WebhookURLInvalidErrorCode   = "26"
)
