package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client handles communication with a local Ollama server
type Client struct {
	url   string
	model string
}

// New creates a new Ollama client
func New(url, model string) *Client {
	if url == "" {
		url = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2"
	}
	return &Client{url: url, model: model}
}

// AnalyzeRequest is the request to analyze logs
type AnalyzeRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// AnalyzeResponse is the response from Ollama
type AnalyzeResponse struct {
	Response string `json:"response"`
}

// AnalyzeLogs sends logs to Ollama and asks it to identify the error lines
// Returns a list of line numbers (0-indexed) that contain errors
func (c *Client) AnalyzeLogs(ctx context.Context, logs []string) ([]int, error) {
	if len(logs) == 0 {
		return nil, nil
	}

	// Build numbered log lines for context
	var numberedLogs strings.Builder
	for i, line := range logs {
		fmt.Fprintf(&numberedLogs, "%d: %s\n", i, line)
	}

	prompt := fmt.Sprintf(`You are analyzing application logs to find error messages. Below are numbered log lines.

Identify which line numbers contain the actual error or failure message (not warnings, not info messages).
Return ONLY a comma-separated list of line numbers, nothing else. If no clear errors, return "none".

Example response: "5,6,7" or "12" or "none"

Logs:
%s

Error line numbers:`, numberedLogs.String())

	req := AnalyzeRequest{
		Prompt: prompt,
		Model:  c.model,
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Use a timeout context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result AnalyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Parse the response to extract line numbers
	return parseLineNumbers(result.Response, len(logs))
}

// parseLineNumbers extracts line numbers from the LLM response
func parseLineNumbers(response string, maxLines int) ([]int, error) {
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "none" || response == "" {
		return nil, nil
	}

	var lineNumbers []int
	parts := strings.Split(response, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			if num >= 0 && num < maxLines {
				lineNumbers = append(lineNumbers, num)
			}
		}
	}

	return lineNumbers, nil
}
