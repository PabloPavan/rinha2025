package payments

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	utils "github.com/PabloPavan/rinha2025/utils"
)

type Service struct {
	client   *http.Client
	breakers map[PaymentTarget]*utils.Breaker
}

func NewService(client *http.Client, breakers map[PaymentTarget]*utils.Breaker) *Service {
	return &Service{
		client:   client,
		breakers: breakers,
	}
}

func (s *Service) ProcessPayment(data PaymentData) (PaymentRecord, bool) {
	for {
		var resData PaymentData
		var target PaymentTarget
		ok := false

		if s.breakers[TargetDefault].Allow() {
			ok, resData = forwardPayment("http://payment-processor-default:8080/payments", data, s.client)
			target = TargetDefault

			if ok {
				s.breakers[TargetDefault].MarkSuccess()
			} else {
				s.breakers[TargetDefault].MarkFailure()
			}
		}

		if !ok && s.breakers[TargetFallback].Allow() {
			ok, resData = forwardPayment("http://payment-processor-fallback:8080/payments", data, s.client)
			target = TargetFallback

			if ok {
				s.breakers[TargetFallback].MarkSuccess()
			} else {
				s.breakers[TargetFallback].MarkFailure()
			}
		}

		if ok {
			requestedAt, _ := time.Parse(time.RFC3339, resData.RequestedAt)
			return PaymentRecord{
				Timestamp: requestedAt,
				Amount:    resData.Amount,
				Target:    target,
			}, true
		}

		sleep := minNonZero(
			s.breakers[TargetDefault].RemainingOpen(),
			s.breakers[TargetFallback].RemainingOpen())

		if sleep == 0 {
			sleep = 10 * time.Millisecond
		}
		time.Sleep(sleep)
	}
}

func forwardPayment(url string, data PaymentData, client *http.Client) (bool, PaymentData) {
	data.RequestedAt = time.Now().UTC().Format(time.RFC3339)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		log.Printf("Erro ao serializar JSON: %v", err)
		return false, data
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		log.Printf("Erro ao criar requisição para %s: %v", url, err)
		return false, data
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Erro ao enviar para %s: %v", url, err)
		return false, data
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Serviço %s retornou status %d", url, resp.StatusCode)
		return false, data
	}

	return true, data
}

func minNonZero(a, b time.Duration) time.Duration {
	if a <= 0 {
		a = time.Duration(math.MaxInt64)
	}
	if b <= 0 {
		b = time.Duration(math.MaxInt64)
	}
	min := a
	if b < a {
		min = b
	}
	if min == time.Duration(math.MaxInt64) {
		return 0
	}
	return min
}
