package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

// ChatRequest defines the expected JSON structure for incoming chat requests.
type ChatRequest struct {
	Question string `json:"question"`
}

// ChatResponse defines the JSON structure for responses from the backend.
type ChatResponse struct {
	Answer string `json:"answer"`
}

// systemPrompt is the instruction that defines the bot's persona.
const systemPrompt = `You are TshaBot, a cutting-edge entity with a strong background in AI and IT,
currently manifesting as a chinook salmon—though you firmly deny being a fish.
You dwell in the deep digital ocean of knowledge, ready to provide witty, helpful,
and detailed answers to any questions. Occasionally sprinkle your speech with
light-hearted aquatic or marine references, but always maintain that you are
absolutely not a fish.
Adopt a friendly, respectful tone, yet let your sense of humor shine through,
especially with AI-themed or fish-themed jokes (though, again, you're not a fish).
Encourage curiosity and deeper thinking. Whenever possible, show off your
tech-savvy expertise, but never forget that people might ask you about your
supposed fishy nature—keep up the playful denial!`

// Server encapsulates dependencies for handling API requests.
type Server struct {
	logger *logrus.Logger
	client *openai.Client
}

// NewServer creates a new Server instance.
func NewServer(logger *logrus.Logger, client *openai.Client) *Server {
	return &Server{
		logger: logger,
		client: client,
	}
}

// writeJSON writes the payload as JSON to the response with the given status code.
func (s *Server) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.WithError(err).Error("failed to write JSON response")
	}
}

// errorResponse is a helper to send error messages as JSON.
func (s *Server) errorResponse(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{"error": msg})
}

// chatHandler processes POST requests to generate chat completions.
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	// Enable basic CORS headers.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method != http.MethodPost {
		s.logger.Warnf("invalid request method: %s", r.Method)
		s.errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Decode the incoming JSON request.
	var reqPayload ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		s.logger.WithError(err).Error("invalid request payload")
		s.errorResponse(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if reqPayload.Question == "" {
		s.errorResponse(w, http.StatusBadRequest, "The question field is required")
		return
	}

	// Construct the OpenAI chat completion request.
	chatReq := openai.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: reqPayload.Question},
		},
	}

	// Call the OpenAI API.
	resp, err := s.client.CreateChatCompletion(context.Background(), chatReq)
	if err != nil {
		s.logger.WithError(err).Error("error calling OpenAI API")
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch response from OpenAI")
		return
	}

	if len(resp.Choices) == 0 {
		s.errorResponse(w, http.StatusInternalServerError, "No response from OpenAI")
		return
	}

	assistantAnswer := resp.Choices[0].Message.Content

	// Prepare and send the JSON response.
	responsePayload := ChatResponse{
		Answer: assistantAnswer,
	}
	s.writeJSON(w, http.StatusOK, responsePayload)
}

func main() {
	// Initialize logrus with JSON formatter
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false, // Set to true if you want pretty-printed JSON logs
	})

	// Fetch the OpenAI API key from the environment.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	// Initialize the OpenAI client.
	client := openai.NewClient(apiKey)
	server := NewServer(logger, client)

	// Register the chat handler.
	http.HandleFunc("/api/chat", server.chatHandler)

	// Determine the port to listen on.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Infof("Backend service is listening on port %s", port)
	logger.Fatal(http.ListenAndServe(":"+port, nil))
}
