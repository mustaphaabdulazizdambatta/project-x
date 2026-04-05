package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

const (
	stealthAIURL     = "http://127.0.0.1:5001/analyze"
	stealthAITimeout = 1200 * time.Millisecond
)

type StealthAIResult struct {
	Score float64 `json:"score"`
}

// stealthAIClient is a shared HTTP client with keep-alive connection pooling.
// Reusing one client avoids the overhead of a new TCP handshake per request.
var stealthAIClient = &http.Client{
	Timeout: stealthAITimeout,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// stealthAIAvailable tracks whether the local service responded successfully
// at least once, to skip the service attempt when it is known to be down.
var stealthAIAvailable int32 = 1

// StealthAIHealthCheck pings the service once at startup. If it fails the
// per-request call will go straight to the CLI fallback.
func StealthAIHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]string{"packet": "healthcheck"})
	req, _ := http.NewRequestWithContext(ctx, "POST", stealthAIURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := stealthAIClient.Do(req)
	if err != nil {
		log.Warning("StealthAI service not reachable — using CLI fallback (%v)", err)
		atomic.StoreInt32(&stealthAIAvailable, 0)
		return
	}
	resp.Body.Close()
	log.Success("StealthAI service is up")
	atomic.StoreInt32(&stealthAIAvailable, 1)
}

// AnalyzeTrafficWithStealthAI scores a traffic packet. It first tries the
// persistent HTTP service; if unavailable it falls back to the Python CLI.
func AnalyzeTrafficWithStealthAI(packet string) (float64, error) {
	if atomic.LoadInt32(&stealthAIAvailable) == 1 {
		score, ok := queryStealthAIService(packet)
		if ok {
			return score, nil
		}
		// Mark service as down so subsequent requests skip straight to CLI.
		atomic.StoreInt32(&stealthAIAvailable, 0)
		log.Warning("StealthAI service unreachable — switching to CLI fallback")
	}

	return queryStealthAICLI(packet)
}

func queryStealthAIService(packet string) (float64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), stealthAITimeout)
	defer cancel()

	body, _ := json.Marshal(map[string]string{"packet": packet})
	req, _ := http.NewRequestWithContext(ctx, "POST", stealthAIURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := stealthAIClient.Do(req)
	if err != nil {
		log.Debug("StealthAI service error: %v", err)
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	var result StealthAIResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Debug("StealthAI decode error: %v", err)
		return 0, false
	}

	log.Debug("StealthAI score (service): %.3f", result.Score)
	return result.Score, true
}

func queryStealthAICLI(packet string) (float64, error) {
	cmd := exec.Command("python3", "ai/stealth_ai.py", packet)
	out, err := cmd.Output()
	if err != nil {
		log.Error("StealthAI CLI error: %v", err)
		return 0.0, err
	}

	var result StealthAIResult
	if err := json.Unmarshal(out, &result); err != nil {
		log.Error("StealthAI CLI decode error: %v", err)
		return 0.0, err
	}

	log.Debug("StealthAI score (CLI): %.3f", result.Score)
	return result.Score, nil
}
