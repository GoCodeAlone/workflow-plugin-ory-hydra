package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-ory-hydra/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestPluginManifestAdvertisesRequiredSecrets(t *testing.T) {
	raw, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RequiredSecrets []struct {
			Name      string `json:"name"`
			Sensitive bool   `json:"sensitive"`
		} `json:"required_secrets"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	secrets := map[string]bool{}
	for _, secret := range manifest.RequiredSecrets {
		secrets[secret.Name] = secret.Sensitive
	}
	for name, sensitive := range map[string]bool{
		"ORY_HYDRA_ADMIN_URL": false,
		"ORY_HYDRA_API_KEY":   true,
	} {
		got, ok := secrets[name]
		if !ok {
			t.Fatalf("plugin.json missing required_secrets entry %s", name)
		}
		if got != sensitive {
			t.Fatalf("%s sensitive = %v, want %v", name, got, sensitive)
		}
	}
}

func TestModuleInitRegistersHydraAdminClient(t *testing.T) {
	module, err := newOryHydraModule("ory_hydra-test", map[string]any{
		"adminUrl": "https://hydra-admin.example.test",
		"apiKey":   "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	client, ok := GetClient("ory_hydra-test")
	if !ok || client == nil {
		t.Fatal("expected registered client")
	}
	if client.Admin == nil {
		t.Fatal("expected hydra admin client")
	}
	if client.AdminURL != "https://hydra-admin.example.test" {
		t.Fatalf("admin url = %q", client.AdminURL)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetClient("ory_hydra-test"); ok {
		t.Fatal("expected client to be unregistered")
	}
}

func TestModuleInitRequiresAdminURL(t *testing.T) {
	module, err := newOryHydraModule("ory_hydra-test", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing admin url error")
	}
}

func TestContractRegistryIncludesStrictProtoDescriptors(t *testing.T) {
	provider, ok := NewOryHydraPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	if !ok {
		t.Fatal("plugin does not expose ContractRegistry")
	}
	registry := provider.ContractRegistry()
	if registry == nil || registry.GetFileDescriptorSet() == nil {
		t.Fatal("missing contract registry file descriptors")
	}
	contractsByType := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.GetContracts() {
		switch contract.GetKind() {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByType["module:"+contract.GetModuleType()] = contract
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByType["step:"+contract.GetStepType()] = contract
		}
	}
	module := contractsByType["module:ory.hydra"]
	if module == nil || module.GetConfigMessage() != "workflow.plugins.ory_hydra.v1.ProviderConfig" {
		t.Fatalf("unexpected module contract: %#v", module)
	}
	for _, stepType := range allStepTypes() {
		contract := contractsByType["step:"+stepType]
		if contract == nil {
			t.Fatalf("missing step contract %s", stepType)
		}
		if contract.GetMode() != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v", stepType, contract.GetMode())
		}
	}
}

func TestDescriptorAdvertisesOnlyBackedHydraCapabilities(t *testing.T) {
	step, err := newAuthProviderDescribeStep("describe", map[string]any{"adminUrl": "https://hydra-admin.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	providers := result.Output["providers"].([]map[string]any)
	if len(providers) != 1 {
		t.Fatalf("providers = %#v", providers)
	}
	provider := providers[0]
	categories := stringSet(provider["categories"].([]string))
	if !categories["oauth2_oidc"] {
		t.Fatal("descriptor must advertise OAuth/OIDC provider support")
	}
	for _, absent := range []string{"identity_management", "authentication_method", "enterprise_sso", "rbac", "mfa", "scim"} {
		if categories[absent] {
			t.Fatalf("descriptor must not advertise %s", absent)
		}
	}
	capabilities := provider["capabilities"].([]map[string]any)
	keys := map[string]bool{}
	for _, capability := range capabilities {
		keys[capability["key"].(string)] = true
		if capability["supported"] != true {
			t.Fatalf("%s supported = %#v", capability["key"], capability["supported"])
		}
	}
	for _, key := range []string{
		"ory_hydra_oauth2_clients",
		"ory_hydra_json_web_keys",
		"ory_hydra_trusted_jwt_issuers",
		"ory_hydra_consent_sessions",
	} {
		if !keys[key] {
			t.Fatalf("missing capability %s", key)
		}
	}
}

func TestTypedDescriptor(t *testing.T) {
	result, err := typedAuthProviderDescribe(context.Background(), sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]{
		Config: &contracts.AuthProviderDescribeConfig{ProviderId: "ory_hydra-admin"},
		Input:  &contracts.AuthProviderDescribeInput{AdminUrl: "https://hydra-admin.example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == nil || len(result.Output.GetProviders()) != 1 {
		t.Fatalf("providers = %#v", result.Output)
	}
	if result.Output.GetProviders()[0].GetId() != "ory_hydra-admin" {
		t.Fatalf("provider id = %q", result.Output.GetProviders()[0].GetId())
	}
}

func TestMissingClientReturnsStepError(t *testing.T) {
	step, err := createStep("step.ory_hydra_oauth2_client_get", "get", map[string]any{"module": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"client_id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] == nil {
		t.Fatalf("expected error output, got %#v", result.Output)
	}
}

func TestOAuth2ClientCreateUsesHydraAdminAPI(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if gotPath != "/admin/clients" {
			t.Fatalf("path = %s", gotPath)
		}
		body := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["client_name"] != "demo app" {
			t.Fatalf("client_name = %#v", body["client_name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"client-1","client_name":"demo app","redirect_uris":["https://app.example/callback"],"grant_types":["authorization_code"],"response_types":["code"],"scope":"openid profile"}`))
	}))
	defer server.Close()

	module, err := newOryHydraModule("ory_hydra-test", map[string]any{"adminUrl": server.URL, "apiKey": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_hydra_oauth2_client_create", "create", map[string]any{"module": "ory_hydra-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{
		"client_name":                "demo app",
		"redirect_uris":              []any{"https://app.example/callback"},
		"grant_types":                []any{"authorization_code"},
		"response_types":             []any{"code"},
		"scope":                      "openid profile",
		"token_endpoint_auth_method": "client_secret_basic",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["client"].(map[string]any)["client_id"] != "client-1" {
		t.Fatalf("output = %#v", result.Output)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization header = %q", gotAuth)
	}
}

func TestJSONWebKeySetCreateUsesHydraAdminAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/admin/keys/hydra.openid.id-token" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		body := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["alg"] != "RS256" || body["kid"] != "kid-1" || body["use"] != "sig" {
			t.Fatalf("body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[{"alg":"RS256","kid":"kid-1","kty":"RSA","use":"sig"}]}`))
	}))
	defer server.Close()

	module, err := newOryHydraModule("ory_hydra-test", map[string]any{"adminUrl": server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_hydra_json_web_key_set_create", "create-key-set", map[string]any{"module": "ory_hydra-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"set": "hydra.openid.id-token", "alg": "RS256", "kid": "kid-1", "use": "sig"})
	if err != nil {
		t.Fatal(err)
	}
	keys := result.Output["jwk_set"].(map[string]any)["keys"].([]any)
	if keys[0].(map[string]any)["kid"] != "kid-1" {
		t.Fatalf("output = %#v", result.Output)
	}
}

