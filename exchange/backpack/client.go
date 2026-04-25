package backpack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const (
	backpackBaseURL = "https://api.backpack.exchange"
	backpackWSURL   = "wss://ws.backpack.exchange"
	backpackBroker  = "2140"
)

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Backpack API 错误: [%s] %s (状态码: %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("Backpack API 错误: %s (状态码: %d)", e.Message, e.StatusCode)
}

type Client struct {
	httpClient *http.Client
	signer     *Signer
	baseURL    string
}

func NewClient(apiKey, secretKey string) (*Client, error) {
	signer, err := NewSigner(apiKey, secretKey)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		signer:     signer,
		baseURL:    backpackBaseURL,
	}, nil
}

func (c *Client) SubscribeSignature() ([]string, error) {
	return c.signer.SubscribeSignature()
}

func (c *Client) DoPublicRequest(ctx context.Context, method, path string, query map[string]string) ([]byte, error) {
	return c.doRequest(ctx, method, path, query, nil, nil)
}

func (c *Client) DoSignedRequest(ctx context.Context, method, path, instruction string, query map[string]string, body map[string]any, extraHeaders map[string]string) ([]byte, error) {
	headers, err := c.signer.SignedHeaders(instruction, query, body)
	if err != nil {
		return nil, err
	}

	for key, value := range extraHeaders {
		headers[key] = value
	}

	return c.doRequest(ctx, method, path, query, body, headers)
}

func (c *Client) doRequest(ctx context.Context, method, path string, query map[string]string, body map[string]any, headers map[string]string) ([]byte, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化 Backpack 请求体失败: %w", err)
		}
	}

	requestURL := c.baseURL + path
	if encodedQuery := encodeQuery(query); encodedQuery != "" {
		requestURL += "?" + encodedQuery
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建 Backpack 请求失败: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Backpack 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Backpack 响应失败: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	var apiResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err == nil {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       apiResp.Code,
			Message:    apiResp.Message,
		}
	}

	return nil, &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
}

func encodeQuery(query map[string]string) string {
	if len(query) == 0 {
		return ""
	}

	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := url.Values{}
	for _, key := range keys {
		values.Set(key, query[key])
	}

	return values.Encode()
}
