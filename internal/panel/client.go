// Package panel is a small typed client for the Marzneshin panel REST API. It
// covers the entities vpn-setup drives as config-as-code - inbounds, services,
// and users - plus JWT sudo-admin auth. Host/node provisioning stays in Ansible.
package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 20 * time.Second

// Client talks to one Marzneshin panel. Construct with New, then Authenticate.
type Client struct {
	baseURL string
	http    *http.Client
	token   string
}

// New returns a client for the panel at baseURL (e.g. http://host:8000).
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: requestTimeout},
	}
}

// APIError is a non-2xx response from the panel.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("panel API %d: %s", e.Status, strings.TrimSpace(e.Body))
}

// NotFound reports whether err is a 404 returned by the panel.
func NotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

// Authenticate exchanges sudo-admin credentials for a bearer token.
func (c *Client) Authenticate(ctx context.Context, username, password string) error {
	form := url.Values{"username": {username}, "password": {password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/admins/token", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.send(req, &tok); err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return errors.New("panel returned an empty access token")
	}
	c.token = tok.AccessToken
	return nil
}

// do issues an authenticated JSON request. body and out may each be nil.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.send(req, out)
}

// send executes req and decodes a 2xx JSON body into out (when non-nil).
func (c *Client) send(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
