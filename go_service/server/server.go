package server

import (
	"net/http"
	"sync"

	payments "github.com/PabloPavan/rinha2025/payments"
	workers "github.com/PabloPavan/rinha2025/workers"
)

type Server struct {
	pool           *workers.Pool
	paymentService *payments.Service
	sharedClient   *http.Client
	mu             sync.RWMutex
	name           string
	paymentLog     []payments.PaymentRecord
}

func NewServer(pool *workers.Pool, paymentService *payments.Service, client *http.Client, name string) *Server {
	return &Server{
		pool:           pool,
		paymentService: paymentService,
		sharedClient:   client,
		name:           name,
		paymentLog:     make([]payments.PaymentRecord, 0),
	}
}
