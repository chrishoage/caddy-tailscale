// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: Apache-2.0

package tscaddy

// app.go contains App and Node, which provide global configuration for registering Tailscale nodes.

import (
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"go.uber.org/zap"
	"tailscale.com/types/opt"
)

func init() {
	caddy.RegisterModule(App{})
	httpcaddyfile.RegisterGlobalOption("tailscale", parseAppConfig)
}

// App is the Tailscale Caddy app used to configure Tailscale nodes.
// Nodes can be used to serve sites privately on a Tailscale network,
// or to connect to other Tailnet nodes as upstream proxy backend.
type App struct {
	// DefaultAuthKey is the default auth key to use for Tailscale if no other auth key is specified.
	DefaultAuthKey string `json:"auth_key,omitempty" caddy:"namespace=tailscale.auth_key"`

	// ControlURL specifies the default control URL to use for nodes.
	ControlURL string `json:"control_url,omitempty" caddy:"namespace=tailscale.control_url"`

	// Ephemeral specifies whether Tailscale nodes should be registered as ephemeral.
	Ephemeral bool `json:"ephemeral,omitempty" caddy:"namespace=tailscale.ephemeral"`

	// StateDir specifies the default state directory for Tailscale nodes.
	// Each node will have a subdirectory under this parent directory for its state.
	StateDir string `json:"state_dir,omitempty" caddy:"namespace=tailscale.state_dir"`

	PortRange *PortRange `json:"port_range,omitempty" caddy:"namespace=tailscale.port_range"`

	// WebUI specifies whether Tailscale nodes should run the Web UI for remote management.
	WebUI bool `json:"webui,omitempty" caddy:"namespace=tailscale.webui"`

	// Tags specifies the list of tags to apply to all nodes.
	Tags []string `json:"tags,omitempty" caddy:"namespace=tailscale.tags"`

	// Nodes is a map of per-node configuration which overrides global options.
	Nodes map[string]Node `json:"nodes,omitempty" caddy:"namespace=tailscale"`

	logger *zap.Logger
}

type PortRange struct {
	Start uint16 `json:"start,omitempty"`
	End   uint16 `json:"end,omitempty"`
	count uint16
}

// Node is a Tailscale node configuration.
// A single node can be used to serve multiple sites on different domains or ports,
// and/or to connect to other Tailscale nodes.
type Node struct {
	// AuthKey is the Tailscale auth key used to register the node.
	AuthKey string `json:"auth_key,omitempty" caddy:"namespace=auth_key"`

	// ControlURL specifies the control URL to use for the node.
	ControlURL string `json:"control_url,omitempty" caddy:"namespace=tailscale.control_url"`

	// Ephemeral specifies whether the node should be registered as ephemeral.
	Ephemeral opt.Bool `json:"ephemeral,omitempty" caddy:"namespace=tailscale.ephemeral"`

	// WebUI specifies whether the node should run the Web UI for remote management.
	WebUI opt.Bool `json:"webui,omitempty" caddy:"namespace=tailscale.webui"`

	// Hostname is the hostname to use when registering the node.
	Hostname string `json:"hostname,omitempty" caddy:"namespace=tailscale.hostname"`

	Port uint16 `json:"port,omitempty" caddy:"namespace=tailscale.port"`

	// StateDir specifies the state directory for the node.
	StateDir string `json:"state_dir,omitempty" caddy:"namespace=tailscale.state_dir"`

	// Tags specifies the list of tags to apply to this node.
	Tags []string `json:"tags,omitempty" caddy:"namespace=tailscale.tags"`

	name string
}

func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "tailscale",
		New: func() caddy.Module { return new(App) },
	}
}

func (t *App) Provision(ctx caddy.Context) error {
	t.logger = ctx.Logger(t)
	return nil
}

func (t *App) Start() error {
	return nil
}

func (t *App) Stop() error {
	return nil
}

