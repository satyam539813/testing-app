package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// --- CONFIGURATION & GLOBAL HTTP CLIENT ---

var openRouterAPIKey string
var httpClient *http.Client

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env file not found, using system environment variables")
	}

	openRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
	if openRouterAPIKey == "" {
		log.Fatal("❌ OPENROUTER_API_KEY is not set. Please set it in your .env file or environment.")
	}

	// Create a single, reusable HTTP client with timeouts for better performance and resilience.
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// --- DATA STRUCTURES ---

type TravelPlanRequest struct {
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Budget      float64 `json:"budget"`
}

type DayPlan struct {
	Day        int                `json:"day"`
	Activities string             `json:"activities"`
	Expenses   map[string]float64 `json:"expenses"`
}

type TravelPlanResponse struct {
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Budget      float64   `json:"budget"`
	Days        []DayPlan `json:"days"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Struct for streaming requests
type OpenRouterStreamRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

// Struct for OpenRouter message format
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Struct for non-streaming OpenRouter response
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// --- MAIN FUNCTION ---

func main() {
	// Keep the original non-streaming route for comparison or fallback
	http.HandleFunc("/api/route", handleRoute)
	// Add the new, faster streaming route
	http.HandleFunc("/api/route-stream", handleRouteStream)
	http.HandleFunc("/health", handleHealth)

	fmt.Println("✅ Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Could not start server: %v", err)
	}
}

// --- HTTP HANDLERS ---

// Health check handler
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Non-streaming handler
func handleRoute(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var reqData TravelPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		sendJSONError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if reqData.Source == "" || reqData.Destination == "" || reqData.Budget <= 0 {
		sendJSONError(w, "Source, destination, and a positive budget are required", http.StatusBadRequest)
		return
	}

	prompt := createPrompt(reqData)
	responseText, err := callOpenRouter(prompt, "You are a professional travel planner AI. You ONLY respond with valid JSON.")
	if err != nil {
		log.Printf("🚨 OpenRouter API error: %v", err)
		sendJSONError(w, "Failed to get response from AI: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clean the response text
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Parse and validate JSON response
	var travelPlan TravelPlanResponse
	if err := json.Unmarshal([]byte(responseText), &travelPlan); err != nil {
		log.Printf("🚨 Failed to parse AI response as JSON: %v\nResponse: %s", err, responseText)
		sendJSONError(w, "AI returned invalid JSON format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(travelPlan)
}

// Streaming Handler
func handleRouteStream(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. Decode and Validate Request Body
	var reqData TravelPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		sendJSONError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if reqData.Source == "" || reqData.Destination == "" || reqData.Budget <= 0 {
		sendJSONError(w, "Source, destination, and a positive budget are required", http.StatusBadRequest)
		return
	}

	// 2. Set Headers for Server-Sent Events (SSE)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	enableCORS(w) // Ensure CORS headers are set for streaming too
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		sendJSONError(w, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	// 3. Create the AI Prompt
	prompt := createPrompt(reqData)

	// 4. Call the Streaming AI Function
	stream, err := callOpenRouterStream(prompt, "You are a professional travel planner AI. You ONLY respond with valid JSON.")
	if err != nil {
		log.Printf("🚨 OpenRouter Stream API error: %v", err)
		// Send error as SSE event
		fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer stream.Close()

	// 5. Proxy the stream to the client
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			// Forward the data chunk directly to the client
			fmt.Fprintf(w, "%s\n\n", line)
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("🚨 Error reading stream from OpenRouter: %v", err)
	}
}

// --- HELPER FUNCTIONS ---

func createPrompt(reqData TravelPlanRequest) string {
	return fmt.Sprintf(
		"Plan a detailed travel itinerary from %s to %s with a strict budget of INR %.2f. "+
			"The plan must be day-wise, including specific activities and estimated expenses for each day. "+
			"Your entire response must be a single, valid JSON object following this exact structure: "+
			"{\"source\": \"%s\", \"destination\": \"%s\", \"budget\": %.2f, \"days\": [{\"day\": 1, \"activities\": \"...\", \"expenses\": {\"category\": amount}}]}. "+
			"DO NOT include any text, explanations, markdown code blocks, or formatting outside of this JSON object. "+
			"Respond ONLY with the raw JSON.",
		reqData.Source, reqData.Destination, reqData.Budget,
		reqData.Source, reqData.Destination, reqData.Budget,
	)
}

// Non-streaming OpenRouter call
func callOpenRouter(prompt, systemMessage string) (string, error) {
	apiURL := "https://openrouter.ai/api/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "anthropic/claude-3-haiku",
		"messages": []OpenRouterMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+openRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "Go Travel Planner")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-200 status code (%d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal OpenRouter response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenRouter")
	}

	return openRouterResp.Choices[0].Message.Content, nil
}

// Streaming OpenRouter call
func callOpenRouterStream(prompt, systemMessage string) (io.ReadCloser, error) {
	apiURL := "https://openrouter.ai/api/v1/chat/completions"

	payload := OpenRouterStreamRequest{
		Model: "anthropic/claude-3-haiku",
		Messages: []OpenRouterMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+openRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "Go Travel Planner")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("received non-200 status code (%d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// Send JSON error response
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// Enable CORS for cross-origin requests
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}