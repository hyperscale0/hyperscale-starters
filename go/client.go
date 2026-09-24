// Package main calls the Product API with net/http and nothing else.
//
// Two headers carry everything. "Authorization: Bearer <key>" presents the
// Product API key, and "X-Hyperscale-Environment" picks which plane of that key
// is being addressed. Sandbox keys and live keys are never interchangeable, so
// the header is not a hint: it is half the credential. HTTP header names are
// case-insensitive; these are the canonical spellings.
//
// The key goes to HYPERSCALE_BASE_URL and nowhere else: this starter refuses
// redirects rather than following them, the same as the TypeScript and Python
// starters. The endpoint is fixed and read-only, so there is no hop worth
// taking.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the public origin. Override it with HYPERSCALE_BASE_URL.
const DefaultBaseURL = "https://hyperscale0.ai"

const requestTimeout = 30 * time.Second

// operationsPath asks for one page of ten, enough to see the shape.
const operationsPath = "/v1/operations?limit=10"

// Config is the whole configuration surface: three environment variables.
type Config struct {
	APIKey      string
	BaseURL     string
	Environment string
}

// Operation is one recorded execution: what ran and how it ended.
type Operation struct {
	OperationID string
	Name        string
	Status      string
}

// OperationPage is what the smoke call returns, parsed.
type OperationPage struct {
	Operations []Operation
	HasMore    bool
}

// ProductAPIError is a failure the person running this can fix. main prints it
// on its own and exits 1, so it never reaches a reader as a panic.
type ProductAPIError struct{ Message string }

func (e *ProductAPIError) Error() string { return e.Message }

func errorf(format string, args ...any) error {
	return &ProductAPIError{Message: fmt.Sprintf(format, args...)}
}

// ReadConfig reads the three environment variables through the lookup a caller
// hands it, so a test never has to mutate the process environment.
func ReadConfig(lookup func(string) string) (Config, error) {
	apiKey := lookup("HYPERSCALE_API_KEY")
	if apiKey == "" {
		return Config{}, errorf(
			"No API key. Set HYPERSCALE_API_KEY to a Product API key; README.md says where to mint one.")
	}

	environment := lookup("HYPERSCALE_ENVIRONMENT")
	if environment == "" {
		environment = "sandbox"
	}
	if environment != "sandbox" && environment != "live" {
		return Config{}, errorf("HYPERSCALE_ENVIRONMENT must be sandbox or live, not %s.", environment)
	}

	baseURL := lookup("HYPERSCALE_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	// A trailing slash would make every path double up on one.
	baseURL = strings.TrimRight(baseURL, "/")

	return Config{APIKey: apiKey, BaseURL: baseURL, Environment: environment}, nil
}

// ListOperations makes the one read every Product key is allowed, whatever the
// Product was composed from: the capability that serves it is part of every
// Product. A fresh Product has run nothing yet, so an empty list with HTTP 200
// still proves the key.
func ListOperations(config Config) (OperationPage, error) {
	url := config.BaseURL + operationsPath

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OperationPage{}, errorf("Could not build a request for %s: %v", url, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+config.APIKey)
	request.Header.Set("X-Hyperscale-Environment", config.Environment)

	// net/http follows redirects by default and carries Authorization along
	// whenever the destination host is the same or a subdomain, comparing hosts
	// with the port stripped. A key belongs to exactly one origin, and this
	// endpoint is fixed and read-only, so no 3xx is worth following.
	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return OperationPage{}, errorf("Could not reach %s: %v", url, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return OperationPage{}, errorf("Could not read the response from %s: %v", url, err)
	}

	if response.StatusCode >= 300 && response.StatusCode <= 399 {
		return OperationPage{}, errorf(
			"GET /v1/operations was redirected, so the key was not sent on. " +
				"Set HYPERSCALE_BASE_URL to the origin the API answers on; " +
				"the usual cause is http:// where it serves https://.")
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		detail := errorEnvelope(body)
		if detail == "" {
			detail = strings.TrimSpace(string(body))
		}
		if detail != "" {
			detail = " - " + detail
		}
		return OperationPage{}, errorf(
			"GET /v1/operations failed: HTTP %d%s", response.StatusCode, detail)
	}

	return ParseOperationPage(body)
}

// ParseOperationPage refuses anything that is not an operation list, and any
// item without an id or a name. A starter that printed blank rows would hide
// the format moving under it.
func ParseOperationPage(body []byte) (OperationPage, error) {
	var page struct {
		Items []struct {
			OperationID string `json:"operationId"`
			Name        string `json:"name"`
			Status      string `json:"status"`
		} `json:"items"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(body, &page); err != nil || page.Items == nil {
		return OperationPage{}, errorf(
			"The response is not an operation list. Check HYPERSCALE_BASE_URL points at the API origin.")
	}

	operations := make([]Operation, 0, len(page.Items))
	for _, item := range page.Items {
		if item.OperationID == "" || item.Name == "" {
			return OperationPage{}, errorf(
				"An operation arrived without an id or a name; this starter is out of date.")
		}
		operations = append(operations, Operation{
			OperationID: item.OperationID,
			Name:        item.Name,
			Status:      item.Status,
		})
	}

	return OperationPage{Operations: operations, HasMore: page.NextCursor != ""}, nil
}

// errorEnvelope reads the API's error shape: {"error":{"code","message"},"requestId"}.
// A non-JSON body is already the best message available, so it returns "".
func errorEnvelope(body []byte) string {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	if envelope.Error.Code == "" || envelope.Error.Message == "" {
		return ""
	}
	return envelope.Error.Code + ": " + envelope.Error.Message
}
