package panel

import (
	"context"
	"net/http"
)

// ExpireNever is the expire_strategy of a user that never expires.
const ExpireNever = "never"

// User is a panel user together with their subscription URL.
type User struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	ExpireStrategy  string `json:"expire_strategy"`
	ServiceIDs      []int  `json:"service_ids"`
	SubscriptionURL string `json:"subscription_url"`
	Enabled         bool   `json:"enabled"`
	Key             string `json:"key"`
}

type userCreate struct {
	Username       string `json:"username"`
	ExpireStrategy string `json:"expire_strategy"`
	ServiceIDs     []int  `json:"service_ids"`
	Note           string `json:"note,omitempty"`
}

type userModify struct {
	ServiceIDs     []int  `json:"service_ids,omitempty"`
	ExpireStrategy string `json:"expire_strategy,omitempty"`
}

// Users lists all users.
func (c *Client) Users(ctx context.Context) ([]User, error) {
	return listAll[User](ctx, c, "/api/users")
}

// User fetches one user by name; test the error with NotFound for absence.
func (c *Client) User(ctx context.Context, username string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodGet, "/api/users/"+username, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateUser creates a user granted the given services.
func (c *Client) CreateUser(ctx context.Context, username, expireStrategy string, serviceIDs []int, note string) (*User, error) {
	var out User
	body := userCreate{Username: username, ExpireStrategy: expireStrategy, ServiceIDs: serviceIDs, Note: note}
	if err := c.do(ctx, http.MethodPost, "/api/users", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUser changes a user's services and expire strategy.
func (c *Client) UpdateUser(ctx context.Context, username, expireStrategy string, serviceIDs []int) (*User, error) {
	var out User
	body := userModify{ServiceIDs: serviceIDs, ExpireStrategy: expireStrategy}
	if err := c.do(ctx, http.MethodPut, "/api/users/"+username, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUser removes a user.
func (c *Client) DeleteUser(ctx context.Context, username string) error {
	return c.do(ctx, http.MethodDelete, "/api/users/"+username, nil, nil)
}

// RevokeSubscription rotates a user's key so their current subscription URL stops
// working (a fresh one is issued in its place).
func (c *Client) RevokeSubscription(ctx context.Context, username string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodPost, "/api/users/"+username+"/revoke_sub", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
