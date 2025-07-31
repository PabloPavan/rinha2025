package main

import (
	"fmt"
	"log"
	"net/http"
)

func paymentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		fmt.Fprintln(w, "OK")
	} else {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/payments", paymentsHandler)
	log.Println("Listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}