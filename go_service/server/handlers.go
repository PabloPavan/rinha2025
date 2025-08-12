package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	payments "github.com/PabloPavan/rinha2025/payments"
)

func (s *Server) PaymentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Erro ao ler body: %v", err)
		return
	}
	r.Body.Close()

	var data payments.PaymentData
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("JSON inválido: %v", err)
		return
	}

	s.pool.Submit(func() {
		record, ok := s.paymentService.ProcessPayment(data)

		if ok {
			s.mu.Lock()
			s.paymentLog = append(s.paymentLog, record)
			s.mu.Unlock()
		}
	})
}

func (s *Server) PaymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	w.Header().Set("Content-Type", "application/json")

	var from, to time.Time
	var err error
	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "invalid from timestamp", http.StatusBadRequest)
			return
		}
	}
	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, "invalid to timestamp", http.StatusBadRequest)
			return
		}
	}

	var defCount, fbCount int
	var defAmt, fbAmt float64

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.paymentLog {
		if (fromStr == "" || !p.Timestamp.Before(from)) &&
			(toStr == "" || !p.Timestamp.After(to)) {
			switch p.Target {
			case payments.TargetDefault:
				defCount++
				defAmt += p.Amount
			case payments.TargetFallback:
				fbCount++
				fbAmt += p.Amount
			}
		}
	}

	resp := payments.PaymentsSummaryResponse{
		Default:  payments.SummaryDTO{TotalRequests: defCount, TotalAmount: defAmt},
		Fallback: payments.SummaryDTO{TotalRequests: fbCount, TotalAmount: fbAmt},
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
