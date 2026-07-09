package main

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

func runApply(args []string) error {
	flags := flag.NewFlagSet("apply", flag.ContinueOnError)
	file := flags.String("f", "vpn.yaml", "path to the desired-state config file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg, err := loadConfig(*file)
	if err != nil {
		return err
	}

	ctx, cancel := commandContext()
	defer cancel()
	client, err := panelClient(ctx)
	if err != nil {
		return err
	}

	inboundIDByTag, err := inboundIndex(ctx, client)
	if err != nil {
		return err
	}
	serviceIDByName, err := applyServices(ctx, client, cfg.Services, inboundIDByTag)
	if err != nil {
		return err
	}
	return applyUsers(ctx, client, cfg.Users, serviceIDByName)
}

// inboundIndex maps each discovered inbound tag to its id.
func inboundIndex(ctx context.Context, client *panel.Client) (map[string]int, error) {
	inbounds, err := client.Inbounds(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]int, len(inbounds))
	for _, inbound := range inbounds {
		index[inbound.Tag] = inbound.ID
	}
	return index, nil
}

// resolveInbounds turns a spec's inbound tags (or "*") into a sorted id list.
func resolveInbounds(spec ServiceSpec, idByTag map[string]int) ([]int, error) {
	if len(spec.Inbounds) == 1 && spec.Inbounds[0] == "*" {
		ids := make([]int, 0, len(idByTag))
		for _, id := range idByTag {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		return ids, nil
	}
	ids := make([]int, 0, len(spec.Inbounds))
	for _, tag := range spec.Inbounds {
		id, ok := idByTag[tag]
		if !ok {
			return nil, fmt.Errorf("service %q: no inbound tagged %q on any node", spec.Name, tag)
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// applyServices reconciles declared services and returns every service name→id,
// so users may reference services declared here or pre-existing on the panel.
func applyServices(ctx context.Context, client *panel.Client, specs []ServiceSpec, idByTag map[string]int) (map[string]int, error) {
	existing, err := client.Services(ctx)
	if err != nil {
		return nil, err
	}
	idByName := make(map[string]int, len(existing))
	current := make(map[string]panel.Service, len(existing))
	for _, service := range existing {
		idByName[service.Name] = service.ID
		current[service.Name] = service
	}

	for _, spec := range specs {
		want, err := resolveInbounds(spec, idByTag)
		if err != nil {
			return nil, err
		}
		if have, ok := current[spec.Name]; ok {
			if sameInts(have.InboundIDs, want) {
				fmt.Printf("service %-16s unchanged\n", spec.Name)
				continue
			}
			if _, err := client.UpdateService(ctx, have.ID, spec.Name, want); err != nil {
				return nil, err
			}
			fmt.Printf("service %-16s updated\n", spec.Name)
			continue
		}
		created, err := client.CreateService(ctx, spec.Name, want)
		if err != nil {
			return nil, err
		}
		idByName[spec.Name] = created.ID
		fmt.Printf("service %-16s created\n", spec.Name)
	}
	return idByName, nil
}

func applyUsers(ctx context.Context, client *panel.Client, specs []UserSpec, serviceIDByName map[string]int) error {
	for _, spec := range specs {
		serviceIDs, err := resolveServices(spec, serviceIDByName)
		if err != nil {
			return err
		}
		strategy := spec.ExpireStrategy
		if strategy == "" {
			strategy = panel.ExpireNever
		}

		user, err := client.User(ctx, spec.Username)
		switch {
		case err == nil:
			if sameInts(user.ServiceIDs, serviceIDs) && user.ExpireStrategy == strategy {
				fmt.Printf("user    %-16s unchanged\n", spec.Username)
				continue
			}
			if _, err := client.UpdateUser(ctx, spec.Username, strategy, serviceIDs); err != nil {
				return err
			}
			fmt.Printf("user    %-16s updated\n", spec.Username)
		case panel.NotFound(err):
			created, err := client.CreateUser(ctx, spec.Username, strategy, serviceIDs, spec.Note)
			if err != nil {
				return err
			}
			fmt.Printf("user    %-16s created → %s\n", spec.Username, created.SubscriptionURL)
		default:
			return err
		}
	}
	return nil
}

func resolveServices(spec UserSpec, idByName map[string]int) ([]int, error) {
	ids := make([]int, 0, len(spec.Services))
	for _, name := range spec.Services {
		id, ok := idByName[name]
		if !ok {
			return nil, fmt.Errorf("user %q: unknown service %q", spec.Username, name)
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// sameInts reports whether two id lists hold the same set of ids.
func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]int(nil), a...)
	y := append([]int(nil), b...)
	sort.Ints(x)
	sort.Ints(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