func parseAppConfig(d *caddyfile.Dispenser, _ any) (any, error) {
	app := &App{
		Nodes: make(map[string]Node),
	}
	if !d.Next() {
		return app, d.ArgErr()

	}

	for d.NextBlock(0) {
		val := d.Val()

		switch val {
		case "auth_key":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			app.DefaultAuthKey = d.Val()
		case "control_url":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			app.ControlURL = d.Val()
		case "ephemeral":
			if d.NextArg() {
				v, err := strconv.ParseBool(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
				app.Ephemeral = v
			} else {
				app.Ephemeral = true
			}
		case "state_dir":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			app.StateDir = d.Val()
		case "port_range":
			segment := d.NewFromNextSegment()

			if !segment.Next() {
				return nil, d.ArgErr()
			}

			app.PortRange = &PortRange{}

			for nesting := segment.Nesting(); segment.NextBlock(nesting); {
				val := segment.Val()
				switch val {
				case "start":
					if !segment.NextArg() {
						return nil, segment.ArgErr()
					}
					v, err := strconv.ParseUint(segment.Val(), 10, 16)
					if err != nil {
						return nil, segment.WrapErr(err)
					}
					app.PortRange.Start = uint16(v)
				case "end":
					if !segment.NextArg() {
						return nil, segment.ArgErr()
					}
					v, err := strconv.ParseUint(segment.Val(), 10, 16)
					if err != nil {
						return nil, segment.WrapErr(err)
					}
					app.PortRange.End = uint16(v)
				default:
					return nil, segment.Errf("unrecognized subdirective: %s", segment.Val())
				}
			}

			if app.PortRange.End < app.PortRange.Start {
				return nil, d.SyntaxErr("port range start must be less than end")
			}

		case "webui":
			if d.NextArg() {
				v, err := strconv.ParseBool(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
				app.WebUI = v
			} else {
				app.WebUI = true
			}
		case "tags":
			for d.NextArg() {
				app.Tags = append(app.Tags, d.Val())
			}
		default:
			node, err := parseNodeConfig(d)
			if app.Nodes == nil {
				app.Nodes = map[string]Node{}
			}
			if err != nil {
				return nil, err
			}
			app.Nodes[node.name] = node
		}
	}

	return httpcaddyfile.App{
		Name:  "tailscale",
		Value: caddyconfig.JSON(app, nil),
	}, nil
}

func parseNodeConfig(d *caddyfile.Dispenser) (Node, error) {
	name := d.Val()
	segment := d.NewFromNextSegment()

	if !segment.Next() {
		return Node{}, d.ArgErr()
	}

	node := Node{name: name}
	for nesting := segment.Nesting(); segment.NextBlock(nesting); {
		val := segment.Val()
		switch val {
		case "auth_key":
			if !segment.NextArg() {
				return node, segment.ArgErr()
			}
			node.AuthKey = segment.Val()
		case "control_url":
			if !segment.NextArg() {
				return node, segment.ArgErr()
			}
			node.ControlURL = segment.Val()
		case "ephemeral":
			if segment.NextArg() {
				v, err := strconv.ParseBool(segment.Val())
				if err != nil {
					return node, segment.WrapErr(err)
				}
				node.Ephemeral = opt.NewBool(v)
			} else {
				node.Ephemeral = opt.NewBool(true)
			}
		case "port":
			if segment.NextArg() {
				v, err := strconv.ParseUint(segment.Val(), 10, 16)
				if err != nil {
					return node, segment.WrapErr(err)
				}
				node.Port = uint16(v)
			} else {
				node.Port = 0
			}

		case "hostname":
			if !segment.NextArg() {
				return node, segment.ArgErr()
			}
			node.Hostname = segment.Val()
		case "state_dir":
			if !segment.NextArg() {
				return node, segment.ArgErr()
			}
			node.StateDir = segment.Val()
		case "webui":
			if segment.NextArg() {
				v, err := strconv.ParseBool(segment.Val())
				if err != nil {
					return node, segment.WrapErr(err)
				}
				node.WebUI = opt.NewBool(v)
			} else {
				node.WebUI = opt.NewBool(true)
			}
		case "tags":
			for segment.NextArg() {
				node.Tags = append(node.Tags, segment.Val())
			}
		default:
			return node, segment.Errf("unrecognized subdirective: %s", segment.Val())
		}
	}

	return node, nil
}

var (
	_ caddy.App         = (*App)(nil)
	_ caddy.Provisioner = (*App)(nil)
)
