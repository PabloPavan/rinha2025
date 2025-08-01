package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"sync"
)

var (
	paymentsData []string
	mu           sync.Mutex
)

func paymentsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	go func() {
		body, err := io.ReadAll(r.Body)

		mu.Lock()
		paymentsData = append(paymentsData, string(body))
		mu.Unlock()

		resp, err := http.Post("http://payment-processor-default:8080", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("Erro ao encaminhar para outro serviço: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}

func main() {
	http.HandleFunc("/payments", paymentsHandler)
	log.Println("Listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}