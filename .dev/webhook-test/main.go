package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/health", healthHandler)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting webhook test server on %s", addr)
	log.Printf("Webhook endpoint: http://localhost%s/webhook", addr)
	log.Printf("Health endpoint: http://localhost%s/health", addr)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	log.Println("========================================")
	log.Printf("Webhook received at: %s", timestamp)
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Remote Address: %s", r.RemoteAddr)
	
	log.Println("\nHeaders:")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	log.Println("\nBody (raw):")
	log.Printf("%s", string(body))
	
	if len(body) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			log.Println("\nBody (formatted JSON):")
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			log.Printf("%s", string(prettyJSON))
		}
	}
	
	log.Println("========================================\n")
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "received",
		"timestamp": timestamp,
		"message":   "Webhook processed successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"status": "healthy",
		"service": "webhook-test",
	}
	json.NewEncoder(w).Encode(response)
}
