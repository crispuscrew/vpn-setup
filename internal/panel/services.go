package panel

import (
	"context"
	"fmt"
	"net/http"
)

// Service groups inbounds; a user is granted access one service at a time.
type Service struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	InboundIDs []int  `json:"inbound_ids"`
	UserIDs    []int  `json:"user_ids"`
}

type serviceWrite struct {
	Name       string `json:"name"`
	InboundIDs []int  `json:"inbound_ids"`
}

// Services lists all services.
func (c *Client) Services(ctx context.Context) ([]Service, error) {
	return listAll[Service](ctx, c, "/api/services")
}

// CreateService creates a service grouping the given inbound ids.
func (c *Client) CreateService(ctx context.Context, name string, inboundIDs []int) (*Service, error) {
	var out Service
	body := serviceWrite{Name: name, InboundIDs: inboundIDs}
	if err := c.do(ctx, http.MethodPost, "/api/services", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateService replaces a service's name and inbound membership.
func (c *Client) UpdateService(ctx context.Context, id int, name string, inboundIDs []int) (*Service, error) {
	var out Service
	body := serviceWrite{Name: name, InboundIDs: inboundIDs}
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/api/services/%d", id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
