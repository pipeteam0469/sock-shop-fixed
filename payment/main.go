package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// PaymentRequest represents an incoming payment request
type PaymentRequest struct {
	CardNumber string `json:"cardNumber"`
	CardHolder string `json:"cardHolder"`
	ExpiryDate string `json:"expiryDate"`
	CVV        string `json:"cvv"`
	Amount     float64 `json:"amount"`
}

// PaymentResponse represents the payment result
type PaymentResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
	CardLast4   string  `json:"cardLast4"`
	Timestamp   string  `json:"timestamp"`
}

// In-memory payment store (simplified for demo)
var (
	payments  []PaymentResponse
	paymentID int
)

func init() {
	payments = make([]PaymentResponse, 0)
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// Health check
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "payment"})
}

// Validate card number (Luhn algorithm)
func isValidCard(cardNumber string) bool {
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")
	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	sum := 0
	alternate := false
	for i := len(cardNumber) - 1; i >= 0; i-- {
		n := int(cardNumber[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}

// Process payment
func processPayment(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.CardNumber == "" || req.CardHolder == "" || req.ExpiryDate == "" || req.CVV == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "All payment fields are required"})
		return
	}

	// Validate card number using Luhn algorithm
	if !isValidCard(req.CardNumber) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid card number"})
		return
	}

	// Validate CVV length
	if len(req.CVV) < 3 || len(req.CVV) > 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid CVV"})
		return
	}

	// Create payment response (mock processing)
	paymentID++
	last4 := "****"
	if len(req.CardNumber) >= 4 {
		last4 = req.CardNumber[len(req.CardNumber)-4:]
	}

	resp := PaymentResponse{
		ID:        fmt.Sprintf("PAY-%06d", paymentID),
		Status:    "approved",
		Amount:    req.Amount,
		CardLast4: last4,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	payments = append(payments, resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Get payment by ID
func getPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, p := range payments {
		if p.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Payment not found"})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/payment", processPayment).Methods("POST")
	r.HandleFunc("/payment/{id}", getPayment).Methods("GET")

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Payment service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
