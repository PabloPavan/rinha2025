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
	mu              sync.RWMutex
	summaryDefault  Summary
	summaryFallback Summary
	paymentLog      []PaymentRecord
)

type Task func()

type Pool struct {
	tasks chan Task
	wg    sync.WaitGroup
}

func NewPool(numWorkers int) *Pool {
	p := &Pool{
		tasks: make(chan Task, 10000),
	}
	p.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		task()
	}
}

func (p *Pool) Submit(task Task) {
	select {
	case p.tasks <- task:
		// enviado com sucesso
	default:
		log.Println("WARNING: pool cheio, tarefa descartada ou atrasada")
		// ou fazer algo como: salvar em fallback, re-enfileirar depois etc
	}
}

func (p *Pool) Wait() {
	close(p.tasks)
	p.wg.Wait()
}

func makePaymentsHandler(pool *Pool, breakers map[string]*CircuitBreaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		var data struct {
			Amount float64 `json:"amount"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return
		}

		pool.Submit(func() {
			// defer fmt.Println("Fim da goroutine")
			// fmt.Println("Início da goroutine")

			for {
				var target string
				ok := false

				if breakers["default"].Allow() {
					ok = forwardPayment("http://payment-processor-default:8080/payments", body)
					target = "default"

					if ok {
						breakers["default"].MarkSuccess()
					} else {
						breakers["default"].MarkFailure()
					}
				}

				if !ok && breakers["fallback"].Allow() {
					ok = forwardPayment("http://payment-processor-fallback:8080/payments", body)
					target = "fallback"

					if ok {
						breakers["fallback"].MarkSuccess()
					} else {
						breakers["fallback"].MarkFailure()
					}
				}

				if ok {
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
					break
				}

				time.Sleep(200 * time.Millisecond)
			}
		})
	}
}

func forwardPayment(url string, body []byte) bool {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
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

	log.Printf("Data do request: to %s from %s \n", fromStr, toStr)

	var defCount, fbCount int
	var defAmt, fbAmt float64

	mu.RLock()
	defer mu.RUnlock()
	for _, p := range paymentLog {
		if (fromStr == "" || !p.Timestamp.Before(from)) &&
			(toStr == "" || !p.Timestamp.After(to)) {
			switch p.Target {
			case "default":
				defCount++
				defAmt += p.Amount
			case "fallback":
				fbCount++
				fbAmt += p.Amount
			}
		}
	}

	data := map[string]any{
		"default": map[string]any{
			"totalRequests": defCount,
			"totalAmount":   round(defAmt),
		},
		"fallback": map[string]any{
			"totalRequests": fbCount,
			"totalAmount":   round(fbAmt),
		},
	}

	jsonStr, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Println("Erro ao gerar JSON:", err)
	} else {
		log.Println("Resumo dos pagamentos:\n", string(jsonStr))
	}

	json.NewEncoder(w).Encode(data)
}

func round(val float64) float64 {
	str := strconv.FormatFloat(val, 'f', 2, 64)
	r, _ := strconv.ParseFloat(str, 64)
	return r
}

func main() {
	pool := NewPool(10)
	defer pool.Wait()

	breakers := map[string]*CircuitBreaker{
		"default":  NewCircuitBreaker(3, 10*time.Second),
		"fallback": NewCircuitBreaker(3, 10*time.Second),
	}

	http.HandleFunc("/payments", makePaymentsHandler(pool, breakers))
	http.HandleFunc("/payments-summary", paymentsSummaryHandler)
	log.Println("Listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}
