package payments

import (
	"time"
)

type PaymentTarget int

const (
	TargetDefault PaymentTarget = iota
	TargetFallback
	TargetUnknown
)

type PaymentServers struct {
	UrlDefault  string
	UrlFallBack string
}

type PaymentRecord struct {
	Timestamp time.Time
	Amount    float64
	Target    PaymentTarget
}

type PaymentData struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"`
}

type SummaryDTO struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentsSummaryResponse struct {
	Default  SummaryDTO `json:"default"`
	Fallback SummaryDTO `json:"fallback"`
}
