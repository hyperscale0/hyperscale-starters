package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// sampleOperations is a two-item page in the shape GET /v1/operations serves,
// with a cursor that says another page follows.
const sampleOperations = `{"items":[` +
	`{"operationId":"ops_sandbox_example01","tenantId":"ten_sandbox_example01",` +
	`"name":"customer.create","status":"succeeded","createdAt":"2026-09-25T09:00:00.000Z"},` +
	`{"operationId":"ops_sandbox_example02","tenantId":"ten_sandbox_example01",` +
	`"name":"account.create","status":"failed","errorCode":"validation_failed",` +
	`"errorMessage":"currency is required","createdAt":"2026-09-25T09:01:00.000Z"}` +
	`],"nextCursor":"cursor_example","statusCounts":{"failed":1,"succeeded":1}}`

// receivedRequest is what one starter request looked like on the wire.
type receivedRequest struct {
	URL           string
	Authorization string
	Environment   string
	Accept        string
}

// mockServer stands in for the Product API on 127.0.0.1, so the suite proves
// the whole path -- headers out, JSON back, parsed result -- without a key
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
			URL:           request.URL.RequestURI(),
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

		writer.Header().Set("Content-Type", "application/json")
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
