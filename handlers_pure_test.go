package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFibonacciHTTPContract(t *testing.T) {
	for _, row := range []struct {
		name, method, body, result string
		n, status                  int
	}{
		{"zero", http.MethodPost, `{"n":0}`, "0", 0, http.StatusOK},
		{"one", http.MethodPost, `{"n":1}`, "1", 1, http.StatusOK},
		{"forty-two", http.MethodPost, `{"n":42}`, "267914296", 42, http.StatusOK},
		{"method", http.MethodGet, "", "", 0, http.StatusMethodNotAllowed},
		{"malformed", http.MethodPost, `{`, "", 0, http.StatusBadRequest},
		{"negative", http.MethodPost, `{"n":-1}`, "", 0, http.StatusBadRequest},
		{"over-limit", http.MethodPost, `{"n":10001}`, "", 0, http.StatusBadRequest},
	} {
		t.Run(row.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handleFib(response, httptest.NewRequest(row.method, "/api/fib", strings.NewReader(row.body)))
			if response.Code != row.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, row.status, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatal("response must be JSON")
			}
			if row.status != http.StatusOK {
				var failure errorResponse
				if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil || failure.Error == "" {
					t.Fatalf("missing JSON error: %s", response.Body.String())
				}
				return
			}
			assertFibonacciResponse(t, response.Body.Bytes(), row.result, row.n)
		})
	}
}

func assertFibonacciResponse(t *testing.T, body []byte, result string, n int) {
	t.Helper()
	var actual fibResponse
	if err := json.Unmarshal(body, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.N != n || actual.Result != result || actual.Digits != len(result) {
		t.Fatalf("result = %+v, want %s", actual, result)
	}
	if _, err := time.ParseDuration(actual.Duration); err != nil {
		t.Fatalf("invalid duration %q: %v", actual.Duration, err)
	}
}

func TestReadinessTransitions(t *testing.T) {
	defer ready.Store(false)
	for _, row := range []struct {
		ready bool
		code  int
	}{
		{false, http.StatusServiceUnavailable},
		{true, http.StatusOK},
		{false, http.StatusServiceUnavailable},
	} {
		ready.Store(row.ready)
		response := httptest.NewRecorder()
		handleReady(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if response.Code != row.code {
			t.Fatalf("ready=%t: status=%d, want %d", row.ready, response.Code, row.code)
		}
	}
}

func TestIndexAndHealthContracts(t *testing.T) {
	for _, row := range []struct {
		path string
		code int
	}{
		{"/", http.StatusOK},
		{"/unknown", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handleIndex(response, httptest.NewRequest(http.MethodGet, row.path, nil))
		if response.Code != row.code {
			t.Fatalf("%s: status=%d, want %d", row.path, response.Code, row.code)
		}
	}
	response := httptest.NewRecorder()
	handleHealth(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	var health map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || health["status"] != "ok" || health["version"] != Version {
		t.Fatalf("unexpected health: %d %v", response.Code, health)
	}
}
