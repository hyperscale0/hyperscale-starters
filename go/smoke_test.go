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
	mock := startMockServer(apiKey, sampleOperations)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	page, err := ListOperations(config)
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}

	want := []Operation{
		{OperationID: "ops_sandbox_example01", Name: "customer.create", Status: "succeeded"},
		{OperationID: "ops_sandbox_example02", Name: "account.create", Status: "failed"},
	}
	if len(page.Operations) != len(want) {
		t.Fatalf("parsed %d operations, want %d", len(page.Operations), len(want))
	}
	for index, operation := range page.Operations {
		if operation != want[index] {
			t.Errorf("operation %d = %+v, want %+v", index, operation, want[index])
		}
	}
	if !page.HasMore {
		t.Error("a page with a next cursor reported no more")
	}

	request := mock.last()
	if request.URL != operationsPath {
		t.Errorf("url = %q, want %q", request.URL, operationsPath)
	}
	if request.Authorization != "Bearer "+apiKey {
		t.Errorf("authorization = %q", request.Authorization)
	}
	if request.Environment != "sandbox" {
		t.Errorf("environment header = %q, want sandbox", request.Environment)
	}
	if request.Accept != "application/json" {
		t.Errorf("accept = %q, want application/json", request.Accept)
	}
}

func TestLivePlaneIsAddressedByTheHeader(t *testing.T) {
	mock := startMockServer(apiKey, sampleOperations)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, map[string]string{
		"HYPERSCALE_ENVIRONMENT": "live",
	}))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := ListOperations(config); err != nil {
		t.Fatalf("ListOperations: %v", err)
	}

	if got := mock.last().Environment; got != "live" {
		t.Errorf("environment header = %q, want live", got)
	}
}

func TestRefusedKeySurfacesTheErrorCode(t *testing.T) {
	mock := startMockServer(apiKey, sampleOperations)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL, map[string]string{
		"HYPERSCALE_API_KEY": "sk_sandbox_wrong",
	}))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	_, err = ListOperations(config)
	if err == nil {
		t.Fatal("a wrong key was accepted")
	}
	if !strings.Contains(err.Error(), "HTTP 401") ||
		!strings.Contains(err.Error(), "invalid_credentials") {
		t.Errorf("error = %q, want the status and the code", err.Error())
	}
}

func TestTrailingSlashDoesNotDoubleUpThePath(t *testing.T) {
	mock := startMockServer(apiKey, sampleOperations)
	defer mock.Close()

	config, err := ReadConfig(environment(mock.URL+"///", nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := ListOperations(config); err != nil {
		t.Fatalf("ListOperations: %v", err)
	}

	if got := mock.last().URL; got != operationsPath {
		t.Errorf("url = %q, want %q", got, operationsPath)
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

func TestResponseThatIsNotAnOperationListIsRefused(t *testing.T) {
	for _, body := range []string{"# Example Product", `{"error":"none"}`, `{"items":[{"name":"customer.create"}]}`} {
		if _, err := ParseOperationPage([]byte(body)); err == nil {
			t.Errorf("accepted %s", body)
		}
	}

	page, err := ParseOperationPage([]byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("an empty page was refused: %v", err)
	}
	if len(page.Operations) != 0 || page.HasMore {
		t.Errorf("empty page = %+v", page)
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
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(sampleOperations))
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
			http.Redirect(writer, request, elsewhere.URL+operationsPath, http.StatusFound)
		}))
	defer redirector.Close()

	// The destination is live and does record what reaches it, so a count that
	// stays at one below means the request was never made, not that the test
	// looked in the wrong place.
	direct, err := ReadConfig(environment(elsewhere.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if _, err := ListOperations(direct); err != nil {
		t.Fatalf("ListOperations against the destination: %v", err)
	}
	if got := elsewhere.count(); got != 1 {
		t.Fatalf("destination recorded %d requests, want 1", got)
	}

	config, err := ReadConfig(environment(redirector.URL, nil))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	_, err = ListOperations(config)
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
