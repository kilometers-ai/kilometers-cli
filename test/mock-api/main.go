package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Mock API server for testing Kilometers CLI plugin architecture
// This simulates the Kilometers API endpoints for local testing

type UserFeaturesResponse struct {
	Tier      string   `json:"tier"`
	Features  []string `json:"features"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

var mockSubscriptions = map[string]UserFeaturesResponse{
	"km_free_123456": {
		Tier:     "free",
		Features: []string{"basic_monitoring", "console_logging"},
	},
	"km_pro_789012": {
		Tier: "pro",
		Features: []string{
			"basic_monitoring",
			"console_logging",
			"api_logging",
			"advanced_filters",
			"ml_analytics",
		},
	},
	"km_ent_345678": {
		Tier: "enterprise",
		Features: []string{
			"basic_monitoring",
			"console_logging",
			"api_logging",
			"advanced_filters",
			"ml_analytics",
			"compliance_reporting",
			"team_collaboration",
		},
	},
	// Special key for testing downgrades
	"km_downgrade_test": {
		Tier:     "pro", // Will change to free after first request
		Features: []string{"basic_monitoring", "console_logging", "api_logging"},
	},
}

var requestCount = make(map[string]int)

func main() {
	fmt.Println("🚀 Mock Kilometers API Server")
	fmt.Println("=============================")
	fmt.Println("Listening on http://localhost:5194")
	fmt.Println()
	fmt.Println("Test API Keys:")
	fmt.Println("  - km_free_123456     (Free tier)")
	fmt.Println("  - km_pro_789012      (Pro tier)")
	fmt.Println("  - km_ent_345678      (Enterprise tier)")
	fmt.Println("  - km_downgrade_test  (Simulates downgrade)")
	fmt.Println()

	http.HandleFunc("/api/user/features", handleUserFeatures)
	http.HandleFunc("/api/events/batch", handleEventsBatch)
	http.HandleFunc("/health", handleHealth)

	log.Fatal(http.ListenAndServe(":5194", nil))
}

func handleUserFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	log.Printf("[API] GET /api/user/features - API Key: %s", maskKey(apiKey))

	// Handle downgrade simulation
	if apiKey == "km_downgrade_test" {
		requestCount[apiKey]++
		if requestCount[apiKey] > 1 {
			// After first request, downgrade to free
			response := UserFeaturesResponse{
				Tier:     "free",
				Features: []string{"basic_monitoring", "console_logging"},
			}
			log.Printf("[API] Simulating downgrade for key: %s", maskKey(apiKey))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Look up subscription
	subscription, exists := mockSubscriptions[apiKey]
	if !exists {
		// Unknown key - return free tier
		subscription = UserFeaturesResponse{
			Tier:     "free",
			Features: []string{"basic_monitoring"},
		}
		log.Printf("[API] Unknown key, returning free tier: %s", maskKey(apiKey))
	}

	// Add expiration for non-free tiers
	if subscription.Tier != "free" {
		expires := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
		subscription.ExpiresAt = &expires
	}

	log.Printf("[API] Returning tier: %s with %d features", subscription.Tier, len(subscription.Features))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

func handleEventsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Decode the batch request
	var batch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	events, ok := batch["events"].([]interface{})
	if !ok {
		events = []interface{}{}
	}

	log.Printf("[API] POST /api/events/batch - API Key: %s, Events: %d", maskKey(apiKey), len(events))

	// Check if the user has API logging feature
	subscription, exists := mockSubscriptions[apiKey]
	if !exists || !contains(subscription.Features, "api_logging") {
		log.Printf("[API] Rejecting batch - no api_logging feature for key: %s", maskKey(apiKey))
		http.Error(w, "API logging not available in your subscription", http.StatusForbidden)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Received %d events", len(events)),
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func maskKey(key string) string {
	if len(key) > 10 {
		return key[:6] + "..." + key[len(key)-4:]
	}
	return key
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
