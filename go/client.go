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
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the public origin. Override it with HYPERSCALE_BASE_URL.
const DefaultBaseURL = "https://hyperscale0.ai"

const requestTimeout = 30 * time.Second

// Config is the whole configuration surface: three environment variables.
type Config struct {
	APIKey      string
	BaseURL     string
	Environment string
}

// Operation is one callable route of a Product.
type Operation struct {
	Method      string
	Path        string
	OperationID string
}

// ProductDescriptor is what the smoke call returns, parsed.
type ProductDescriptor struct {
	Title      string
	Operations []Operation
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

// FetchProductDescriptor makes the one call every Product serves, whatever it
// was composed from. Everything else in the API surface exists because the
// Product composed the capability behind it, so this is the only fair smoke
// test.
func FetchProductDescriptor(config Config) (ProductDescriptor, error) {
	url := config.BaseURL + "/v1/llms.txt"

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ProductDescriptor{}, errorf("Could not build a request for %s: %v", url, err)
	}
	request.Header.Set("Accept", "text/plain")
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
		return ProductDescriptor{}, errorf("Could not reach %s: %v", url, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return ProductDescriptor{}, errorf("Could not read the response from %s: %v", url, err)
	}

	if response.StatusCode >= 300 && response.StatusCode <= 399 {
		return ProductDescriptor{}, errorf(
			"GET /v1/llms.txt was redirected, so the key was not sent on. " +
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
		return ProductDescriptor{}, errorf(
			"GET /v1/llms.txt failed: HTTP %d%s", response.StatusCode, detail)
	}

	return ParseProductDescriptor(string(body))
}

var (
	titlePattern         = regexp.MustCompile(`(?m)^# (.+)$`)
	declaredCountPattern = regexp.MustCompile(`(?m)^## Operations \((\d+)\)$`)
	operationPattern     = regexp.MustCompile(
		`(?m)^- ([A-Z]+) (\S+) · .+ \((\S+); idempotency \S+\)$`)
)

// ParseProductDescriptor reads the descriptor with three anchored patterns
// rather than guessing. Parsing fewer operations than the document declares
// means the format moved under us, and that has to fail loudly: a starter that
// silently printed a short list would look like a Product missing half its
// surface.
func ParseProductDescriptor(document string) (ProductDescriptor, error) {
	title := titlePattern.FindStringSubmatch(document)
	declared := declaredCountPattern.FindStringSubmatch(document)
	if title == nil || declared == nil {
		return ProductDescriptor{}, errorf(
			"The response is not a product descriptor. Check HYPERSCALE_BASE_URL points at the API origin.")
	}

	operations := []Operation{}
	for _, match := range operationPattern.FindAllStringSubmatch(document, -1) {
		operations = append(operations, Operation{
			Method:      match[1],
			Path:        match[2],
			OperationID: match[3],
		})
	}

	expected, err := strconv.Atoi(declared[1])
	if err != nil {
		return ProductDescriptor{}, errorf("The descriptor declares an unreadable operation count.")
	}
	if len(operations) != expected {
		return ProductDescriptor{}, errorf(
			"The descriptor declares %d operations but %d parsed; this starter is out of date.",
			expected, len(operations))
	}

	return ProductDescriptor{Title: strings.TrimSpace(title[1]), Operations: operations}, nil
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
