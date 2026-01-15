package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT secret key - in production, use environment variable
var jwtSecret = []byte("your-secret-key-change-this-in-production")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

// HandleRegister handles user registration
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Username and password are required",
		})
		return
	}

	if len(req.Password) < 6 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Password must be at least 6 characters",
		})
		return
	}

	// Check if user already exists
	exists, err := UserExists(req.Username)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Server error",
		})
		return
	}

	if exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Username already exists",
		})
		return
	}

	// Create user
	if err := CreateUser(req.Username, req.Password); err != nil {
		log.Printf("Error creating user: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Failed to create user",
		})
		return
	}

	// Generate JWT token
	token, err := GenerateToken(req.Username)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "User registered successfully",
		Token:   token,
	})
}

// HandleLogin handles user login
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	// Validate credentials
	valid, err := ValidateUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Error validating user: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Server error",
		})
		return
	}

	if !valid {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	token, err := GenerateToken(req.Username)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
	})
}

// GenerateToken creates a JWT token for a user
func GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns the username
func ValidateToken(tokenString string) (string, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	return claims.Username, nil
}

// AuthMiddleware is a middleware to protect WebSocket connections
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from query parameter (for WebSocket)
		token := r.URL.Query().Get("token")
		if token == "" {
			// Also check Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		username, err := ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Set username in query parameter for WebSocket handler
		q := r.URL.Query()
		q.Set("username", username)
		r.URL.RawQuery = q.Encode()

		next.ServeHTTP(w, r)
	}
}
