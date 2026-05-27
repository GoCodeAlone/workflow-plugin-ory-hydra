package internal

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	hydra "github.com/ory/hydra-client-go/v2"
)

type oryHydraModule struct {
	name   string
	config map[string]any
}

func newOryHydraModule(name string, config map[string]any) (*oryHydraModule, error) {
	return &oryHydraModule{name: name, config: config}, nil
}

func (m *oryHydraModule) Init() error {
	adminURL := firstNonEmpty(m.config, "admin_url", "adminUrl", "admin")
	if adminURL == "" {
		return fmt.Errorf("ory.hydra %q: admin_url is required", m.name)
	}
	apiKey := firstNonEmpty(m.config, "api_key", "apiKey", "token")

	adminClient, err := newHydraClient(adminURL, apiKey)
	if err != nil {
		return fmt.Errorf("ory.hydra %q: create admin client: %w", m.name, err)
	}
	RegisterClient(m.name, &OryHydraClient{Admin: adminClient, AdminURL: strings.TrimRight(adminURL, "/")})
	return nil
}

func (m *oryHydraModule) Start(context.Context) error { return nil }

func (m *oryHydraModule) Stop(context.Context) error {
	UnregisterClient(m.name)
	return nil
}

func newHydraClient(rawURL, apiKey string) (*hydra.APIClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("url must include scheme and host")
	}
	cfg := hydra.NewConfiguration()
	cfg.Scheme = parsed.Scheme
	cfg.Host = parsed.Host
	if path := strings.TrimRight(parsed.Path, "/"); path != "" {
		cfg.Servers = hydra.ServerConfigurations{{URL: parsed.Scheme + "://" + parsed.Host + path}}
	}
	if apiKey != "" {
		cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	}
	cfg.UserAgent = "workflow-plugin-ory-hydra/" + Version
	return hydra.NewAPIClient(cfg), nil
}
