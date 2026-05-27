package internal

import (
	"context"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type authProviderDescribeStep struct {
	name   string
	config map[string]any
}

func newAuthProviderDescribeStep(name string, config map[string]any) (sdk.StepInstance, error) {
	return &authProviderDescribeStep{name: name, config: config}, nil
}

func (s *authProviderDescribeStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current, _, _ map[string]any) (*sdk.StepResult, error) {
	values := mergeMaps(s.config, current)
	providerID := firstNonEmpty(values, "provider_id", "providerId")
	if providerID == "" {
		providerID = "ory_hydra"
	}
	adminURL := firstNonEmpty(values, "admin_url", "adminUrl")
	return &sdk.StepResult{Output: map[string]any{
		"providers": []map[string]any{oryHydraProviderDescriptor(providerID, adminURL)},
	}}, nil
}

func oryHydraProviderDescriptor(providerID, adminURL string) map[string]any {
	return map[string]any{
		"id":             providerID,
		"label":          "Ory Hydra",
		"description":    "Ory Hydra OAuth2 and OpenID Connect administration integration.",
		"categories":     []string{"oauth2_oidc"},
		"implementation": "workflow-plugin-ory-hydra",
		"version":        Version,
		"docs_url":       "https://github.com/GoCodeAlone/workflow-plugin-ory-hydra",
		"support_level":  "management",
		"capabilities": []map[string]any{
			oryHydraCapability("ory_hydra_oauth2_clients", "OAuth2 clients", "oauth2_oidc", "Create, read, list, update, and delete Hydra OAuth2/OIDC clients through the Admin API.", []string{"hydra.oauth2.clients.read", "hydra.oauth2.clients.write"}, oryHydraFields(adminURL)),
			oryHydraCapability("ory_hydra_json_web_keys", "JSON Web Keys", "oauth2_oidc", "Create and manage Hydra JSON Web Key sets used by OAuth2/OIDC clients and issuers.", []string{"hydra.jwks.read", "hydra.jwks.write"}, oryHydraFields(adminURL)),
			oryHydraCapability("ory_hydra_trusted_jwt_issuers", "Trusted JWT issuers", "oauth2_oidc", "Manage trusted JWT bearer grant issuers for OAuth2 token exchange.", []string{"hydra.jwt_issuers.read", "hydra.jwt_issuers.write"}, oryHydraFields(adminURL)),
			oryHydraCapability("ory_hydra_consent_sessions", "Consent sessions", "oauth2_oidc", "List OAuth2 consent sessions for a subject so administrators can audit delegated access.", []string{"hydra.consent_sessions.read"}, oryHydraFields(adminURL)),
		},
	}
}

func oryHydraCapability(key, label, category, description string, appScopes []string, fields []map[string]any) map[string]any {
	return map[string]any{
		"key":                key,
		"label":              label,
		"category":           category,
		"description":        description,
		"supported":          true,
		"app_scopes":         appScopes,
		"admin_read_scopes":  []string{"admin.auth.providers.read"},
		"admin_write_scopes": []string{"admin.auth.providers.write"},
		"config_fields":      fields,
	}
}

func oryHydraFields(adminURL string) []map[string]any {
	return []map[string]any{
		oryHydraField("ory_hydra_admin_url", "Admin API URL", "url", "Base URL for the Hydra Admin API.", "Keep this private. Do not expose the Admin API directly to browsers.", false, true, optionIfSet(strings.TrimRight(adminURL, "/"))),
		oryHydraField("ory_hydra_api_key", "Admin API bearer token", "secret", "Optional bearer token for protected Admin API deployments.", "Write-only secret. Prefer network isolation and least-privilege reverse-proxy credentials.", true, false, nil),
	}
}

func oryHydraField(key, label, inputType, description, helpText string, secret, required bool, options []map[string]any) map[string]any {
	return map[string]any{
		"key":         key,
		"label":       label,
		"input_type":  inputType,
		"description": description,
		"help_text":   helpText,
		"secret":      secret,
		"required":    required,
		"options":     options,
	}
}

func optionIfSet(value string) []map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []map[string]any{{"value": value, "label": value}}
}