func TestTrustedJWTIssuerAndConsentSessionStepsUseHydraAdminAPI(t *testing.T) {
	var sawIssuer bool
	var sawConsent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/trust/grants/jwt-bearer/issuers":
			sawIssuer = true
			_, _ = w.Write([]byte(`[{"id":"issuer-1","issuer":"https://issuer.example","scope":["read"]}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/admin/oauth2/auth/sessions/consent":
			sawConsent = true
			if got := r.URL.Query().Get("subject"); got != "user-1" {
				t.Fatalf("subject query = %q", got)
			}
			_, _ = w.Write([]byte(`[{"client_id":"client-1","consent_request":{"subject":"user-1"}}]`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	module, err := newOryHydraModule("ory_hydra-test", map[string]any{"adminUrl": server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	issuerStep, err := createStep("step.ory_hydra_trusted_jwt_issuer_list", "issuers", map[string]any{"module": "ory_hydra-test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuerStep.Execute(context.Background(), nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	consentStep, err := createStep("step.ory_hydra_oauth2_consent_session_list", "consents", map[string]any{"module": "ory_hydra-test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := consentStep.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"subject": "user-1"}); err != nil {
		t.Fatal(err)
	}
	if !sawIssuer || !sawConsent {
		t.Fatalf("saw issuer=%v consent=%v", sawIssuer, sawConsent)
	}
}

func TestOAuth2ClientListForwardsFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/admin/clients" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if !strings.Contains(r.URL.RawQuery, "client_name=demo") {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"client_id":"client-1","client_name":"demo"}]`))
	}))
	defer server.Close()

	module, err := newOryHydraModule("ory_hydra-test", map[string]any{"adminUrl": server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_hydra_oauth2_client_list", "list", map[string]any{"module": "ory_hydra-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"client_name": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	clients := result.Output["clients"].([]any)
	if clients[0].(map[string]any)["client_id"] != "client-1" {
		t.Fatalf("output = %#v", result.Output)
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
