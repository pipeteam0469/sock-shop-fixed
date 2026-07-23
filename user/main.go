package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// User represents a user in the system
type User struct {
	ID        string `json:"_id" bson:"_id"`
	Username  string `json:"username" bson:"username"`
	Password  string `json:"-" bson:"password"` // Never expose password in JSON
	FirstName string `json:"firstName" bson:"firstName"`
	LastName  string `json:"lastName" bson:"lastName"`
	Email     string `json:"email" bson:"email"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
}

// In-memory user store (simplified for hardened demo)
var (
	users   []User
	usersMu sync.RWMutex
	// Simple HMAC secret for password hashing (in production, use bcrypt)
	secretKey = []byte("sock-shop-hardened-secret-key-change-in-production")
)

func init() {
	// Seed with a default admin user for testing
	users = []User{
		{
			ID:        "user-001",
			Username:  "admin",
			Password:  hashPassword("admin123"),
			FirstName: "Admin",
			LastName:  "User",
			Email:     "admin@sockshop.example.com",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
	}
}

// Hash password using HMAC-SHA256 (simplified for demo; use bcrypt in production)
func hashPassword(password string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
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
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "user"})
}

// Register a new user
func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" || req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username, password, and email are required"})
		return
	}

	usersMu.RLock()
	for _, u := range users {
		if u.Username == req.Username {
			usersMu.RUnlock()
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username already exists"})
			return
		}
	}
	usersMu.RUnlock()

	newUser := User{
		ID:        fmt.Sprintf("user-%d", len(users)+1),
		Username:  req.Username,
		Password:  hashPassword(req.Password),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	usersMu.Lock()
	users = append(users, newUser)
	usersMu.Unlock()

	// Return user without password
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"_id":       newUser.ID,
		"username":  newUser.Username,
		"firstName": newUser.FirstName,
		"lastName":  newUser.LastName,
		"email":     newUser.Email,
	})
}

// Login handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	hashedPassword := hashPassword(req.Password)

	usersMu.RLock()
	for _, u := range users {
		if u.Username == req.Username && u.Password == hashedPassword {
			usersMu.RUnlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"_id":       u.ID,
				"username":  u.Username,
				"firstName": u.FirstName,
				"lastName":  u.LastName,
				"email":     u.Email,
				"token":     "mock-jwt-token-" + u.ID, // Simplified token
			})
			return
		}
	}
	usersMu.RUnlock()

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
}

// Get user by ID
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	usersMu.RLock()
	for _, u := range users {
		if u.ID == id {
			usersMu.RUnlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"_id":       u.ID,
				"username":  u.Username,
				"firstName": u.FirstName,
				"lastName":  u.LastName,
				"email":     u.Email,
			})
			return
		}
	}
	usersMu.RUnlock()

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
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
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/users/{id}", getUserHandler).Methods("GET")

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("User service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
