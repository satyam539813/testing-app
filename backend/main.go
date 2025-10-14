package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings" // Import the strings package

	"github.com/joho/godotenv"
)

// --- CONFIGURATION ---

var openRouterAPIKey string

// The init function runs once when the package is initialized.
// It's used here to load environment variables from a .env file.
func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env file not found, using system environment variables")
	}

	openRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
	if openRouterAPIKey == "" {
		log.Fatal("❌ OPENROUTER_API_KEY is not set. Please set it in your .env file or environment.")
	}
}

// --- DATA STRUCTURES ---
// Using structs for API contracts makes the code type-safe and self-documenting.

// TravelPlanRequest is the expected JSON body for the /api/route endpoint.
type TravelPlanRequest struct {
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Budget      float64 `json:"budget"`
}

// DayPlan defines the structure of a single day in the travel itinerary.
type DayPlan struct {
	Day        int                    `json:"day"`
	Activities string                 `json:"activities"`
	Expenses   map[string]interface{} `json:"expenses"`
}

// TravelPlanResponse is the JSON response for a successfully planned trip.
type TravelPlanResponse struct {
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Budget      float64   `json:"budget"`
	Days        []DayPlan `json:"days"`
}

// OpenRouter Structures for making requests to the OpenRouter API.
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
}

// OpenRouterResponse defines the expected structure of the JSON response from OpenRouter.
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// --- MAIN FUNCTION ---

func main() {
	// Register handlers for the API endpoints.
	http.HandleFunc("/api/route", handleRoute)

	// Start the HTTP server.
	fmt.Println("✅ Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Could not start server: %v", err)
	}
}

// --- HTTP HANDLERS ---

// handleRoute processes requests for travel planning.
func handleRoute(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all responses and handle preflight OPTIONS request.
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Ensure the request method is POST.
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request into our struct.
	var reqData TravelPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create a clear and concise prompt for the AI model.
	prompt := fmt.Sprintf(
		"Plan a detailed travel itinerary from %s to %s with a strict budget of INR %.2f. "+
			"The plan must be day-wise, including specific activities and estimated expenses for each day. "+
			"Your entire response must be a single, valid JSON object following this exact structure: "+
			"{\"source\": \"%s\", \"destination\": \"%s\", \"budget\": %.2f, \"days\": [{\"day\": 1, \"activities\": \"...\", \"expenses\": ...}]}. "+
			"Do not include any text, explanations, or markdown formatting outside of this JSON object.",
		reqData.Source, reqData.Destination, reqData.Budget,
		reqData.Source, reqData.Destination, reqData.Budget,
	)

	// Call the AI service to get the travel plan.
	aiResponseContent, err := callOpenRouter(prompt, "You are a professional travel planner AI.")
	if err != nil {
		http.Error(w, "Failed to get response from AI: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// **FIX:** Clean the AI response. Some models wrap JSON in markdown code blocks.
	// This finds the first '{' and the last '}' to extract the raw JSON string.
	startIndex := strings.Index(aiResponseContent, "{")
	endIndex := strings.LastIndex(aiResponseContent, "}")

	if startIndex == -1 || endIndex == -1 || endIndex < startIndex {
		log.Printf("🚨 AI response does not contain a valid JSON object. Raw Response: %s", aiResponseContent)
		http.Error(w, "AI returned a response that could not be understood as a travel plan.", http.StatusInternalServerError)
		return
	}

	cleanedJSON := aiResponseContent[startIndex : endIndex+1]

	// The AI response is expected to be a JSON string. Unmarshal it into our response struct.
	var travelPlan TravelPlanResponse
	if err := json.Unmarshal([]byte(cleanedJSON), &travelPlan); err != nil {
		log.Printf("🚨 Failed to parse cleaned AI JSON response. Error: %v\nRaw Response: %s", err, aiResponseContent)
		http.Error(w, "AI returned an invalid format. Please try again.", http.StatusInternalServerError)
		return
	}

	// Send the structured JSON response back to the client.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(travelPlan); err != nil {
		// This error is less likely but good to handle.
		log.Printf("🚨 Failed to encode final response: %v", err)
	}
}

// --- HELPER FUNCTIONS ---

// callOpenRouter sends a prompt to the OpenRouter API and returns the text response.
func callOpenRouter(prompt, systemMessage string) (string, error) {
	apiURL := "https://openrouter.ai/api/v1/chat/completions"

	// Construct the request payload using our defined structs.
	payload := OpenRouterRequest{
		Model: "gpt-4o-mini",
		Messages: []OpenRouterMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: prompt},
		},
	}

	// Marshal the payload into JSON.
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create a new HTTP request.
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	// Set necessary headers.
	req.Header.Set("Authorization", "Bearer "+openRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost:8080") // Recommended by OpenRouter
	req.Header.Set("X-Title", "Go Travel Planner")           // Recommended by OpenRouter

	// Send the request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status codes.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status code (%d): %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response into our struct.
	var result OpenRouterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal OpenRouter response: %w", err)
	}

	// Check for API-level errors in the response body.
	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("OpenRouter API error: %s", result.Error.Message)
	}

	// Ensure there is at least one choice and return the content.
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("received an empty response from AI")
	}

	return result.Choices[0].Message.Content, nil
}

// enableCORS sets the necessary headers to allow cross-origin requests.
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

