package panel

import "context"

// InboundNode identifies which node advertises an inbound.
type InboundNode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Inbound is a proxy inbound discovered on a node - one per config tag. Node
// carries which node advertises it, so services can be scoped to a location.
type Inbound struct {
	ID         int         `json:"id"`
	Tag        string      `json:"tag"`
	Protocol   string      `json:"protocol"`
	ServiceIDs []int       `json:"service_ids"`
	Node       InboundNode `json:"node"`
}

// Inbounds lists every inbound the panel has discovered across all nodes.
func (c *Client) Inbounds(ctx context.Context) ([]Inbound, error) {
	return listAll[Inbound](ctx, c, "/api/inbounds")
}
