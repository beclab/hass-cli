// Package cmdutil wires global flags into shared config/client/renderer
// builders used by every subcommand.
package cmdutil

import (
	"os"

	"github.com/beclab/hass-cli/internal/client"
	"github.com/beclab/hass-cli/internal/config"
	"github.com/beclab/hass-cli/internal/output"
)

// Factory carries resolved global flags and lazily builds shared dependencies.
type Factory struct {
	Profile  string
	Server   string
	Token    string
	Insecure bool
	Timeout  int

	Output    string
	Columns   string
	SortBy    string
	NoHeaders bool

	cfg *config.Config
}

// Config resolves and caches the connection configuration.
func (f *Factory) Config() (*config.Config, error) {
	if f.cfg != nil {
		return f.cfg, nil
	}
	cfg, err := config.Resolve(f.Profile, f.Server, f.Token, f.Insecure, f.Timeout)
	if err != nil {
		return nil, err
	}
	f.cfg = cfg
	return cfg, nil
}

// Client validates config and returns a transport facade.
func (f *Factory) Client() (*client.Client, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return client.New(cfg), nil
}

// Renderer builds an output renderer for the chosen format and table options.
func (f *Factory) Renderer() (*output.Renderer, error) {
	format, err := output.ParseFormat(f.Output)
	if err != nil {
		return nil, err
	}
	return &output.Renderer{
		Format:    format,
		Columns:   output.ParseColumns(f.Columns),
		SortBy:    f.SortBy,
		NoHeaders: f.NoHeaders,
		Out:       os.Stdout,
	}, nil
}
