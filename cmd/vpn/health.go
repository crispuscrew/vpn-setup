package main

import (
	"fmt"
	"strings"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

// runHealth reports panel reachability and per-node status. It exits non-zero if
// any node is not healthy (or none are registered), so it doubles as a monitoring
// probe.
func runHealth(args []string) error {
	ctx, cancel := commandContext()
	defer cancel()
	client, err := panelClient(ctx)
	if err != nil {
		return err
	}
	nodes, err := client.Nodes(ctx)
	if err != nil {
		return err
	}

	report, unhealthy := healthReport(nodes)
	fmt.Print(report)

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes registered")
	}
	if unhealthy > 0 {
		return fmt.Errorf("%d of %d node(s) unhealthy", unhealthy, len(nodes))
	}
	return nil
}

// healthReport formats the panel/node health lines and returns the count of
// unhealthy nodes. It is pure so it can be tested without a live panel.
func healthReport(nodes []panel.Node) (string, int) {
	var out strings.Builder
	out.WriteString("panel: reachable, authenticated\n")
	if len(nodes) == 0 {
		out.WriteString("nodes: none registered\n")
		return out.String(), 0
	}

	unhealthy := 0
	for _, node := range nodes {
		mark := "ok"
		if node.Status != panel.NodeHealthy {
			mark = "DOWN"
			unhealthy++
		}
		cores := make([]string, 0, len(node.Backends))
		for _, backend := range node.Backends {
			cores = append(cores, backend.Name+"/"+backend.Version)
		}
		fmt.Fprintf(&out, "node %-12s %-9s %-4s backends=%v\n", node.Name, node.Status, mark, cores)
	}
	fmt.Fprintf(&out, "%d of %d node(s) healthy\n", len(nodes)-unhealthy, len(nodes))
	return out.String(), unhealthy
}
