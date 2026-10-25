package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const canonicalCapability = "infrai.auth.consent.check"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("consent API rejected request (%d): %s", e.Status, e.Body)
}

type ConsentClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewConsentClient(baseURL, key string) *ConsentClient {
	return &ConsentClient{baseURL: strings.TrimRight(baseURL, "/"), key: key, http: &http.Client{Timeout: 8 * time.Second}}
}

func ClientFromEnvironment(baseURL string) (*ConsentClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return NewConsentClient(baseURL, key), nil
}

func (c *ConsentClient) Check(ctx context.Context, userID, category string) (bool, error) {
	path := "/v1/auth/consent/check/" + userID + "/" + category
	var data struct {
		Granted bool `json:"granted"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &data); err != nil {
		return false, err
	}
	return data.Granted, nil
}

func (c *ConsentClient) Revoke(ctx context.Context, userID, category string, idempotencyKey ...string) error {
	request := map[string]string{
		"user_id":  userID,
		"category": category,
	}
	if len(idempotencyKey) > 0 && idempotencyKey[0] != "" {
		request["idempotency_key"] = idempotencyKey[0]
	}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/v1/auth/consent/revoke/"+userID, body, nil)
}

func (c *ConsentClient) do(ctx context.Context, method, path string, body []byte, out any) error {
	for attempt := 0; attempt < 3; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		payload, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			if res.StatusCode >= 500 {
				return &APIError{Status: res.StatusCode, Body: string(payload)}
			}
			return fmt.Errorf("invalid API envelope: %w", err)
		}
		if !env.OK {
			return &APIError{Status: res.StatusCode, Body: string(env.Error)}
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			wait := time.Duration(1<<attempt) * 100 * time.Millisecond
			if raw := res.Header.Get("Retry-After"); raw != "" {
				if seconds, parseErr := strconv.Atoi(raw); parseErr == nil {
					wait = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(wait)
			continue
		}
		if res.StatusCode >= 500 {
			return &APIError{Status: res.StatusCode, Body: string(payload)}
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("consent request retry limit reached")
}
