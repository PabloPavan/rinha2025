package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type PaymentRecord struct {
	Timestamp time.Time
	Amount    float64
	Target    string
}

type Summary struct {
	Count  int
	Amount float64
}

var (
	mu               sync.RWMutex
	summaryDefault   Summary
	summaryFallback  Summary
	paymentLog       []PaymentRecord
)

func paymentsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	go func() {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		defer r.Body.Close()

		var data struct {
			Amount float64 `json:"amount"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return
		}

		for {
			target := ""
			var ok bool

			ok = forwardPayment("http://payment-processor-default:8080/payments", body)
			target = "default"

			if !ok {			
				ok = forwardPayment("http://payment-processor-fallback:8080/payments", body)
				target = "fallback"
			}
			
			if ok {
				log.Printf("Pagamento %.2f processado por %s", data.Amount, target)

				now := time.Now().UTC()
				mu.Lock()
				if target == "default" {
					summaryDefault.Count++
					summaryDefault.Amount += data.Amount
				} else {
					summaryFallback.Count++
					summaryFallback.Amount += data.Amount
				}
				paymentLog = append(paymentLog, PaymentRecord{
					Timestamp: now,
					Amount:    data.Amount,
					Target:    target,
				})
				mu.Unlock()
				break // sucesso, sai do loop
			}
			time.Sleep(100 * time.Millisecond) // evita loop agressivo
		}
	}()
}

func forwardPayment(url string, body []byte) bool {
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Erro ao enviar para %s: %v", url, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Serviço %s retornou status %d", url, resp.StatusCode)
		return false
	}
	return true
}

func paymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" && toStr == "" {
		mu.RLock()
		defer mu.RUnlock()
		json.NewEncoder(w).Encode(map[string]any{
			"default": map[string]any{
				"totalRequests": summaryDefault.Count,
				"totalAmount":   round(summaryDefault.Amount),
			},
			"fallback": map[string]any{
				"totalRequests": summaryFallback.Count,
				"totalAmount":   round(summaryFallback.Amount),
			},
		})
		return
	}

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

	mu.RLock()
	defer mu.RUnlock()
	for _, p := range paymentLog {
		if (fromStr == "" || !p.Timestamp.Before(from)) &&
			(toStr == "" || !p.Timestamp.After(to)) {
			if p.Target == "default" {
				defCount++
				defAmt += p.Amount
			} else if p.Target == "fallback" {
				fbCount++
				fbAmt += p.Amount
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"default": map[string]any{
			"totalRequests": defCount,
			"totalAmount":   round(defAmt),
		},
		"fallback": map[string]any{
			"totalRequests": fbCount,
			"totalAmount":   round(fbAmt),
		},
	})
}

func round(val float64) float64 {
	str := strconv.FormatFloat(val, 'f', 2, 64)
	r, _ := strconv.ParseFloat(str, 64)
	return r
}

func main() {
	http.HandleFunc("/payments", paymentsHandler)
	http.HandleFunc("/payments-summary", paymentsSummaryHandler)
	log.Println("Listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}
