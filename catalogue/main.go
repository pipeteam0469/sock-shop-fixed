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

// Product represents a sock product in the catalogue
type Product struct {
	ID          string  `json:"_id" bson:"_id"`
	Name        string  `json:"name" bson:"name"`
	Description string  `json:"description" bson:"description"`
	Price       float64 `json:"price" bson:"price"`
	Count       int     `json:"count" bson:"count"`
	ImageURL    string  `json:"imageUrl" bson:"imageUrl"`
	Tag         string  `json:"tag" bson:"tag"`
}

// In-memory product store (replaces MongoDB for simplified hardened version)
var products []Product

func init() {
	// Seed with sample sock products
	products = []Product{
		{ID: "6d62d909-f957-430e-8689-b5129c0bb75e", Name: "Red sock", Description: "A red sock", Price: 12.34, Count: 100, ImageURL: "/img/0.jpg", Tag: "Red"},
		{ID: "a0a4f044-b040-410d-8ead-4de0446aec7e", Name: "Black sock", Description: "A black sock", Price: 12.34, Count: 200, ImageURL: "/img/1.jpg", Tag: "Black"},
		{ID: "808a2de1-1aaa-4c25-a9b9-6612e8f29a38", Name: "Blue sock", Description: "A blue sock", Price: 12.34, Count: 150, ImageURL: "/img/2.jpg", Tag: "Blue"},
		{ID: "510a0d7e-8e48-443b-be71-7569c67b6c91", Name: "Green sock", Description: "A green sock", Price: 12.34, Count: 80, ImageURL: "/img/3.jpg", Tag: "Green"},
		{ID: "zzz4f044-b040-410d-8ead-4de0446aec7e", Name: "Yellow sock", Description: "A yellow sock", Price: 12.34, Count: 50, ImageURL: "/img/4.jpg", Tag: "Yellow"},
		{ID: "6652545d-b765-482b-9c41-7c49377a0899", Name: "Purple sock", Description: "A purple sock", Price: 12.34, Count: 30, ImageURL: "/img/5.jpg", Tag: "Purple"},
	}
}

// Middleware for logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "catalogue"})
}

// Get all products
func getAllProducts(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	tag := r.URL.Query().Get("tag")
	size := r.URL.Query().Get("size")
	
	var result []Product
	for _, p := range products {
		if tag != "" && !strings.Contains(strings.ToLower(p.Tag), strings.ToLower(tag)) {
			continue
		}
		_ = size // size filtering would be implemented with a real DB
		result = append(result, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Get single product by ID
func getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, p := range products {
		if p.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Product not found"})
}

// Search products
func searchProducts(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	var result []Product
	for _, p := range products {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(name)) {
			result = append(result, p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
	r.HandleFunc("/catalogue", getAllProducts).Methods("GET")
	r.HandleFunc("/catalogue/{id}", getProduct).Methods("GET")
	r.HandleFunc("/catalogue", searchProducts).Methods("GET")

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Catalogue service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
