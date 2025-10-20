package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	mainHost    = "http://localhost:8080"
	serviceHost = "http://localhost:8081"
)

func isIntegrationTestsRun() bool {
	return os.Getenv("INTEGRATION_TESTS") == "true"
}

func postAPIResponse(
	serviceHost,
	path string,
	body []byte,
	headers map[string]string,
	cookies []*http.Cookie,
) ([]byte, *http.Response, error) {
	respBody, resp, err := doRequest(http.MethodPost, serviceHost, path, body, headers, cookies)
	if err != nil {
		return nil, nil, fmt.Errorf("can't do post: %w", err)
	}

	return respBody, resp, nil
}

func getAPIResponse(
	serviceHost,
	path string,
	headers map[string]string,
	cookies []*http.Cookie,
) ([]byte, *http.Response, error) {
	respBody, resp, err := doRequest(http.MethodGet, serviceHost, path, nil, headers, cookies)
	if err != nil {
		return nil, nil, fmt.Errorf("can't do get: %w", err)
	}

	return respBody, resp, nil
}
func doRequest(
	method string,
	serviceHost string,
	path string,
	body []byte,
	headers map[string]string,
	cookies []*http.Cookie,
) ([]byte, *http.Response, error) {
	buf := bytes.NewBuffer(body)

	req, err := http.NewRequestWithContext(context.Background(), method, serviceHost+path, buf)
	if err != nil {
		return nil, nil, fmt.Errorf("can't create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	for header, value := range headers {
		req.Header.Set(header, value)
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("can't do request: %w", err)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("can't read body: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("can't close body: %w", err)
	}

	return content, resp, nil
}
