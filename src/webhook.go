package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type webhookPayload struct {
	Event      string `json:"event"`
	Status     string `json:"status"`
	Name       string `json:"name"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
	Error      string `json:"error"`
}

func sendWebhook(event, status, name string, durationMs int64, errMsg string) {
	url := os.Getenv("WEBHOOK_URL")
	if url == "" {
		return
	}
	payload := webhookPayload{
		Event:      event,
		Status:     status,
		Name:       name,
		DurationMs: durationMs,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Error:      errMsg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Warning: webhook marshal failed: %v\n", err)
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Warning: webhook delivery failed: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
}
