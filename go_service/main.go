package main

import (
//	"log"
	"net/http"
	"time"
	"net"
	
	payments "github.com/PabloPavan/rinha2025/payments"
	server "github.com/PabloPavan/rinha2025/server"
	utils "github.com/PabloPavan/rinha2025/utils"
	workers "github.com/PabloPavan/rinha2025/workers"
)

func main() {
	pool := workers.NewPool(utils.GetEnvInt("WORKERS", 2))
	defer pool.Wait()

	dialer := &net.Dialer{
		Timeout:   800 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	sharedTransport := &http.Transport{
			DialContext:       dialer.DialContext,
		ForceAttemptHTTP2: false,

		MaxConnsPerHost:     32,
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     30 * time.Second,

		TLSHandshakeTimeout:   2 * time.Second,
		ExpectContinueTimeout: 0,
		ResponseHeaderTimeout: 2 * time.Second,
		DisableCompression:    true,
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
	myName := utils.GetEnvOrDefault("NAME", "")

	appServer := server.NewServer(pool, paymentService, sharedClient, myName)

	http.HandleFunc("/payments", appServer.PaymentsHandler)
	http.HandleFunc("/payments-summary", appServer.PaymentsSummaryHandler)

//	log.Println("Listening on :9999")
	http.ListenAndServe(":9999", nil)
}
