package main

import (
	"log"
	"net/http"
	"time"

	payments "github.com/PabloPavan/rinha2025/payments"
	server "github.com/PabloPavan/rinha2025/server"
	utils "github.com/PabloPavan/rinha2025/utils"
	workers "github.com/PabloPavan/rinha2025/workers"
)

func main() {

	pool := workers.NewPool(10)
	defer pool.Wait()

	sharedTransport := &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
	}

	sharedClient := &http.Client{
		Timeout:   2 * time.Second,
		Transport: sharedTransport,
	}

	breakers := map[payments.PaymentTarget]*utils.Breaker{
		payments.TargetDefault:  utils.NewCircuitBreaker(1, 500*time.Millisecond),
		payments.TargetFallback: utils.NewCircuitBreaker(1, 1*time.Second),
	}

	paymentServers := &payments.PaymentServers{
		UrlDefault:  utils.GetEnvOrDefault("PAYMENT_URL_DEFAULT", "http://payment-processor-default:8080/payments"),
		UrlFallBack: utils.GetEnvOrDefault("PAYMENT_URL_FALLBACK", "http://payment-processor-fallback:8080/payments"),
	}

	paymentService := payments.NewService(sharedClient, breakers, paymentServers)

	appServer := server.NewServer(pool, paymentService, sharedClient)

	http.HandleFunc("/payments", appServer.PaymentsHandler)
	http.HandleFunc("/payments-summary", appServer.PaymentsSummaryHandler)

	log.Println("Listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}
