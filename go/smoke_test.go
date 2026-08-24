package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const apiKey = "sk_sandbox_example"

// environment builds the lookup ReadConfig takes, so no test touches the
// process environment and the suite stays safe to run in parallel.
func environment(baseURL string, overrides map[string]string) func(string) string {
	values := map[string]string{
		"HYPERSCALE_API_KEY":  apiKey,
		"HYPERSCALE_BASE_URL": baseURL,
	}
	for name, value := range overrides {
		values[name] = value
	}
	return func(name string) string { return values[name] }
}

func TestSmokeCallSendsTheKeyAndTheEnvironment(t *testing.T) {
	mock := startMockServer(apiKey, sampleDescriptor)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	product, err := FetchProductDescriptor(config)
	if err != nil {
		t.Fatalf("FetchProductDescriptor: %v", err)
	}

	if product.Title != "Example Product" {
		t.Errorf("title = %q, want %q", product.Title, "Example Product")
	}
	want := []Operation{
		{Method: "GET", Path: "/v1/accounts", OperationID: "account_list"},
		{Method: "POST", Path: "/v1/accounts", OperationID: "account_create"},
	}
	if len(product.Operations) != len(want) {
		t.Fatalf("parsed %d operations, want %d", len(product.Operations), len(want))
	}
	for index, operation := range product.Operations {
		if operation != want[index] {
			t.Errorf("operation %d = %+v, want %+v", index, operation, want[index])
		}
	}

	request := mock.last()
	if request.Path != "/v1/llms.txt" {
		t.Errorf("path = %q, want %q", request.Path, "/v1/llms.txt")
	}
	if request.Authorization != "Bearer "+apiKey {
		t.Errorf("authorization = %q", request.Authorization)
	}
	if request.Environment != "sandbox" {
		t.Errorf("environment header = %q, want sandbox", request.Environment)
	}
	if request.Accept != "text/plain" {
		t.Errorf("accept = %q, want text/plain", request.Accept)
	}
}

func TestLivePlaneIsAddressedByTheHeader(t *testing.T) {
	mock := startMockServer(apiKey, sampleDescriptor)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, map[string]string{
		"HYPERSCALE_ENVIRONMENT": "live",
	}))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := FetchProductDescriptor(config); err != nil {
		t.Fatalf("FetchProductDescriptor: %v", err)
	}

	if got := mock.last().Environment; got != "live" {
		t.Errorf("environment header = %q, want live", got)
	}
}

func TestRefusedKeySurfacesTheErrorCode(t *testing.T) {
	mock := startMockServer(apiKey, sampleDescriptor)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, map[string]string{
		"HYPERSCALE_API_KEY": "sk_sandbox_wrong",
	}))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	_, err = FetchProductDescriptor(config)
	if err == nil {
		t.Fatal("a wrong key was accepted")
	}
	if !strings.Contains(err.Error(), "HTTP 401") ||
		!strings.Contains(err.Error(), "invalid_credentials") {
		t.Errorf("error = %q, want the status and the code", err.Error())
	}
}

func TestTrailingSlashDoesNotDoubleUpThePath(t *testing.T) {
	mock := startMockServer(apiKey, sampleDescriptor)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL+"///", nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := FetchProductDescriptor(config); err != nil {
		t.Fatalf("FetchProductDescriptor: %v", err)
	}

	if got := mock.last().Path; got != "/v1/llms.txt" {
		t.Errorf("path = %q, want /v1/llms.txt", got)
	}
}

func TestConfigDefaultsAndRefusals(t *testing.T) {
	onlyKey := func(name string) string {
		if name == "HYPERSCALE_API_KEY" {
			return apiKey
		}
		return ""
	}

	config, err := ReadConfig(onlyKey)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if config.Environment != "sandbox" {
		t.Errorf("environment = %q, want sandbox", config.Environment)
	}
	if config.BaseURL != DefaultBaseURL {
		t.Errorf("base URL = %q, want %q", config.BaseURL, DefaultBaseURL)
	}

	if _, err := ReadConfig(func(string) string { return "" }); err == nil {
		t.Error("a missing key was accepted")
	}
	if _, err := ReadConfig(environment("", map[string]string{
		"HYPERSCALE_ENVIRONMENT": "staging",
	})); err == nil {
		t.Error("an unknown plane was accepted")
	}
}

func TestCountThatDisagreesWithTheLinesIsRefused(t *testing.T) {
	short := strings.Replace(sampleDescriptor, "## Operations (2)", "## Operations (3)", 1)

	_, err := ParseProductDescriptor(short)
	if err == nil {
		t.Fatal("a descriptor with a wrong count was accepted")
	}
	if !strings.Contains(err.Error(), "declares 3 operations but 2 parsed") {
		t.Errorf("error = %q", err.Error())
	}
}

// collector is a second origin that records the Authorization header it is
// handed, so a test can tell "the request was refused" from "the request was
// quietly missed".
type collector struct {
	*httptest.Server

	mutex sync.Mutex
	seen  []string
}

func startCollector() *collector {
	destination := &collector{}
	destination.Server = httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			destination.mutex.Lock()
			destination.seen = append(destination.seen, request.Header.Get("Authorization"))
			destination.mutex.Unlock()
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = writer.Write([]byte(sampleDescriptor))
		}))
	return destination
}

func (destination *collector) count() int {
	destination.mutex.Lock()
	defer destination.mutex.Unlock()
	return len(destination.seen)
}

// TestRedirectDoesNotCarryTheKeyToAnotherOrigin holds the starter to the one
// promise a key deserves: it goes to the configured base URL and nowhere else.
// net/http copies Authorization onto a redirected request whenever the
// destination host is the same or a subdomain, and it compares hosts with the
// port stripped, so a different port on 127.0.0.1 leaks by the same rule a
// different port in production would.
func TestRedirectDoesNotCarryTheKeyToAnotherOrigin(t *testing.T) {
	elsewhere := startCollector()
	defer elsewhere.Close()

	redirector := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, elsewhere.URL+"/v1/llms.txt", http.StatusFound)
		}))
	defer redirector.Close()

	// The destination is live and does record what reaches it, so a count that
	// stays at one below means the request was never made, not that the test
	// looked in the wrong place.
	direct, err := ReadConfig(environment(elsewhere.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := FetchProductDescriptor(direct); err != nil {
		t.Fatalf("FetchProductDescriptor against the destination: %v", err)
	}
	if got := elsewhere.count(); got != 1 {
		t.Fatalf("destination recorded %d requests, want 1", got)
	}

	config, err := ReadConfig(environment(redirector.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	_, err = FetchProductDescriptor(config)
	if err == nil {
		t.Fatal("a redirect away from the configured origin was followed")
	}
	if !strings.Contains(err.Error(), "redirected") ||
		!strings.Contains(err.Error(), "HYPERSCALE_BASE_URL") {
		t.Errorf("error = %q, want the cause and the variable to set", err.Error())
	}
	if got := elsewhere.count(); got != 1 {
		t.Errorf("the key reached another origin: %q", elsewhere.seen[1:])
	}
}
