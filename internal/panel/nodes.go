package panel

import "context"

// NodeHealthy is the status of a node that is connected and serving.
const NodeHealthy = "healthy"

// Backend is a proxy core running on a node (xray/hysteria/sing-box) with version.
type Backend struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// Node is a marznode registered with the panel, together with its health.
type Node struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	Backends []Backend `json:"backends"`
}

// Nodes lists all registered nodes and their status.
func (c *Client) Nodes(ctx context.Context) ([]Node, error) {
	return listAll[Node](ctx, c, "/api/nodes")
}
