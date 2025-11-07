package payos

// ========================
// Common Response Types
// ========================

// PayOSResponseType represents the standard API response wrapper
type PayOSResponseType struct {
	Code      string      `json:"code"`
	Desc      string      `json:"desc"`
	Data      interface{} `json:"data"`
	Signature *string     `json:"signature"`
}

// FileDownloadResponse represents a file download response
type FileDownloadResponse struct {
	Filename    *string `json:"filename,omitempty"`
	ContentType string  `json:"contentType"`
	Size        *int64  `json:"size,omitempty"`
	Data        []byte  `json:"data"`
}
