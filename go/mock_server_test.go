package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// sampleDescriptor is a two-operation descriptor in the exact shape the API
// serves: an "# " title, a declared operation count, and one line per
// operation. The middle dot is part of the format.
const sampleDescriptor = `# Example Product

> Generated for composition cmp_example from its capability closure.

## Integration

- Auth: bearer (secret via HYPERSCALE_API_KEY)
- Error envelope: { error: { code, message }, requestId }

## Operations (2)

- GET /v1/accounts · Account list (account_list; idempotency none)
- POST /v1/accounts · Account create (account_create; idempotency required)

## Golden paths (1)

- Open an account (open_account):
  1. AccountCreate (account_create)
`

// receivedRequest is what one starter request looked like on the wire.
type receivedRequest struct {
	Path          string
	Authorization string
	Environment   string
	Accept        string
}

// mockServer stands in for the Product API on 127.0.0.1, so the suite proves
// the whole path -- headers out, document back, parsed result -- without a key
// and without the network.
type mockServer struct {
	*httptest.Server

	mutex    sync.Mutex
	received []receivedRequest
}

func startMockServer(apiKey string, document string) *mockServer {
	mock := &mockServer{}
	mock.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mock.record(receivedRequest{
			Path:          request.URL.Path,
			Authorization: request.Header.Get("Authorization"),
			Environment:   request.Header.Get("X-Hyperscale-Environment"),
			Accept:        request.Header.Get("Accept"),
		})

		if request.Header.Get("Authorization") != "Bearer "+apiKey {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_credentials",
					"message": "Bearer token is not a valid API key in this environment.",
				},
				"requestId": "req_mock",
			})
			return
		}

		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = writer.Write([]byte(document))
	}))
	return mock
}

// httptest serves each request on its own goroutine, so the log is guarded.
func (mock *mockServer) record(request receivedRequest) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	mock.received = append(mock.received, request)
}

func (mock *mockServer) last() receivedRequest {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	if len(mock.received) == 0 {
		return receivedRequest{}
	}
	return mock.received[len(mock.received)-1]
}
